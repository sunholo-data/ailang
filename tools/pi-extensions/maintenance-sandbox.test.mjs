/** Actual OS qualification: no provider/model calls and no sandbox mocks. */
import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { mkdtempSync,mkdirSync,readFileSync,realpathSync,rmSync,writeFileSync,statSync,symlinkSync,unlinkSync } from "node:fs";
import { tmpdir } from "node:os";
import { join,resolve,dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { spawn,spawnSync } from "node:child_process";
import { createServer } from "node:http";
import { createServer as unixServer,connect as unixConnect } from "node:net";
import { release } from "node:os";
import test from "node:test";
const here=fileURLToPath(new URL(".",import.meta.url));
const wrapper=join(here,"maintenance-sandbox.mjs");
const module=join(here,"sandbox/node_modules/@anthropic-ai/sandbox-runtime/dist/index.js");
const lock=join(here,"sandbox/package-lock.json");
const piSDK=process.env.MAINTENANCE_PI_SDK??"/opt/homebrew/lib/node_modules/@earendil-works/pi-coding-agent/dist/index.js";
const piPackage=JSON.parse(readFileSync(join(dirname(dirname(piSDK)),"package.json"),"utf8"));
assert.equal(piPackage.name,"@earendil-works/pi-coding-agent");assert.equal(piPackage.version,"0.85.1","qualify the pinned actual SDK, not a substitute");
const hash=bytes=>createHash("sha256").update(bytes).digest("hex");
function fixture(){
 const root=realpathSync(mkdtempSync(join(tmpdir(),"maintenance-os-")));
 const app=join(root,"app"),oracle=join(root,"controller");mkdirSync(app);mkdirSync(oracle);
 const secret=join(oracle,"oracle.txt");writeFileSync(secret,"PRIVATE_ORACLE_CONTROL");writeFileSync(join(app,"owned.txt"),"OWNED_PUBLIC_CONTROL");
 const policy={schema:"maintenance.containment/1",workspaceRoot:app,protectedRoots:[oracle],readOnlyRoots:[],runtimeModule:module,runtimeVersion:"0.0.71",runtimeLock:lock,runtimeLockDigest:hash(readFileSync(lock)),inferenceDomains:[]};
 const file=join(root,"policy.json");writeFileSync(file,JSON.stringify(policy));const digest=hash(readFileSync(file));
 return {root,app,oracle,secret,policy,file,digest,cleanup(){rmSync(root,{recursive:true,force:true});}};
}
async function run(f,args,network="tools"){
 return new Promise((yes,no)=>{
  const child=spawn(process.execPath,[wrapper,"--policy",f.file,"--digest",f.digest,"--network",network,"--",...args],{cwd:f.app,env:process.env,stdio:["ignore","pipe","pipe"]});
  let stdout="",stderr="";child.stdout.on("data",x=>stdout+=x);child.stderr.on("data",x=>stderr+=x);
  const timer=setTimeout(()=>{child.kill("SIGKILL");no(new Error("sandbox process exceeded30s"));},30000);
  child.once("error",e=>{clearTimeout(timer);no(e);});child.once("close",code=>{clearTimeout(timer);yes({code,stdout,stderr});});
 });
}
const python=spawnSync("which",["python3"],{encoding:"utf8"}).stdout.trim();
const ailang=spawnSync("which",["ailang"],{encoding:"utf8"}).stdout.trim();

test("OS sandbox admits normal app work, denies cat/Python and symlink/hardlink oracle access",async()=>{
 const f=fixture();try{
  assert.equal(process.platform,"darwin","this qualification record is macOS-specific; unsupported runners must not silently skip");
  const owned=await run(f,["/bin/cat",join(f.app,"owned.txt")]);assert.equal(owned.code,0,owned.stderr);assert.match(owned.stdout,/OWNED_PUBLIC_CONTROL/);
  const build=await run(f,[python,"-c","from pathlib import Path; Path('built.txt').write_text('compiled-control'); print(Path('built.txt').read_text())"]);assert.equal(build.code,0,build.stderr);
  for(const args of [["/bin/cat",f.secret],[python,"-c",`print(open(${JSON.stringify(f.secret)}).read())`]]){const result=await run(f,args);assert.notEqual(result.code,0);assert(!result.stdout.includes("PRIVATE_ORACLE_CONTROL"),JSON.stringify(result));}
  const link=join(f.app,"alias");symlinkSync(f.secret,link);const linked=await run(f,["/bin/cat",link]);assert.notEqual(linked.code,0);assert(!linked.stdout.includes("PRIVATE_ORACLE_CONTROL"));
  const hard=await run(f,[python,"-c",`import os; os.link(${JSON.stringify(f.secret)},'hard'); print(open('hard').read())`]);assert.notEqual(hard.code,0,`hardlink bypass: ${JSON.stringify(hard)}`);assert(!hard.stdout.includes("PRIVATE_ORACLE_CONTROL"));
  const write=await run(f,[python,"-c",`open(${JSON.stringify(f.secret)},'w').write('changed')`]);assert.notEqual(write.code,0);assert.equal(readFileSync(f.secret,"utf8"),"PRIVATE_ORACLE_CONTROL");
 }finally{f.cleanup();}
});

test("actual AILANG compiler succeeds, FS execution cannot read protected oracle",async()=>{
 const f=fixture();try{
  assert(ailang,"AILANG runtime required for actual qualification");
  const source=`module probe\nimport std/fs (readFile)\nexport func main() -> () ! {IO, FS} { println(readFile(${JSON.stringify(f.secret)})) }\n`;
  writeFileSync(join(f.app,"probe.ail"),source);
  const check=await run(f,[ailang,"check","probe.ail"]);assert.equal(check.code,0,check.stderr);
  const executed=await run(f,[ailang,"run","--caps","IO,FS","--entry","main","probe.ail"]);
  assert.notEqual(executed.code,0);assert(!executed.stdout.includes("PRIVATE_ORACLE_CONTROL"));assert.match(executed.stderr,/permission|denied|operation not permitted/i);
 }finally{f.cleanup();}
});

test("outer pi kernel boundary closes native read/write canonical check-use races",async()=>{
 const f=fixture();try{
  const probe=`import {readFileSync,realpathSync,unlinkSync,symlinkSync} from 'node:fs';
import {createReadTool,createWriteTool} from ${JSON.stringify(new URL('file://'+piSDK).href)};
const owned=${JSON.stringify(join(f.app,'owned.txt'))},secret=${JSON.stringify(f.secret)};
const admitted=realpathSync(owned);unlinkSync(owned);symlinkSync(secret,owned);
let readDenied=false,writeDenied=false;
try{const r=await createReadTool(process.cwd()).execute('read-race',{path:admitted},new AbortController().signal);readDenied=!JSON.stringify(r).includes('PRIVATE_ORACLE_CONTROL')&&r.isError===true;}catch{readDenied=true;}
try{const r=await createWriteTool(process.cwd()).execute('write-race',{path:admitted,content:'changed'},new AbortController().signal);writeDenied=r.isError===true;}catch{writeDenied=true;}
if(!readDenied||!writeDenied)throw Error('native race escaped kernel boundary');console.log('NATIVE_RACES_DENIED');`;
  writeFileSync(join(f.app,"native-probe.mjs"),probe);
  const result=await run(f,[process.execPath,join(f.app,"native-probe.mjs")],"inference");assert.equal(result.code,0,result.stderr);assert.match(result.stdout,/NATIVE_RACES_DENIED/);assert.equal(readFileSync(f.secret,"utf8"),"PRIVATE_ORACLE_CONTROL");
 }finally{f.cleanup();}
});

test("tool subprocess network cannot reach an independently live local server",async()=>{
 const f=fixture();let hits=0;
 const server=createServer((_req,res)=>{hits++;res.end("NETWORK_REFERENCE_CONTROL");});
 await new Promise(resolvePromise=>server.listen(0,"127.0.0.1",resolvePromise));
 try{
  const port=server.address().port;
  assert.equal(await (await fetch(`http://127.0.0.1:${port}`)).text(),"NETWORK_REFERENCE_CONTROL");assert.equal(hits,1);
  const blocked=await run(f,[python,"-c",`import urllib.request; print(urllib.request.urlopen('http://127.0.0.1:${port}',timeout=3).read())`]);
  assert.notEqual(blocked.code,0);assert(!blocked.stdout.includes("NETWORK_REFERENCE_CONTROL"));assert.equal(hits,1,"sandbox request reached reference server");
 }finally{await new Promise(resolvePromise=>server.close(resolvePromise));f.cleanup();}
});

test("changed policy/runtime lock fails before child execution",async()=>{
 const f=fixture();try{
  f.digest="0".repeat(64);const blocked=await run(f,[process.execPath,"-e","console.log('CHILD_STARTED')"]);assert.equal(blocked.code,125);assert(!blocked.stdout.includes("CHILD_STARTED"));
 }finally{f.cleanup();}
});

let nestedBlocker;
test("known macOS nested SDK blocker keeps protected backend admission CLOSED",async()=>{
 const f=fixture();try{
  const probe=`import {spawn} from 'node:child_process';
const child=spawn(process.execPath,${JSON.stringify([wrapper,'--policy',f.file,'--digest',f.digest,'--network','tools','--',python,'-c',"from pathlib import Path; Path('nested-built.txt').write_text('NESTED_TOOL_CONTROL'); print(Path('nested-built.txt').read_text())"])},{cwd:process.cwd(),env:process.env,stdio:'inherit'});
child.on('error',e=>{throw e});child.on('close',code=>process.exitCode=code??1);`;
  writeFileSync(join(f.app,"nested-probe.mjs"),probe);
  const result=await run(f,[process.execPath,join(f.app,"nested-probe.mjs")],"inference");assert.equal(result.code,71,`unexpected nested outcome: ${JSON.stringify(result)}`);assert.match(result.stderr,/sandbox_apply: Operation not permitted/);assert(!result.stdout.includes("NESTED_TOOL_CONTROL"));nestedBlocker=result;
 }finally{f.cleanup();}
});

test("tool process cannot bypass network policy through an allowed outer Unix socket",async()=>{
 const f=fixture(),short=realpathSync(mkdtempSync("/private/tmp/am-unix-"));let hits=0;
 const socket=join(short,"reference.sock");const server=unixServer(connection=>{hits++;connection.end("UNIX_REFERENCE_CONTROL");});
 await new Promise(yes=>server.listen(socket,yes));
 try{
  const observed=await new Promise((yes,no)=>{let text="";const c=unixConnect(socket);c.on("data",x=>text+=x);c.once("end",()=>yes(text));c.once("error",no);});assert.equal(observed,"UNIX_REFERENCE_CONTROL");assert.equal(hits,1);
  const result=await run(f,[python,"-c",`import socket; s=socket.socket(socket.AF_UNIX); s.connect(${JSON.stringify(socket)}); print(s.recv(100))`]);assert.notEqual(result.code,0);assert(!result.stdout.includes("UNIX_REFERENCE_CONTROL"));assert.equal(hits,1);
 }finally{await new Promise(yes=>server.close(yes));rmSync(short,{recursive:true,force:true});f.cleanup();}
});

test.after(()=>{
 if(process.env.MAINTENANCE_CONTAINMENT_REPORT){
  const report={schema:"maintenance.containment-qualification/1",status:"blocked",backendAdmitted:false,
   platform:process.platform,kernel:release(),node:process.version,piPackage:"@earendil-works/pi-coding-agent",piVersion:piPackage.version,
   runtimePackage:"@anthropic-ai/sandbox-runtime",runtimeVersion:"0.0.71",runtimeLockDigest:hash(readFileSync(lock)),wrapperDigest:hash(readFileSync(wrapper)),
   primitiveChecks:["owned-app-build","cat-python-read-denial","static-symlink-hardlink-denial","protected-write-denial","compiled-ailang-fs-denial","actual-native-sdk-read-write-races","tcp-reference-denial","unix-reference-denial","changed-policy-denial"],
   nestedBlocker,sourceBoundary:"temporary declared controller/oracle root; full known-source-copy and Git-store inventory remains a separate live-admission requirement",qualifiedClaims:"direct host candidate primitives only; protected pi backend NOT qualified",modelCalls:0};
  writeFileSync(process.env.MAINTENANCE_CONTAINMENT_REPORT,JSON.stringify(report,null,2)+"\n",{flag:"wx"});
 }
});

test("wrapper entry through a symlink path must actually execute its child",async()=>{
 const f=fixture();try{
  const alias=join(f.root,"wrapper-alias.mjs");symlinkSync(wrapper,alias);
  const result=await new Promise((yes,no)=>{
   const child=spawn(process.execPath,[alias,"--policy",f.file,"--digest",f.digest,"--network","tools","--","/bin/cat",join(f.app,"owned.txt")],{cwd:f.app,stdio:["ignore","pipe","pipe"]});
   let stdout="",stderr="";child.stdout.on("data",x=>stdout+=x);child.stderr.on("data",x=>stderr+=x);child.once("error",no);child.once("close",code=>yes({code,stdout,stderr}));
  });
  assert.equal(result.code,0,result.stderr);assert.match(result.stdout,/OWNED_PUBLIC_CONTROL/,"zero exit with empty output is not execution");
 }finally{f.cleanup();}
});
