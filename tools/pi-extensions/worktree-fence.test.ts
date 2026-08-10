import fence from "./worktree-fence.ts";
import { mkdirSync, writeFileSync, symlinkSync, rmSync, realpathSync } from "node:fs";
import { join } from "node:path";

const TMP = "/tmp/fence-test";
rmSync(TMP, { recursive: true, force: true });
const WT = join(TMP, "wt");
mkdirSync(join(WT, "sub"), { recursive: true });
mkdirSync(join(TMP, "outside"), { recursive: true });
writeFileSync(join(WT, "ok.txt"), "x");
// A symlink INSIDE the worktree pointing OUT of it — the classic escape.
symlinkSync(join(TMP, "outside"), join(WT, "escape"));

process.env.PI_FENCE_ROOT = WT;
process.chdir(WT);

let handler: any;
fence({ on: (_e: string, h: any) => { handler = h; } });

let pass = 0, fail = 0;
async function arm(name: string, input: any, wantBlock: boolean, toolName = "write") {
  const r = await handler({ toolName, input }, { hasUI: false });
  const blocked = !!(r && r.block);
  if (blocked === wantBlock) { console.log(`  ✓ ${name} → ${blocked ? "BLOCKED" : "allowed"}`); pass++; }
  else { console.log(`  ✗✗ ${name} → ${blocked ? "BLOCKED" : "allowed"} (wanted ${wantBlock ? "BLOCKED" : "allowed"})`); fail++; }
}

(async () => {
  console.log("--- must ALLOW (legitimate worktree writes) ---");
  await arm("new file in root", { path: join(WT, "new.go") }, false);
  await arm("existing file", { path: join(WT, "ok.txt") }, false);
  await arm("nested new dir", { path: join(WT, "a/b/c.go") }, false);
  await arm("relative path", { path: "sub/x.go" }, false);
  await arm("edit tool", { path: join(WT, "ok.txt") }, false, "edit");
  await arm("macOS /tmp symlink root", { path: join(WT, "sub/deep.go") }, false);

  console.log("--- must BLOCK (escapes) ---");
  await arm("absolute outside", { path: join(TMP, "outside/evil.go") }, true);
  await arm("dotdot escape", { path: join(WT, "../outside/evil.go") }, true);
  await arm("relative dotdot", { path: "../outside/evil.go" }, true);
  await arm("SYMLINK escape", { path: join(WT, "escape/evil.go") }, true);
  await arm("home dir", { path: process.env.HOME + "/.zshenv" }, true);
  await arm("sibling prefix /a/bc vs /a/b", { path: WT + "-sibling/evil.go" }, true);
  await arm("main checkout", { path: "/Users/voightkampff/dev/sunholo-data/ailang/Makefile" }, true);

  console.log("--- fail-closed ---");
  await arm("no path key", { contents: "x" }, true);
  await arm("path is not a string", { path: 42 }, true);

  console.log("--- resolve-base: root != cwd (the bug the live run caught) ---");
  // Re-arm the fence with a root that is NOT the cwd. A relative path must be
  // judged against cwd (where pi writes), not against root. With the old code
  // "sneaky.go" resolved to <root>/sneaky.go and was allowed, while pi wrote it
  // to <cwd>/sneaky.go — outside the fence.
  process.env.PI_FENCE_ROOT = join(TMP, "elsewhere");
  mkdirSync(join(TMP, "elsewhere"), { recursive: true });
  let handler2: any;
  fence({ on: (_e: string, h: any) => { handler2 = h; } });
  const r2 = await handler2({ toolName: "write", input: { path: "sneaky.go" } }, { hasUI: false });
  if (r2 && r2.block) { console.log("  \u2713 relative path judged against cwd \u2192 BLOCKED"); pass++; }
  else { console.log("  \u2717\u2717 relative path judged against root \u2192 allowed (ESCAPE)"); fail++; }
  process.env.PI_FENCE_ROOT = WT;

  console.log("--- must IGNORE (not a write tool) ---");
  await arm("bash passes through (sandbox handles it)", { command: "rm -rf /" }, false, "bash");
  await arm("read passes through", { path: "/etc/passwd" }, false, "read");

  console.log(`\n  ${pass} passed, ${fail} failed`);
  rmSync(TMP, { recursive: true, force: true });
  process.exit(fail ? 1 : 0);
})();
