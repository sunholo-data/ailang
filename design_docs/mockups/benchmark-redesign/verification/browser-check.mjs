import fs from 'node:fs/promises';
import assert from 'node:assert/strict';
const tabs = await (await fetch('http://127.0.0.1:9233/json/list')).json();
const tab = tabs.find(t => t.type === 'page');
const socket = new WebSocket(tab.webSocketDebuggerUrl);
await new Promise((resolve, reject) => { socket.onopen = resolve; socket.onerror = reject; });
let seq = 0; const pending = new Map(), errors = [];
socket.onmessage = event => {
  const m = JSON.parse(event.data);
  if (m.id) { const p = pending.get(m.id); pending.delete(m.id); m.error ? p.reject(new Error(JSON.stringify(m.error))) : p.resolve(m.result); }
  else if (m.method === 'Runtime.exceptionThrown') errors.push(m.params.exceptionDetails.text + ': ' + m.params.exceptionDetails.exception?.description);
  else if (m.method === 'Runtime.consoleAPICalled' && m.params.type === 'error') console.error(JSON.stringify(m.params.args.map(a=>a.description || a.value)));
};
function call(method, params = {}) { return new Promise((resolve, reject) => { const id = ++seq; pending.set(id, { resolve, reject }); socket.send(JSON.stringify({ id, method, params })); }); }
async function js(expression) { const r = await call('Runtime.evaluate', { expression, returnByValue: true, awaitPromise: true }); if(r.exceptionDetails) throw new Error(JSON.stringify(r.exceptionDetails)); return r.result.value; }
async function until(expression) { const start = Date.now(); while (Date.now()-start<20000) { if(await js(expression)) return; await new Promise(r=>setTimeout(r,100)); } throw new Error('Timed out: '+expression); }
async function capture(name) { await new Promise(r=>setTimeout(r,300)); const {data}=await call('Page.captureScreenshot',{format:'png',captureBeyondViewport:false}); await fs.writeFile(`/private/tmp/benchmark-${name}.png`,Buffer.from(data,'base64')); }
async function scrollTo(selector) { await js(`window.scrollTo({top:document.querySelector(${JSON.stringify(selector)}).getBoundingClientRect().top+scrollY-80,behavior:'instant'})`); }
async function select(label,value) { await js(`(()=>{const s=[...document.querySelectorAll('label')].find(e=>e.firstChild?.textContent===${JSON.stringify(label)})?.querySelector('select');if(!s)throw Error('select missing');s.value=${JSON.stringify(value)};s.dispatchEvent(new Event('change',{bubbles:true}));})()`); }
async function click(text) { await js(`(()=>{const b=[...document.querySelectorAll('button')].find(e=>e.textContent===${JSON.stringify(text)});if(!b)throw Error('button missing');b.click()})()`); }
const body = '(document.body?.innerText || "")';
await call('Runtime.enable'); await call('Page.enable');
await call('Network.enable'); await call('Network.setCacheDisabled',{cacheDisabled:true});
await call('Page.addScriptToEvaluateOnNewDocument',{source:"localStorage.setItem('theme','light');"});
await call('Emulation.setDeviceMetricsOverride',{width:1440,height:1100,deviceScaleFactor:1,mobile:false});
await call('Page.navigate',{url:'http://127.0.0.1:8983/docs/benchmarks/performance'});
await until(`${body}.includes('49 / 49 runs')`);
await until(`document.querySelectorAll('.recharts-line-curve').length > 1`);
errors.length = 0; // Navigation replaced the previous test document.
const desktop = await js(`({theme:document.documentElement.dataset.theme,width:innerWidth,scroll:document.documentElement.scrollWidth,plots:document.querySelectorAll('.recharts-line-curve').length,heading:document.querySelector('h1')?.textContent})`);
assert(desktop.scroll<=desktop.width,'desktop overflow');
await capture('desktop-light');
const argon = await js(`[...document.querySelectorAll('tr')].find(e=>e.querySelector('button')?.textContent==='Gemini 4 Argon')?.innerText`);
assert.match(argon,/49 \/ 49/); assert.match(argon,/Not measured/);
const astra = await js(`[...document.querySelectorAll('tr')].find(e=>e.querySelector('button')?.textContent.endsWith('Astra'))?.innerText`);
assert.match(astra,/Provisional/);assert.match(astra,/0\/23 Core/);
await scrollTo('#compare-models'); await capture('table-desktop');
await click('Gemini 4 Argon');
await until(`${body}.includes('Final passes')`);
assert(await js(`${body}.includes('48 of 49')`));
await scrollTo('section[aria-label="Evidence for Gemini 4 Argon"]');
await capture('argon-detail'); await click('Close evidence');
await select('Order by','cost');
assert(await js(`document.querySelector('select option[value="cost"]').parentElement.value==='cost'`));
await js(`(()=>{const e=document.querySelector('input[type=search]');const setter=Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set;setter.call(e,'zz-no-model-zz');e.dispatchEvent(new Event('input',{bubbles:true}));})()`);
await until(`${body}.includes('No saved measurements match')`);
await js(`(()=>{const e=document.querySelector('input[type=search]');Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set.call(e,'');e.dispatchEvent(new Event('input',{bubbles:true}));})()`);
await until(`${body}.includes('49 / 49 runs')`);
await click('Coding agent'); await until(`${body}.includes('2 / 2 runs')`);
await click('Direct model'); await until(`${body}.includes('49 / 49 runs')`);
await select('Tasks','core'); await until(`${body}.includes('23 / 23 runs')`);
await select('Tasks','all'); await select('Order by','elo');
await js(`(()=>{const e=document.querySelector('#snapshot-inspector');Object.getOwnPropertyDescriptor(HTMLInputElement.prototype,'value').set.call(e,'0');e.dispatchEvent(new Event('input',{bubbles:true}));})()`);
await until(`document.querySelector('#snapshot-inspector').value==='0'`);
await js(`document.documentElement.setAttribute('data-theme','dark');window.scrollTo(0,0)`);
await capture('desktop-dark');
await call('Emulation.setDeviceMetricsOverride',{width:390,height:844,deviceScaleFactor:1,mobile:true});
await js('window.scrollTo(0,0)');
await new Promise(r=>setTimeout(r,250));
const mobile = await js(`({width:innerWidth,scroll:document.documentElement.scrollWidth,plots:document.querySelectorAll('.recharts-line-curve').length})`);
assert(mobile.scroll<=mobile.width, 'mobile page overflow');
await capture('mobile-dark');
await scrollTo('#compare-models'); await capture('table-mobile-dark');
await js(`document.documentElement.setAttribute('data-theme','light')`); await capture('table-mobile-light');
await js(`window.scrollTo({top:0,behavior:'instant'})`); await capture('mobile-light');
// Fail only the first request so the real helper takes its static-copy fallback.
let intercepted = 0;
const prior = socket.onmessage;
socket.onmessage = event => { const m=JSON.parse(event.data); if(m.method==='Fetch.requestPaused') {
  intercepted++; call(intercepted===1?'Fetch.failRequest':'Fetch.continueRequest',intercepted===1?{requestId:m.params.requestId,errorReason:'Failed'}:{requestId:m.params.requestId});
} else prior(event); };
await call('Fetch.enable',{patterns:[{urlPattern:'*benchmarks/latest.json*',requestStage:'Request'}]});
await call('Page.reload',{ignoreCache:true});
await until(`${body}.includes('Static fallback copy')`);
await until(`${body}.includes('49 / 49 runs')`);
await capture('fallback');
await call('Fetch.disable');
assert.equal(errors.length,0,errors.join('\n'));
const report={desktop,mobile,argon,astra,checks:['rendered plots','Argon 49/49; missing Python','Astra provisional 0/23 Core','48/49 first pass detail','cost sort','empty search','agent 2/2','Core filter','history inspector','light/dark desktop/mobile','real static fallback'],runtimeErrors:errors};
await fs.writeFile('/private/tmp/benchmark-ui-report.json',JSON.stringify(report,null,2));
console.log(JSON.stringify(report,null,2)); socket.close();
