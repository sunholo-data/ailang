(() => {
  'use strict';
  const db=window.BENCHMARK_PREVIEW;
  const esc=x=>String(x??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
  const pct=v=>v==null?'Not retained':`${(v*100).toFixed(1)}%`;
  const finite=v=>typeof v==='number'&&Number.isFinite(v);
  const history=db.overallHistory||[];
  const label=id=>db.standard.find(r=>r.id===id)?.name||id.replaceAll('-',' ');
  const available=[...new Set(history.flatMap(h=>Object.keys(h.elo||{})))].sort();
  function project(records,scope){return records.map(h=>({
    ...h,
    a:finite(h.scopes?.[scope]?.ailang?.rate)&&h.scopes[scope].ailang.runs>0?h.scopes[scope].ailang.rate*100:null,
    p:finite(h.scopes?.[scope]?.python?.rate)&&h.scopes[scope].python.runs>0?h.scopes[scope].python.rate*100:null,
  }));}
  function segments(points,values,x,y,anchored=false){
    const paths=[];let path='',previousAnchor=null;
    values.forEach((v,i)=>{
      if(!finite(v)){if(path)paths.push(path);path='';previousAnchor=null;return;}
      const anchor=points[i].anchor;
      if(anchored&&path&&anchor!==previousAnchor){paths.push(path);path='';}
      path+=`${path?'L':'M'}${x(i).toFixed(2)},${y(v).toFixed(2)} `;previousAnchor=anchor;
    });
    if(path)paths.push(path);return paths;
  }
  window.BenchmarkTrendUtils={project,segments};
  window.BenchmarkTrends={mount(id){
    const root=document.getElementById(id);if(!root||!history.length)return;
    root.hidden=false;
    const state={scope:'core',range:'all',selected:history.length-1,models:['gemini-3-1-pro','gpt5-6-sol','or-glm-5-2','claude-opus-5-5'].filter(m=>available.includes(m))};
    root.innerHTML=`<div class="overall-head"><div><h2>Overall trends</h2><p>Saved direct-model results across AILANG releases</p></div><div class="overall-controls"><label>Pass-rate tasks<select id="trend-scope"><option value="core">Core</option><option value="all">All measured tasks</option><option value="stretch">Stretch</option><option value="frontier">Frontier</option></select></label><label>History<select id="trend-range"><option value="all">Full history</option><option value="recent">Recent 20 snapshots</option></select></label></div></div><div class="overall-grid"><section class="overall-plot"><h3>AILANG and Python pass rates</h3><p>Percentage of saved runs that passed</p><div id="overall-pass-chart"></div><div class="overall-legend"><span><i class="line-key key-ailang"></i>AILANG</span><span><i class="line-key key-python"></i>Python</span></div></section><section class="overall-plot"><h3>Model ELO over releases</h3><p>All-task ratings; each model identity stays separate</p><div id="overall-elo-chart"></div><div id="overall-elo-legend" class="overall-legend"></div></section></div><div class="snapshot-inspector"><label for="trend-snapshot">Inspect a saved release snapshot</label><input id="trend-snapshot" type="range" min="0" max="${history.length-1}" value="${history.length-1}"><div id="overall-snapshot-detail" aria-live="polite"></div></div><div class="overall-foot"><p>Historical model and task mixes vary. A snapshot may reuse older runs; it does not mean every model was tested for that release. Pass-rate task filters do not change the all-task ELO fit.</p><details><summary>Choose ELO models</summary><div id="elo-model-choices" class="elo-model-choices">${available.map(m=>`<label><input type="checkbox" value="${esc(m)}" ${state.models.includes(m)?'checked':''}>${esc(label(m))}</label>`).join('')}</div></details></div>`;
    const el=id=>document.getElementById(id);
    function points(){const all=project(history,state.scope);return state.range==='recent'?all.slice(-20):all;}
    function chart(target,points,series,domain,title){
      const width=Math.max(260,Math.floor(target.getBoundingClientRect().width||450)),height=225;
      const left=48,right=15,top=18,bottom=34,w=width-left-right,h=height-top-bottom;
      const x=i=>left+i*w/Math.max(1,points.length-1),y=v=>top+(domain[1]-v)/(domain[1]-domain[0])*h;
      const selected=Math.min(state.selected,points.length-1),sx=x(selected);
      const ticks=[domain[0],(domain[0]+domain[1])/2,domain[1]];
      const xi=[...new Set([0,Math.floor((points.length-1)/2),points.length-1])];
      const format=v=>Math.round(v)+(title==='Pass rate'?'%':'');
      target.innerHTML=`<svg viewBox="0 0 ${width} ${height}" width="${width}" height="${height}" role="img" aria-label="${esc(title)} across ${points.length} saved release snapshots"><title>${esc(title)}</title><desc>Missing values break the line. Use the snapshot slider below to inspect exact values and counts.</desc>${ticks.map(v=>`<line class="overall-axis" x1="${left}" x2="${width-right}" y1="${y(v)}" y2="${y(v)}"/><text x="${left-9}" y="${y(v)+4}" text-anchor="end">${format(v)}</text>`).join('')}${xi.map(i=>`<text x="${x(i)}" y="${height-10}" text-anchor="${i===0?'start':i===points.length-1?'end':'middle'}">${esc(points[i]?.version||'')}</text>`).join('')}<line class="snapshot-guide" x1="${sx}" x2="${sx}" y1="${top}" y2="${height-bottom}"/>${series.map(s=>{const values=points.map(s.value);return segments(points,values,x,y,s.anchored).map(d=>`<path class="overall-series ${s.cls}" d="${d}"/>`).join('')+values.map((v,i)=>finite(v)?`<circle class="overall-dot ${s.cls}" cx="${x(i)}" cy="${y(v)}" r="${i===selected?4:2}"><title>${esc(points[i].version)} · ${esc(s.name)}: ${format(v)}</title></circle>`:'').join('');}).join('')}<rect class="trend-hit" x="${left}" y="${top}" width="${w}" height="${h}" fill="transparent"/></svg>`;
      const svg=target.querySelector('svg');
      function inspect(event){
        const box=svg.getBoundingClientRect();
        const px=(event.clientX-box.left)*width/box.width;
        state.selected=Math.max(0,Math.min(points.length-1,Math.round((px-left)/w*(points.length-1))));
        el('trend-snapshot').value=state.selected;draw(false);
      }
      svg.addEventListener('pointermove',inspect);svg.addEventListener('click',inspect);
    }
    function draw(redraw=true){
      const ps=points();state.selected=Math.max(0,Math.min(state.selected,ps.length-1));
      const s=ps[state.selected];if(!s)return;
      const chosen=state.models.map((id,i)=>({id,name:label(id),value:h=>finite(h.elo?.[id])?h.elo[id]:null,cls:`elo-series-${i%6}`,anchored:true}));
      if(redraw){
        chart(el('overall-pass-chart'),ps,[{name:'AILANG',value:h=>h.a,cls:'pass-ailang'},{name:'Python',value:h=>h.p,cls:'pass-python'}],[0,100],'Pass rate');
        const vals=chosen.flatMap(m=>ps.map(m.value)).filter(finite);
        const low=vals.length?Math.floor((Math.min(...vals)-60)/100)*100:1000;
        const high=vals.length?Math.ceil((Math.max(...vals)+60)/100)*100:3000;
        chart(el('overall-elo-chart'),ps,chosen,[low,Math.max(low+100,high)],'Model ELO');
      }else{
        ['overall-pass-chart','overall-elo-chart'].forEach(id=>{
          const svg=el(id).querySelector('svg');if(!svg)return;
          const width=Number(svg.getAttribute('width')),x=48+state.selected*(width-63)/Math.max(1,ps.length-1);
          const guide=svg.querySelector('.snapshot-guide');guide.setAttribute('x1',x);guide.setAttribute('x2',x);
        });
      }
      el('trend-snapshot').max=ps.length-1;el('trend-snapshot').value=state.selected;
      el('overall-elo-legend').innerHTML=chosen.map(m=>`<span><i class="line-key ${m.cls}"></i>${esc(m.name)}</span>`).join('');
      const scope=s.scopes[state.scope]||{};
      el('overall-snapshot-detail').innerHTML=`<strong>${esc(s.version)}</strong><span>Published ${esc(s.date)}</span><span>AILANG <b>${pct(s.a==null?null:s.a/100)}</b> <small>${scope.ailang?.runs||0} runs</small></span><span>Python <b>${pct(s.p==null?null:s.p/100)}</b> <small>${scope.python?.runs||0} runs</small></span><span>${s.models} model records</span><div class="snapshot-elo-values">${chosen.map(m=>`<span>${esc(m.name)} <b>${finite(s.elo[m.id])?Math.round(s.elo[m.id]):'Not in snapshot'}</b></span>`).join('')}</div>`;
    }
    el('trend-scope').onchange=e=>{state.scope=e.target.value;draw();};
    el('trend-range').onchange=e=>{state.range=e.target.value;state.selected=points().length-1;draw();};
    el('trend-snapshot').oninput=e=>{state.selected=Number(e.target.value);draw(false);};
    el('elo-model-choices').addEventListener('change',e=>{
      const selected=[...el('elo-model-choices').querySelectorAll('input:checked')];
      if(selected.length>6){e.target.checked=false;return;}
      state.models=selected.map(c=>c.value);draw();
    });
    draw();
    if(typeof ResizeObserver!=='undefined')new ResizeObserver(()=>draw()).observe(root);
  }};
})();
