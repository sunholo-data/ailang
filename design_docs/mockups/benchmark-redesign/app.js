(() => {
  'use strict';
  const db = window.BENCHMARK_PREVIEW;
  const q = new URLSearchParams(location.search);
  const view = ['table','studio','atlas'].includes(q.get('view')) ? q.get('view') : 'table';
  const state = {mode:'standard',scope:'all',lang:'ailang',sort:'elo',search:'',section:'models',selected:null,progressModel:'gpt6-sol'};
  const $ = id => document.getElementById(id);
  const esc = x => String(x ?? '').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
  const percent = r => r == null ? '—' : `${(r*100).toFixed(1)}%`;
  const money = v => v == null ? '—' : v === 0 ? '$0 recorded' : `$${v.toFixed(v<.01?4:3)}`;
  const time = v => !v ? '—' : v < 1000 ? `${Math.round(v)} ms` : v < 60000 ? `${(v/1000).toFixed(1)} s` : `${(v/60000).toFixed(1)} min`;
  const scopeLabel = t => ({all:'All measured tasks',core:'Core',stretch:'Stretch',frontier:'Frontier',smoke:'Smoke'})[t] || t;
  const measurement = (r,scope=state.scope,lang=state.lang) => r.scopes[scope]?.[lang];
  const metrics = r => r.metrics[state.scope]?.[state.lang] || {};
  const rating = r => r.ratings[state.lang]?.elo;
  const undefeated = r => { const s=measurement(r,'all');return s?.runs>0&&s.pass===s.runs; };
  const rankGroup = r => rating(r)==null&&undefeated(r)?0:rating(r)!=null?1:2;
  const rowLabel = r => r.name;
  const title = {table:'Models, measured.',studio:'Find your AILANG model.',atlas:'See the whole benchmark picture.'};
  document.body.dataset.view=view;
  document.body.dataset.theme = localStorage.getItem('benchmark-preview-theme') || (view==='studio'?'light':'dark');
  document.querySelector(`.reviewbar a[href="?view=${view}"]`).classList.add('current');
  $('candidate-name').textContent={table:'A · Evidence table',studio:'B · Model studio',atlas:'C · Benchmark atlas'}[view];
  $('title').textContent=title[view];
  $('intro').textContent={table:'ELO, task success, cost and measurement history. One comparison, with the gaps left visible.',studio:'Start with the model. Open the evidence when you want to know more.',atlas:'Compare model capability, see which tasks still discriminate, and keep every saved run in view.'}[view];
  function eloHTML(r){
    const v=rating(r);if(v!=null)return `<span class="elo-value">${Math.round(v)}</span><small>${state.lang==='ailang'?'AILANG':'Python'} ELO</small>`;
    const s=measurement(r,'all');
    return undefeated(r) ? '<span class="undefeated">Undefeated</span><small>No losses to calibrate ELO</small>' : '<span class="absent">—</span><small>ELO unavailable</small>';
  }
  function scoreHTML(s,python=false){
    if(!s)return '<span class="absent">—</span><small>Not measured</small>';
    return `<span class="rate">${percent(s.rate)}</span><small>${s.pass} / ${s.runs} runs</small><div class="measure"><span style="width:${s.rate*100}%"></span></div>`;
  }
  function evidence(r){const version=r.versions[state.lang];return `<small>${esc(r.dates?.[state.lang]||r.date||'Date unavailable')}</small><small>${esc(version||'Version unavailable')}</small>`;}
  function sample(r){
    const s=measurement(r,'all'),c=r.coreCoverage?.[state.lang];
    if(undefeated(r)&&((c&&c.done<c.required)||s.runs<10))return `<span class="badge">Provisional · undefeated</span><small>${s.runs} recorded runs${c?` · ${c.done}/${c.required} Core tasks`:''}</small>`;
    return s&&s.runs<10?'<span class="badge">Limited sample</span>':'';
  }
  function rows(){
    let rs=db[state.mode].filter(r=>(view==='atlas'?(measurement(r,'all')||measurement(r,'all','python')):(measurement(r)||measurement(r,state.scope,'python')))&&`${r.name} ${r.id} ${r.harness}`.toLowerCase().includes(state.search.toLowerCase()));
    return rs.sort((a,b)=>{
      if(state.sort==='elo'){
        const ag=rankGroup(a),bg=rankGroup(b);if(ag!==bg)return ag-bg;
        if(ag===0){
          const ac=a.coreCoverage?.[state.lang],bc=b.coreCoverage?.[state.lang];
          const af=!!ac&&ac.done===ac.required,bf=!!bc&&bc.done===bc.required;
          if(af!==bf)return af?-1:1;
          const an=ac?.benchmarks||measurement(a,'all')?.runs||0,bn=bc?.benchmarks||measurement(b,'all')?.runs||0;
          if(an!==bn)return bn-an;
          return a.name.localeCompare(b.name);
        }
      }
      const av=state.sort==='elo'?rating(a):state.sort==='rate'?measurement(a)?.rate:state.sort==='cost'?metrics(a).cost:null;
      const bv=state.sort==='elo'?rating(b):state.sort==='rate'?measurement(b)?.rate:state.sort==='cost'?metrics(b).cost:null;
      if(state.sort!=='name'){if(av==null&&bv!=null)return 1;if(bv==null&&av!=null)return -1;if(av!=null&&bv!=null&&av!==bv)return state.sort==='cost'?av-bv:bv-av;}
      return a.name.localeCompare(b.name);
    });
  }
  function table(rs){return `<div class="table-wrap"><table><thead><tr><th>Model ${state.mode==='agent'?'& harness':''}</th><th>ELO / status</th><th>${state.lang==='ailang'?'AILANG':'Python'} passed</th>${state.lang==='ailang'?'<th>Python reference</th>':''}<th>Cost / solved task</th><th>Recorded time</th><th>Measured on</th></tr></thead><tbody>${rs.map((r,i)=>{
    const group=rankGroup(r),heading=state.sort==='elo'&&(i===0||rankGroup(rs[i-1])!==group)?`<tr class="ranking-group"><th colspan="${state.lang==='ailang'?7:6}" scope="colgroup">${['Undefeated on measured tasks · complete Core coverage first, smaller samples provisional','Rated models · highest ELO first','Rating unavailable'][group]}</th></tr>`:'';
    return heading+`<tr class="${r.id===state.selected?'selected-row':''}"><td class="model-cell"><button class="model-button" data-model="${esc(r.id)}">${esc(rowLabel(r))}</button><small>${esc(r.harness)}</small>${sample(r)}</td><td>${eloHTML(r)}</td><td>${scoreHTML(measurement(r))}</td>${state.lang==='ailang'?`<td class="python">${scoreHTML(measurement(r,state.scope,'python'),true)}</td>`:''}<td>${money(metrics(r).cost)}<small>${metrics(r).cost==null?'Unavailable':`per ${state.lang==='ailang'?'AILANG':'Python'} success`}</small></td><td>${time(metrics(r).time)}<small>${metrics(r).time?'saved duration field':'Timing unavailable'}</small></td><td>${evidence(r)}</td></tr>`;
  }).join('')}</tbody></table></div>`;}
  function studio(rs){return `<div class="studio-list">${rs.map(r=>{
    const a=measurement(r),p=measurement(r,state.scope,'python'),m=metrics(r);
    return `<article class="studio-row"><div><button class="model-button" data-model="${esc(r.id)}">${esc(r.name)}</button><small>${esc(r.harness)}</small>${sample(r)}<small>${esc(r.date||'Date unavailable')} ${esc(r.versions[state.lang]||'')}</small></div><div><div class="barline"><span>${state.lang==='ailang'?'AILANG':'Python'}</span><b>${a?`${percent(a.rate)} · ${a.pass}/${a.runs}`:'Not measured'}</b></div><div class="studio-meter"><span style="width:${(a?.rate||0)*100}%"></span></div>${state.lang==='ailang'?`<div class="barline"><span>Python reference</span><b>${p?`${percent(p.rate)} · ${p.pass}/${p.runs}`:'Not measured'}</b></div><div class="studio-meter python"><span style="width:${(p?.rate||0)*100}%"></span></div>`:''}</div><div class="studio-value"><div>${eloHTML(r)}</div><div><b>${money(m.cost)}</b><span>per solved task</span></div><div><b>${time(m.time)}</b><span>recorded duration</span></div></div><button data-model="${esc(r.id)}" type="button">Evidence</button></article>`;
  }).join('')}</div>`;}
  function atlas(rs){
    const scopes=state.mode==='standard'?['smoke','core','stretch','frontier']:['all'];
    return `<div class="atlas-intro"><b>Task coverage is part of the result.</b> A 100% on basic checks means something different from a 100% on frontier tasks. Empty cells remain visible.</div><div class="table-wrap"><table><thead><tr><th>Model & evidence</th><th>ELO</th>${scopes.map(t=>`<th>${scopeLabel(t)}</th>`).join('')}<th>Cost / success<small>Selected task group</small></th></tr></thead><tbody>${rs.map(r=>`<tr><td class="model-cell"><button class="model-button" data-model="${esc(r.id)}">${esc(r.name)}</button><small>${esc(r.harness)}</small>${sample(r)}${evidence(r)}</td><td class="matrix-elo">${eloHTML(r)}</td>${scopes.map(t=>{const s=measurement(r,t);return `<td class="matrix-cell"><button type="button" data-model="${esc(r.id)}" class="${s?'has-data':''}" style="--strength:${s?.rate||0}"><strong>${s?percent(s.rate):'—'}</strong><small>${s?`${s.pass}/${s.runs} runs`:'Not measured'}</small></button></td>`;}).join('')}<td>${money(metrics(r).cost)}</td></tr>`).join('')}</tbody></table></div>`;
  }
  function render(){
    $('scope').disabled=state.mode==='agent';
    $('mode-description').textContent=state.mode==='standard'?'Direct model = API generation with the recorded self-repair policy. Pass rates use the selected tasks. ELO always uses all available tasks in this mode and language.':'Coding agent = model plus its named harness and multi-turn tools. Saved aggregate agent data is shown across all tasks; per-model tier detail is not available in this preview projection.';
    $('scope-note').textContent=state.scope==='all'?'Different models have different measured task sets. These are descriptive results, not a matched-task language delta. Click a model to inspect its evidence.':`Pass rate and cost use ${scopeLabel(state.scope)} evidence. ELO uses all measured tasks. Models without ${scopeLabel(state.scope)} measurements are shown with unavailable cells in the atlas.`;
    const rs=rows();
    $('result-count').textContent=`${rs.length} ${state.mode==='agent'?'model + harness configurations':'models'} · ${scopeLabel(state.scope)}`;
    $('results-title').textContent=view==='atlas'?'Models × task difficulty':'Compare models';
    $('results').innerHTML=rs.length?({table,studio,atlas}[view])(rs):'<div class="empty">No saved measurements match this view. Try another language or task group.</div>';
    document.querySelectorAll('[data-mode]').forEach(b=>b.setAttribute('aria-pressed',b.dataset.mode===state.mode));
    if(state.selected)showDetail(state.selected,false);
  }
  function showDetail(id,scroll=true){
    const r=db[state.mode].find(x=>x.id===id);if(!r){$('detail').hidden=true;return;}
    state.selected=id;const s=measurement(r),m=metrics(r);const all=measurement(r,'all');
    $('detail').hidden=false;
    $('detail').innerHTML=`<div class="detail-top"><div><h3>${esc(r.name)}</h3><p class="detail-intro">${esc(r.harness)} · ${scopeLabel(state.scope)} · ${state.lang==='ailang'?'AILANG':'Python'}</p></div><button id="close-detail" type="button">Close evidence</button></div><div class="detail-grid"><p>Passed after allowed repair<strong>${s?`${s.pass} of ${s.runs}`:'Not measured'}</strong></p><p>Passed first attempt<strong>${m.first!=null&&s?`${m.first} of ${s.runs}`:'Not retained'}</strong></p><p>Cost per solved task<strong>${money(m.cost)}</strong></p><p>All-task ELO<strong>${rating(r)!=null?Math.round(rating(r)):all?.rate===1?'Undefeated':'Unavailable'}</strong></p></div><p class="detail-footer">Measured ${esc(r.date||'date unavailable')} on ${esc(r.versions[state.lang]||'version unavailable')}. Reused in the latest evidence view. ${s&&s.runs<10?'Limited sample: fewer than 10 runs. ':''}Cost values come from ${esc(m.source||'an unavailable saved aggregate')}; task and language counts must match before a cost is shown. Recorded duration is the saved duration field; its timing boundary needs review before it is used as an end-to-end speed claim.</p>`;
    const extra=r.detail||{},eff=extra.efficiency||{},agg=extra.aggregates||{};
    $('detail').insertAdjacentHTML('beforeend',`<details class="method-short"><summary>More recorded metrics & outcomes</summary><div class="detail-grid"><p>P90 cost per success<strong>${money(eff.p90_cost_per_success)}</strong></p><p>Median turns to success<strong>${eff.median_turns_to_success??'—'}</strong></p><p>Recorded tokens / second<strong>${eff.median_tokens_per_sec>0?eff.median_tokens_per_sec.toFixed(1):'Not retained'}</strong></p><p>Repairs used<strong>${agg.repairUsed??'Not retained'}</strong></p></div><p class="detail-footer">These additional metrics describe the model's complete recorded task mix in this mode. They do not inherit the task filter above.</p><div class="outcome-list">${Object.entries(extra.outcomes||{}).map(([k,v])=>`<span>${esc(k.replaceAll('_',' '))}<b>${v}</b></span>`).join('')}</div><a class="download" href="evidence.json" download>Download the full published evidence</a></details>`);
    $('close-detail').onclick=()=>{state.selected=null;$('detail').hidden=true;};
    if(scroll)$('detail').scrollIntoView({behavior:matchMedia('(prefers-reduced-motion: reduce)').matches?'instant':'smooth',block:'nearest'});
  }
  function progress(){
    const choices=db.standard.filter(r=>db.history.some(h=>h.elo[r.id]!=null));
    const points=db.history.filter(h=>h.elo[state.progressModel]!=null).slice(-14);
    const ys=points.map(h=>h.elo[state.progressModel]);const low=Math.floor((Math.min(...ys)-20)/10)*10,high=Math.ceil((Math.max(...ys)+20)/10)*10;
    const xy=points.map((h,i)=>[64+i*(700/Math.max(1,points.length-1)),195-(h.elo[state.progressModel]-low)/(high-low)*150]);
    const ticks=[low,(low+high)/2,high];
    $('progress-panel').innerHTML=`<div class="section-head"><h2>Keep the evidence. Track the changes.</h2><p>A release is a code milestone, not a requirement to rerun every model. The rolling view reuses saved runs and records which version produced them.</p></div><div class="chart-head"><div><h3>Historical ELO snapshots</h3><p>Recorded combined-language ratings · same anchor where available · no fabricated upward trend</p></div><label>Model<select id="progress-model">${choices.map(r=>`<option value="${esc(r.id)}" ${r.id===state.progressModel?'selected':''}>${esc(r.name)}</option>`).join('')}</select></label></div><div class="chart-surface"><svg viewBox="0 0 800 245" role="img" aria-label="Historical recorded ELO snapshots for ${esc(state.progressModel)}">${ticks.map(v=>{const y=195-(v-low)/(high-low)*150;return `<line class="axis" x1="64" x2="764" y1="${y}" y2="${y}"/><text x="52" y="${y+4}" text-anchor="end">${Math.round(v)}</text>`;}).join('')}<path class="trend-line" d="${xy.map(([x,y],i)=>`${i?'L':'M'}${x},${y}`).join(' ')}"/>${xy.map(([x,y],i)=>`<circle class="trend-dot" cx="${x}" cy="${y}" r="4"><title>${esc(points[i].version)}: ${Math.round(ys[i])} ELO</title></circle>`).join('')}<text x="64" y="228">${esc(points[0]?.version||'')}</text><text x="764" y="228" text-anchor="end">${esc(points.at(-1)?.version||'')}</text></svg></div><div class="progress-note"><strong>A changing leaderboard is not automatically language progress.</strong> Model upgrades, new tasks and newly imported results can move these snapshots. A separate fixed-model, fixed-task panel is the place to claim that AILANG itself got easier. No panel result is invented here.</div><div class="history-list"><table><thead><tr><th>Saved snapshot</th><th>Published</th><th>Recorded ELO</th><th>Anchor</th></tr></thead><tbody>${points.slice(-6).reverse().map(h=>`<tr><td>${esc(h.version)}</td><td>${esc(h.date)}</td><td>${Math.round(h.elo[state.progressModel])}</td><td>${esc(h.anchor||'Unavailable')}</td></tr>`).join('')}</tbody></table></div>`;
    $('progress-panel').insertAdjacentHTML('beforeend',`<details class="method-short"><summary>Browse all ${db.releases.length} saved release snapshots</summary><p>Publication snapshots preserve the evidence assembled at that point. They do not mean every model was remeasured for that release.</p><div class="history-list table-wrap"><table><thead><tr><th>Release snapshot</th><th>Published</th><th>Recorded standard runs</th><th>ELO retained</th></tr></thead><tbody>${db.releases.map(h=>`<tr><td>${esc(h.version)}</td><td>${esc(h.published)}</td><td>${h.runs}</td><td>${h.rated?'Yes':'Not retained'}</td></tr>`).join('')}</tbody></table></div><a class="download" href="evidence.json" download>Download full snapshots and per-model records</a></details>`);
    $('progress-model').onchange=e=>{state.progressModel=e.target.value;progress();};
  }
  function tasks(){
    const filtered=db.tasks.filter(t=>!state.search||t.name.includes(state.search.toLowerCase()));
    $('tasks-panel').innerHTML=`<div class="section-head"><h2>What are the models being asked to do?</h2><p>${db.tasks.length} published task records. Smoke checks the basics, Core covers everyday tasks, Stretch and Frontier expose more headroom. Harder tasks can be added later without discarding the existing evidence.</p></div><div class="task-list">${filtered.map(t=>`<div class="task-item"><a href="https://ailang.sunholo.com/docs/benchmarks/gallery#${encodeURIComponent(t.id)}">${esc(t.name)}</a><span>${esc(t.tier||'Core')}</span></div>`).join('')}</div>`;
  }
  function method(){
    $('method-panel').innerHTML=`<div class="section-head"><h2>Every number has a measurement behind it.</h2><p>This design preview uses saved published results. It does not run models, refresh benchmarks, or change the production site.</p></div><div class="explanation-grid"><div><h3>Latest evidence, not latest release</h3><p>Keep the latest saved observation for each model, language, task, mode and trial. Preserve its actual run date and measured AILANG version. The publication date only tells you when this view was assembled.</p></div><div><h3>ELO stays central</h3><p>ELO uses task difficulty to separate models when pass rates saturate. An undefeated model stays visible with an explicit unavailable finite rating; no made-up number or rank.</p></div><div><h3>Cost per solved task</h3><p>Saved spending, including failed attempts, divided by saved successes in the same scope. No inferred zero-dollar run when cost was not retained. Mode, language and selected tasks remain explicit.</p></div><div><h3>Speed needs a clear boundary</h3><p>The prototype exposes saved duration fields for review. Some imported records retain execution time rather than complete generation latency, so they must not be promoted as end-to-end speed without provenance.</p></div><div><h3>All data remains reachable</h3><p>The comparison is the front door. Individual models open first-attempt results, final results, costs and provenance. Progress and task views retain the research detail without repeating it above the table.</p></div><div><h3>Historical data is not rewritten</h3><p>Release snapshots remain inspectable. Comparable trend cohorts must name their task set, model panel and anchor. New evidence is appended; frequent releases do not mandate new API spending.</p></div></div><a class="download" href="evidence.json" download>Download the full published dataset</a>`;
  }
  function section(name){state.section=name;['models','progress','tasks','method'].forEach(s=>{$(`${s}-panel`).hidden=s!==name;document.querySelector(`[data-section="${s}"]`).setAttribute('aria-pressed',s===name);});if(name==='progress')progress();if(name==='tasks')tasks();if(name==='method')method();}
  document.addEventListener('click',e=>{const b=e.target.closest('[data-model]');if(b)showDetail(b.dataset.model);const tab=e.target.closest('[data-section]');if(tab)section(tab.dataset.section);const link=e.target.closest('[data-page]');if(link){e.preventDefault();section(link.dataset.page);}const mode=e.target.closest('[data-mode]');if(mode){state.mode=mode.dataset.mode;state.scope='all';$('scope').value='all';state.selected=null;$('detail').hidden=true;render();}});
  $('scope').onchange=e=>{state.scope=e.target.value;render();};$('language').onchange=e=>{state.lang=e.target.value;render();};$('sort').onchange=e=>{state.sort=e.target.value;render();};$('search').oninput=e=>{state.search=e.target.value;render();};
  $('theme').onclick=()=>{document.body.dataset.theme=document.body.dataset.theme==='dark'?'light':'dark';localStorage.setItem('benchmark-preview-theme',document.body.dataset.theme);};
  document.querySelector('.sidebar>a').onclick=e=>{e.preventDefault();section('models');};
  render();
  if(view==='table' && window.BenchmarkTrends)window.BenchmarkTrends.mount('overview-trends');
})();
