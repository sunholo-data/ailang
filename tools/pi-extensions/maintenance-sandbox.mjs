/** Trusted argv bridge to the existing locked sandbox runtime. No model API. */
import { createHash } from "node:crypto";
import { readFileSync, realpathSync, statSync, mkdtempSync, rmSync } from "node:fs";
import { dirname, isAbsolute, join, relative, resolve, sep } from "node:path";
import { pathToFileURL } from "node:url";
import { tmpdir, homedir } from "node:os";
import { spawn } from "node:child_process";

const hash = bytes => createHash("sha256").update(bytes).digest("hex");
const inside = (root, path) => { const rel = relative(root, path); return rel === "" || (rel !== ".." && !rel.startsWith(`..${sep}`) && !isAbsolute(rel)); };
const quote = value => `'${value.replaceAll("'", `'"'"'`)}'`;

export function readContainmentPolicy(path, digest) {
 if (!isAbsolute(path) || realpathSync(path) !== path) throw new Error("Policy must be canonical absolute");
 const raw = readFileSync(path);
 if (!/^[a-f0-9]{64}$/.test(digest) || hash(raw) !== digest) throw new Error("Policy digest differs");
 const p = JSON.parse(raw);
 const keys = ["schema", "workspaceRoot", "protectedRoots", "readOnlyRoots", "runtimeModule", "runtimeVersion", "runtimeLock", "runtimeLockDigest", "inferenceDomains"];
 if (Object.keys(p).join("\0") !== keys.join("\0") || p.schema !== "maintenance.containment/1" || p.runtimeVersion !== "0.0.71") throw new Error("Unknown containment schema/fields/runtime");
 for (const name of ["protectedRoots", "readOnlyRoots", "inferenceDomains"]) if (!Array.isArray(p[name]) || p[name].some(v => typeof v !== "string")) throw new Error(`Invalid ${name}`);
 if (!p.protectedRoots.length) throw new Error("Protected source roots are required");
 for (const target of [p.workspaceRoot, p.runtimeModule, p.runtimeLock, ...p.protectedRoots, ...p.readOnlyRoots]) {
  if (typeof target !== "string" || !isAbsolute(target) || resolve(target) !== target || realpathSync(target) !== target) throw new Error("Noncanonical/symlink containment path");
 }
 if (!statSync(p.workspaceRoot).isDirectory() || p.protectedRoots.some(root => !statSync(root).isDirectory() || inside(root,p.workspaceRoot) || inside(p.workspaceRoot,root))) throw new Error("Protected source overlaps writable workspace");
 if (p.readOnlyRoots.some(root => inside(root,p.workspaceRoot) || inside(p.workspaceRoot,root))) throw new Error("Trusted runtime overlaps writable workspace");
 if (hash(readFileSync(p.runtimeLock)) !== p.runtimeLockDigest) throw new Error("Runtime lock digest differs");
 const packageInfo = JSON.parse(readFileSync(join(dirname(dirname(p.runtimeModule)),"package.json"),"utf8"));
 if (packageInfo.name !== "@anthropic-ai/sandbox-runtime" || packageInfo.version !== p.runtimeVersion) throw new Error("Resolved runtime package/version differs");
 if (p.inferenceDomains.some(domain => !domain || /[/*\\:@\s]/.test(domain))) throw new Error("Inference domains must be exact hostnames");
 return p;
}

export async function runSandboxed(options) {
 const p = readContainmentPolicy(options.policy,options.digest);
 if (process.platform !== "darwin" && process.platform !== "linux") throw new Error(`Unqualified sandbox platform ${process.platform}`);
 const runtimeRoot = options.runtimeRoot ? realpathSync(options.runtimeRoot) : p.workspaceRoot;
 if (!isAbsolute(runtimeRoot) || runtimeRoot === sep || !statSync(runtimeRoot).isDirectory() || p.protectedRoots.some(root => inside(root,runtimeRoot) || inside(runtimeRoot,root)) || p.readOnlyRoots.some(root => inside(root,runtimeRoot) || inside(runtimeRoot,root))) throw new Error("Runtime root overlaps protected/trusted source");
 if (!inside(runtimeRoot,realpathSync(process.cwd()))) throw new Error("Candidate cwd must be inside its owned runtime root");
 const tools = options.network === "tools";
 if (!tools && options.network !== "inference") throw new Error("Explicit tools/inference network mode required");
 if (!Array.isArray(options.argv) || !options.argv.length || options.argv.some(v => typeof v !== "string" || v.includes("\0"))) throw new Error("Child argv required");
 const inheritedScratch=process.env.AILANG_MAINTENANCE_SCRATCH;
 if(inheritedScratch && (!isAbsolute(inheritedScratch) || realpathSync(inheritedScratch)!==inheritedScratch || !statSync(inheritedScratch).isDirectory()))throw new Error("Invalid inherited SDK scratch root");
 // SDK proxy Unix socket paths must stay short on macOS. This is a new owned
 // scratch directory, never a global writable /tmp grant or an oracle mount.
 const scratch=mkdtempSync(join(inheritedScratch??(process.platform==="darwin"?"/private/tmp":realpathSync(tmpdir())),"am-"));
 process.env.TMPDIR=scratch;process.env.CLAUDE_CODE_TMPDIR=scratch;
 const { SandboxManager: sandbox } = await import(pathToFileURL(p.runtimeModule).href);
 if (!sandbox.isSupportedPlatform()) throw new Error("Sandbox runtime does not support this platform");
 // Read exceptions are only the declared trusted runtime/policy assets. Native
 // tool fencing separately refuses them; this is the kernel race boundary.
 const config = {filesystem:{denyRead:p.protectedRoots,allowRead:p.readOnlyRoots,
  allowWrite:[runtimeRoot,scratch],denyWrite:[...p.protectedRoots,...p.readOnlyRoots,options.policy,"/tmp/claude","/private/tmp/claude",join(homedir(),".npm/_logs"),join(homedir(),".claude/debug"),"/dev/tty","/dev/dtracehelper","/dev/autofs_nowait"]},
  network:{allowedDomains:tools?[]:p.inferenceDomains,deniedDomains:[],allowLocalBinding:!tools,allowUnixSockets:tools?[]:[scratch]}};
 let child;
 const forward = signal => { if(child && !child.killed) child.kill(signal); };
 try {
  await sandbox.initialize(config,undefined,false);
  if (!sandbox.isSandboxingEnabled()) throw new Error("Sandbox failed to initialize");
  const wrapped = await sandbox.wrapWithSandboxArgv(options.argv.map(quote).join(" "),"/bin/bash",undefined,undefined,process.cwd());
  // Tool/app processes receive no provider credentials, loader hooks or proxy
  // overrides from the controller. The SDK inserts its own proxy in the wrapper.
  const env = tools ? Object.fromEntries(Object.entries(process.env).filter(([key]) => /^(PATH|LANG|LC_[A-Z_]+|TZ|TMPDIR|TMP|TEMP)$/.test(key))) : {...process.env};
  if (tools) env.HOME = runtimeRoot;
  else { env.AILANG_MAINTENANCE_SCRATCH=scratch;env.CLAUDE_CODE_TMPDIR=scratch; }
  child=spawn(wrapped.argv[0],wrapped.argv.slice(1),{cwd:process.cwd(),env,stdio:"inherit"});
  process.on("SIGTERM",forward);process.on("SIGINT",forward);
  return await new Promise((yes,no)=>{child.once("error",no);child.once("close",(code,signal)=>yes(code??(signal?128:1)));});
 } finally {
  process.removeListener("SIGTERM",forward);process.removeListener("SIGINT",forward);
  await sandbox.reset();
  rmSync(scratch,{recursive:true,force:true});
 }
}

async function main() {
 const options={};let index=2;
 while(index<process.argv.length && process.argv[index]!=="--"){
  const name=process.argv[index++];const value=process.argv[index++];
  const key={"--policy":"policy","--digest":"digest","--network":"network","--runtime-root":"runtimeRoot"}[name];
  if(!key || value===undefined || options[key]!==undefined)throw new Error("Invalid/duplicate containment argument");options[key]=value;
 }
 if(process.argv[index++]!=="--")throw new Error("Containment child argv delimiter required");
 options.argv=process.argv.slice(index);process.exitCode=await runSandboxed(options);
}
if(process.argv[1] && pathToFileURL(realpathSync(process.argv[1])).href===import.meta.url){
 main().catch(error=>{console.error(`maintenance-sandbox: ${error.message}`);process.exitCode=125;});
}
