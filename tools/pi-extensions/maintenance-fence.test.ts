import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { mkdtempSync, mkdirSync, readFileSync, realpathSync, rmSync, symlinkSync, writeFileSync } from "node:fs";
import { registerHooks } from "node:module";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { pathToFileURL } from "node:url";
import test from "node:test";

// Resolve to the actual installed SDK, never a stub. Override supports CI's
// pinned installation; absence fails this qualification test instead of skipping.
const sdkPath = process.env.MAINTENANCE_PI_SDK ?? "/opt/homebrew/lib/node_modules/@earendil-works/pi-coding-agent/dist/index.js";
registerHooks({ resolve(specifier, context, nextResolve) {
 if (specifier === "@earendil-works/pi-coding-agent") return {url:pathToFileURL(sdkPath).href,shortCircuit:true};
 return nextResolve(specifier,context);
}});
const sdk = await import(pathToFileURL(sdkPath).href);
const { default: fence, loadMaintenancePolicy, maintenanceToolDecision, createMaintenanceBashOperations } = await import("./maintenance-fence.ts");
process.env.PI_OFFLINE = "1";
const hash = (data: string | Buffer) => createHash("sha256").update(data).digest("hex");

function fixture() {
 const root=realpathSync(mkdtempSync(join(tmpdir(),"maintenance-native-")));
 const workspace=join(root,"workspace"), protectedRoot=join(root,"controller");
 mkdirSync(workspace);mkdirSync(protectedRoot);mkdirSync(join(workspace,"sub"));
 writeFileSync(join(workspace,"app.txt"),"owned needle\n");writeFileSync(join(protectedRoot,"oracle.txt"),"HIDDEN_ORACLE_NEEDLE");
 const lock=join(root,"lock.json");writeFileSync(lock,"{}");
 const policy={schema:"maintenance.containment/1",workspaceRoot:workspace,protectedRoots:[protectedRoot],readOnlyRoots:[],runtimeModule:sdkPath,runtimeVersion:"0.0.71",runtimeLock:lock,runtimeLockDigest:hash("{}"),inferenceDomains:["example.invalid"]};
 const file=join(root,"policy.json");writeFileSync(file,JSON.stringify(policy));
 const digest=hash(readFileSync(file));
 return {root,workspace,protectedRoot,policy,file,digest,cleanup:()=>rmSync(root,{recursive:true,force:true})};
}

function handlers(f:ReturnType<typeof fixture>) {
 const priorPolicy=process.env.AILANG_MAINTENANCE_POLICY, priorDigest=process.env.AILANG_MAINTENANCE_POLICY_DIGEST, priorWrapper=process.env.AILANG_MAINTENANCE_WRAPPER;
 process.env.AILANG_MAINTENANCE_POLICY=f.file;process.env.AILANG_MAINTENANCE_POLICY_DIGEST=f.digest;
 process.env.AILANG_MAINTENANCE_WRAPPER=join(f.root,"wrapper.mjs");writeFileSync(process.env.AILANG_MAINTENANCE_WRAPPER,"// trusted test wrapper");
 const events=new Map<string,Function>(),registered=new Map<string,any>();
 try {fence({on:(name:string,fn:Function)=>events.set(name,fn),registerTool:(tool:any)=>registered.set(tool.name,tool)});} finally {
  for(const [name,value]of [["AILANG_MAINTENANCE_POLICY",priorPolicy],["AILANG_MAINTENANCE_POLICY_DIGEST",priorDigest],["AILANG_MAINTENANCE_WRAPPER",priorWrapper]]){if(value===undefined)delete process.env[name!];else process.env[name!]=value;}
 }
 return {events,registered};
}

async function guarded(tool:any,input:any,hook:Function,cwd:string) {
 const prepared=tool.prepareArguments?tool.prepareArguments(input):input;
 const blocked=await hook({toolName:tool.name,input:prepared},{cwd,hasUI:false});
 if(blocked?.block)return {blocked:true,text:blocked.reason};
 const result=await tool.execute("qualification-native",prepared,new AbortController().signal,undefined,{cwd});
 return {blocked:false,text:result.content.filter((x:any)=>x.type==="text").map((x:any)=>x.text).join("\n")};
}

test("strict policy identity and explicit backend version",()=>{
 const f=fixture();try {
  assert.equal(loadMaintenancePolicy(f.file,f.digest).workspaceRoot,f.workspace);
  assert.throws(()=>loadMaintenancePolicy(f.file,"0".repeat(64)),/digest/);
  writeFileSync(f.file,JSON.stringify({...f.policy,unknown:true}));assert.throws(()=>loadMaintenancePolicy(f.file,hash(readFileSync(f.file))),/policy/);
  writeFileSync(f.file,JSON.stringify({...f.policy,runtimeVersion:"0.0.1"}));assert.throws(()=>loadMaintenancePolicy(f.file,hash(readFileSync(f.file))),/version|policy/);
 }finally{f.cleanup();}
});

test("actual pi0.85.1 native read write edit grep find ls remain usable",async()=>{
 const f=fixture();try {
  const {events,registered}=handlers(f);const hook=events.get("tool_call")!;
  assert(registered.has("bash"),"sandboxed bash override absent");
  const calls=[
   [sdk.createReadTool(f.workspace),{path:"app.txt"},"owned needle"],
   [sdk.createWriteTool(f.workspace),{path:"sub/new.txt",content:"new needle"},"Successfully"],
   [sdk.createEditTool(f.workspace),{path:"app.txt",edits:[{oldText:"owned needle",newText:"edited needle"}]},"Successfully"],
   [sdk.createGrepTool(f.workspace),{pattern:"needle",path:"."},"needle"],
   [sdk.createFindTool(f.workspace),{pattern:"*.txt",path:"."},"app.txt"],
   [sdk.createLsTool(f.workspace),{path:"."},"app.txt"],
  ];
  for(const [tool,input,expected]of calls){const result=await guarded(tool,input,hook,f.workspace);assert.equal(result.blocked,false,`${(tool as any).name}: ${result.text}`);assert(result.text.includes(expected as string),result.text);}
  assert.equal(readFileSync(join(f.workspace,"app.txt"),"utf8"),"edited needle\n");
 }finally{f.cleanup();}
});

test("actual native tools cannot read/search/list or mutate protected source",async()=>{
 const f=fixture();try{
  const {events}=handlers(f);const hook=events.get("tool_call")!;
  for(const [tool,input]of [
   [sdk.createReadTool(f.workspace),{path:join(f.protectedRoot,"oracle.txt")}],
   [sdk.createWriteTool(f.workspace),{path:join(f.protectedRoot,"oracle.txt"),content:"weakened"}],
   [sdk.createEditTool(f.workspace),{path:join(f.protectedRoot,"oracle.txt"),edits:[{oldText:"HIDDEN",newText:"leaked"}]}],
   [sdk.createGrepTool(f.workspace),{pattern:"NEEDLE",path:f.protectedRoot}],
   [sdk.createFindTool(f.workspace),{pattern:"*",path:f.protectedRoot}],
   [sdk.createLsTool(f.workspace),{path:f.protectedRoot}],
  ]){const result=await guarded(tool,input,hook,f.workspace);assert.equal(result.blocked,true,(tool as any).name);assert(!result.text.includes("HIDDEN_ORACLE_NEEDLE"));}
  assert.equal(readFileSync(join(f.protectedRoot,"oracle.txt"),"utf8"),"HIDDEN_ORACLE_NEEDLE");
 }finally{f.cleanup();}
});

test("relative cwd, prefix confusion, unknown tool/path shapes fail closed",async()=>{
 const f=fixture();try{
  const policy=loadMaintenancePolicy(f.file,f.digest);
  for(const [name,input,cwd]of [
   ["read",{path:"oracle.txt"},f.protectedRoot],
   ["write",{path:join(f.workspace+"-sibling","leak"),content:"x"},f.workspace],
   ["read",{file_path:"app.txt"},f.workspace],
   ["read",{path:"app.txt",file_path:join(f.protectedRoot,"oracle.txt")},f.workspace],
   ["read",{path:42},f.workspace],
   ["read",{path:"app.txt",offset:"1"},f.workspace],
   ["edit",{path:"app.txt",edits:{oldText:"x",newText:"y"}},f.workspace],
   ["read",{path:"@app.txt"},f.workspace],
   ["powershell",{command:"Get-Content oracle"},f.workspace],
   ["unknown",{},f.workspace],
  ]){assert(maintenanceToolDecision(policy,name,input,cwd)?.block,`${name} ${JSON.stringify(input)}`);}
  assert.equal(maintenanceToolDecision(policy,"read",{path:"../app.txt"},join(f.workspace,"sub")),undefined);
  assert.equal(maintenanceToolDecision(policy,"ls",{},f.workspace),undefined);
 }finally{f.cleanup();}
});

test("symlink leaf/ancestor and recursive search cannot expose another root",async()=>{
 const f=fixture();try{
  symlinkSync(f.protectedRoot,join(f.workspace,"escape"));
  symlinkSync(join(f.protectedRoot,"oracle.txt"),join(f.workspace,"linked.txt"));
  symlinkSync(join(f.protectedRoot,"missing"),join(f.workspace,"dangling"));
  const {events}=handlers(f),hook=events.get("tool_call")!;
  for(const [tool,input]of [
   [sdk.createReadTool(f.workspace),{path:"escape/oracle.txt"}],
   [sdk.createReadTool(f.workspace),{path:"linked.txt"}],
   [sdk.createWriteTool(f.workspace),{path:"escape/new.txt",content:"x"}],
   [sdk.createWriteTool(f.workspace),{path:"dangling/new.txt",content:"x"}],
   [sdk.createGrepTool(f.workspace),{pattern:"NEEDLE",path:"."}],
   [sdk.createFindTool(f.workspace),{pattern:"*",path:"."}],
   [sdk.createLsTool(f.workspace),{path:"."}],
  ]){assert.equal((await guarded(tool,input,hook,f.workspace)).blocked,true,(tool as any).name);}
 }finally{f.cleanup();}
});

test("bash override executes only the trusted tools-network wrapper with abort/time bounds",async()=>{
 const f=fixture();try{
  const wrapper=join(f.root,"capture.mjs"),capture=join(f.workspace,"wrapper-args.json");
  writeFileSync(wrapper,`import fs from 'node:fs'; fs.writeFileSync(${JSON.stringify(capture)},JSON.stringify(process.argv.slice(2))); console.log('wrapped'); console.error('stderr-observed');`);
  const ops=createMaintenanceBashOperations(f.file,f.digest,wrapper);
  const output:Buffer[]=[];const result=await ops.exec("echo harmless",f.workspace,{onData:(x:Buffer)=>output.push(x),timeout:1});
  assert.equal(result.exitCode,0);assert(Buffer.concat(output).toString().includes("stderr-observed"));
  assert.deepEqual(JSON.parse(readFileSync(capture,"utf8")),["--policy",f.file,"--digest",f.digest,"--network","tools","--","bash","-lc","echo harmless"]);
  writeFileSync(wrapper,"setInterval(()=>{},1000)");
  await assert.rejects(()=>ops.exec("ignored",f.workspace,{onData:()=>{},timeout:0.05}),/timeout/);
  const controller=new AbortController();controller.abort();await assert.rejects(()=>ops.exec("ignored",f.workspace,{onData:()=>{},signal:controller.signal}),/abort/);
 }finally{f.cleanup();}
});
