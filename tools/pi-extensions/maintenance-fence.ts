/**
 * Trusted maintenance-eval native tool fence (pi0.85.1).
 *
 * Native paths stay inside one owned app workspace. Controller/source/oracle
 * roots are never readable through native tools, including recursive searches.
 * Bash delegates to the same trusted SDK wrapper with network=tools (deny all),
 * while the outer pi process may use only its declared inference endpoints.
 *
 * Canonical preflight alone does not close check/use races. Production admission
 * also requires M8's outer OS read/write boundary and actual process canaries.
 */
import { spawn } from "node:child_process";
import { createHash } from "node:crypto";
import { lstatSync, readFileSync, readdirSync, realpathSync } from "node:fs";
import { dirname, isAbsolute, join, relative, resolve, sep } from "node:path";
import { createBashTool } from "@earendil-works/pi-coding-agent";

export interface MaintenancePolicy {
 schema: "maintenance.containment/1";
 workspaceRoot: string;
 protectedRoots: string[];
 readOnlyRoots: string[];
 runtimeModule: string;
 runtimeVersion: string;
 runtimeLock: string;
 runtimeLockDigest: string;
 inferenceDomains: string[];
}
const policyKeys = ["schema","workspaceRoot","protectedRoots","readOnlyRoots","runtimeModule","runtimeVersion","runtimeLock","runtimeLockDigest","inferenceDomains"];
const shapes:Record<string,string[]> = {
 read:["path","offset","limit"],write:["path","content"],edit:["path","edits"],
 grep:["pattern","path","glob","ignoreCase","literal","context","limit"],find:["pattern","path","limit"],ls:["path","limit"],bash:["command","timeout"],
};
const object = (x:unknown):x is Record<string,unknown> => typeof x==="object"&&x!==null&&!Array.isArray(x);
const digest = (x:string|Buffer) => createHash("sha256").update(x).digest("hex");
const sha = (x:unknown):x is string => typeof x==="string"&&/^[a-f0-9]{64}$/.test(x);
function strings(x:unknown):x is string[] {return Array.isArray(x)&&x.every(v=>typeof v==="string"&&v.length>0&&!v.includes("\0"));}
function inside(root:string,target:string):boolean {const p=relative(root,target);return p===""||p!==".."&&!p.startsWith(".."+sep)&&!isAbsolute(p);}

export function loadMaintenancePolicy(file:string,expectedDigest:string):MaintenancePolicy {
 if(!isAbsolute(file)||!sha(expectedDigest))throw new Error("maintenance-fence: absolute policy and sha256 digest required");
 const raw=readFileSync(file);
 if(digest(raw)!==expectedDigest)throw new Error("maintenance-fence: policy digest mismatch");
 const p:unknown=JSON.parse(raw.toString("utf8"));
 if(!object(p)||Object.keys(p).length!==policyKeys.length||Object.keys(p).some(k=>!policyKeys.includes(k))||p.schema!=="maintenance.containment/1"||typeof p.workspaceRoot!=="string"||!isAbsolute(p.workspaceRoot)||!strings(p.protectedRoots)||p.protectedRoots.length===0||!strings(p.readOnlyRoots)||typeof p.runtimeModule!=="string"||!isAbsolute(p.runtimeModule)||typeof p.runtimeLock!=="string"||!isAbsolute(p.runtimeLock)||!sha(p.runtimeLockDigest)||!strings(p.inferenceDomains)||p.runtimeVersion!=="0.0.71")throw new Error("maintenance-fence: invalid policy/version");
 const policy=p as unknown as MaintenancePolicy;
 policy.workspaceRoot=realpathSync(policy.workspaceRoot);
 if(!lstatSync(policy.workspaceRoot).isDirectory())throw new Error("maintenance-fence: workspace is not a directory");
 for(const roots of [policy.protectedRoots,policy.readOnlyRoots])for(let i=0;i<roots.length;i++){
  if(!isAbsolute(roots[i]))throw new Error("maintenance-fence: absolute policy roots required");
  roots[i]=realpathSync(roots[i]);
 }
 if(policy.protectedRoots.some(root=>inside(root,policy.workspaceRoot)))throw new Error("maintenance-fence: workspace overlaps protected root");
 if(digest(readFileSync(policy.runtimeLock))!==policy.runtimeLockDigest)throw new Error("maintenance-fence: runtime lock digest mismatch");
 return policy;
}

// Resolve missing write targets through their nearest existing ancestor. A
// dangling symlink must fail, rather than appear to be an ordinary missing file.
function canonical(target:string):string {
 let existing=target;const suffix:string[]=[];
 for(;;){
  try{lstatSync(existing);}
  catch(error:any){
   if(error.code!=="ENOENT")throw error;
   const parent=dirname(existing);if(parent===existing)throw error;
   suffix.unshift(existing.slice(parent.length+1));existing=parent;continue;
  }
  // A dangling symlink's realpath error propagates. It is not a missing suffix.
  return resolve(realpathSync(existing),...suffix);
 }
}

function allowed(policy:MaintenancePolicy,target:string):void {
 if(!inside(policy.workspaceRoot,target)||policy.protectedRoots.some(root=>inside(root,target)))throw new Error("path is outside owned workspace or protected");
}
function regular(target:string,mayBeDirectory=false):void {
 const info=lstatSync(target);
 if(info.isSymbolicLink()||!(info.isFile()||mayBeDirectory&&info.isDirectory())||info.isFile()&&info.nlink>1)throw new Error("non-regular/link target refused");
}
function safeTree(policy:MaintenancePolicy,target:string,recursive:boolean):void {
 let entries=0;
 const visit=(current:string)=>{
  allowed(policy,current);regular(current,true);
  if(!lstatSync(current).isDirectory())return;
  for(const entry of readdirSync(current)){
   if(++entries>10000)throw new Error("native search preflight exceeded bounded tree");
   const child=join(current,entry);allowed(policy,child);regular(child,true);
   if(recursive&&lstatSync(child).isDirectory())visit(child);
  }
 };
 visit(target);
}

/** Returns a refusal or undefined. Rewrites the admitted path to its canonical
 * absolute identity so SDK @/Unicode/path expansion cannot select a second file. */
export function maintenanceToolDecision(policy:MaintenancePolicy,name:string,input:unknown,cwd:string):{block:true,reason:string}|undefined {
 try{
  const keys=shapes[name];if(!keys||!object(input)||Object.keys(input).some(k=>!keys.includes(k)))throw new Error("unknown tool/input shape");
  for(const key of ["offset","limit","context"])if(input[key]!==undefined&&(typeof input[key]!=="number"||!Number.isSafeInteger(input[key])||(input[key] as number)<(key==="context"?0:1)))throw new Error("unknown numeric option shape");
  for(const key of ["ignoreCase","literal"])if(input[key]!==undefined&&typeof input[key]!=="boolean")throw new Error("unknown boolean option shape");
  if(input.glob!==undefined&&typeof input.glob!=="string")throw new Error("unknown glob shape");
  if(name==="edit"&&(!Array.isArray(input.edits)||input.edits.length===0||input.edits.some(edit=>!object(edit)||Object.keys(edit).some(k=>k!=="oldText"&&k!=="newText")||typeof edit.oldText!=="string"||typeof edit.newText!=="string")))throw new Error("unknown edit shape");
  if(!isAbsolute(cwd))throw new Error("absolute tool cwd required");
  const base=realpathSync(cwd);allowed(policy,base);
  if(name==="bash"){
   if(typeof input.command!=="string"||!input.command||input.command.includes("\0")||input.timeout!==undefined&&(typeof input.timeout!=="number"||!Number.isFinite(input.timeout)||input.timeout<=0))throw new Error("unknown bash shape");
   input.timeout=Math.min((input.timeout as number|undefined)??120,120);
   return;
  }
  const pathRequired=name==="read"||name==="write"||name==="edit";
  if(pathRequired&&typeof input.path!=="string"||input.path!==undefined&&typeof input.path!=="string")throw new Error("missing/unknown path shape");
  const raw=(input.path??".") as string;
  if(raw.length===0||raw.includes("\0")||raw.startsWith("@")||raw.startsWith("~")||/[^\S ]/u.test(raw))throw new Error("unsupported path expansion/whitespace");
  const target=canonical(resolve(base,raw));allowed(policy,target);
  if(/[^\S ]/u.test(target))throw new Error("unsupported canonical path whitespace");
  if(name==="write"){
   if(typeof input.content!=="string")throw new Error("write content required");
   try{regular(target);}catch(error:any){if(error.code!=="ENOENT")throw error;}
  }else if(name==="read"||name==="edit")regular(target);
  else{
   if((name==="grep"||name==="find")&&typeof input.pattern!=="string")throw new Error("search pattern required");
   safeTree(policy,target,name!=="ls");
  }
  input.path=target;
  return;
 }catch(error:any){return {block:true,reason:`maintenance-fence: refused ${name}: ${error.message}`};}
}

// The wrapper is controller materialized, never an agent-supplied command path.
// Both streams stay observable. Timeouts/abort kill the detached process group.
export function createMaintenanceBashOperations(file:string,expectedDigest:string,wrapper:string) {
 if(!isAbsolute(wrapper)||!lstatSync(wrapper).isFile())throw new Error("maintenance-fence: trusted absolute wrapper required");
 return {async exec(command:string,cwd:string,{onData,signal,timeout,env}:{onData:(data:Buffer)=>void,signal?:AbortSignal,timeout?:number,env?:NodeJS.ProcessEnv}):Promise<{exitCode:number|null}>{
  if(signal?.aborted)throw new Error("aborted");
  const policy=loadMaintenancePolicy(file,expectedDigest);
  const args={command,timeout};
  const refused=maintenanceToolDecision(policy,"bash",args,cwd);if(refused)throw new Error(refused.reason);
  const boundedTimeout=args.timeout!;
  return new Promise((resolvePromise,reject)=>{
   const child=spawn(process.execPath,[wrapper,"--policy",file,"--digest",expectedDigest,"--network","tools","--","bash","-lc",command],{cwd,env:env??process.env,detached:process.platform!=="win32",stdio:["ignore","pipe","pipe"],windowsHide:true});
   let timedOut=false,settled=false;let timer:ReturnType<typeof setTimeout>|undefined;
   const kill=()=>{if(child.pid){try{if(process.platform!=="win32")process.kill(-child.pid,"SIGKILL");else child.kill("SIGKILL");}catch{child.kill("SIGKILL");}}};
   const cleanup=()=>{if(timer)clearTimeout(timer);signal?.removeEventListener("abort",kill);};
   const fail=(error:Error)=>{if(settled)return;settled=true;cleanup();reject(error);};
   child.stdout.on("data",onData);child.stderr.on("data",onData);
   child.on("error",fail);
   child.on("close",code=>{if(settled)return;settled=true;cleanup();if(signal?.aborted)reject(new Error("aborted"));else if(timedOut)reject(new Error(`timeout:${timeout}`));else resolvePromise({exitCode:code});});
   timer=setTimeout(()=>{timedOut=true;kill();},boundedTimeout*1000);
   signal?.addEventListener("abort",kill,{once:true});if(signal?.aborted)kill();
  });
 }};
}

export default function(pi:any){
 const file=process.env.AILANG_MAINTENANCE_POLICY??"",expectedDigest=process.env.AILANG_MAINTENANCE_POLICY_DIGEST??"";
 const policy=loadMaintenancePolicy(file,expectedDigest);
 const operations=createMaintenanceBashOperations(file,expectedDigest,process.env.AILANG_MAINTENANCE_WRAPPER??"");
 const bash=createBashTool(policy.workspaceRoot,{operations,exposeSessionEnvironment:false});
 pi.registerTool({...bash,label:"bash (maintenance confined)"});
 pi.on("tool_call",async(event:any,ctx:any)=>maintenanceToolDecision(policy,event.toolName,event.input,ctx?.cwd??process.cwd()));
 // Interactive shell escapes must share the same no-network tool operations.
 pi.on("user_bash",()=>({operations}));
}
