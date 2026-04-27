/**
 * microrag-plugin.test.ts — unit tests for the opencode microrag plugin.
 *
 * Run: npx vitest run  (from this directory)
 *
 * Uses vitest vi.mock to intercept child_process.spawnSync — no live ailang
 * binary required. Each test verifies the exact arguments forwarded to the
 * CLI subprocess and the injection text returned to opencode.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import * as child_process from "child_process";

// --- mock spawnSync before importing the plugin module ---

vi.mock("child_process", () => ({
  spawnSync: vi.fn(),
}));

const mockedSpawn = vi.mocked(child_process.spawnSync);

// Import after mocking.
import { preToolUse, postToolUse, sessionStart } from "./microrag-plugin.js";

const RAG_INJECTION = "🧠 μRAG: relevant context snippet";

function makeSpawnResult(injectionText: string | null) {
  if (injectionText === null) {
    return { status: 1, stdout: "", stderr: "error" };
  }
  return {
    status: 0,
    stdout: JSON.stringify({
      injection: { injection_text: injectionText },
    }),
    stderr: "",
  };
}

beforeEach(() => {
  vi.resetAllMocks();
  delete process.env["AILANG_MICRORAG_ENABLED"];
  delete process.env["AILANG_MICRORAG_SESSION"];
});

afterEach(() => {
  delete process.env["AILANG_MICRORAG_ENABLED"];
  delete process.env["AILANG_MICRORAG_SESSION"];
});

// --- sessionStart ---

describe("sessionStart", () => {
  it("sets AILANG_MICRORAG_SESSION from info.id", () => {
    sessionStart({ id: "ses_abc123" });
    expect(process.env["AILANG_MICRORAG_SESSION"]).toBe("ses_abc123");
  });

  it("is a no-op when id is missing", () => {
    sessionStart({});
    expect(process.env["AILANG_MICRORAG_SESSION"]).toBeUndefined();
  });
});

// --- preToolUse: disabled guard ---

describe("preToolUse disabled guard", () => {
  it("returns undefined when AILANG_MICRORAG_ENABLED=0", () => {
    process.env["AILANG_MICRORAG_ENABLED"] = "0";
    const result = preToolUse("Edit", { file_path: "/foo/bar.go", new_string: "x" });
    expect(result).toBeUndefined();
    expect(mockedSpawn).not.toHaveBeenCalled();
  });
});

// --- preToolUse: tool filtering ---

describe("preToolUse tool filtering", () => {
  it("passes for Edit", () => {
    mockedSpawn.mockReturnValue(makeSpawnResult(RAG_INJECTION) as ReturnType<typeof child_process.spawnSync>);
    const r = preToolUse("Edit", { file_path: "/a/b.go", new_string: "x" });
    expect(r).toBe(RAG_INJECTION);
  });

  it("passes for Write", () => {
    mockedSpawn.mockReturnValue(makeSpawnResult(RAG_INJECTION) as ReturnType<typeof child_process.spawnSync>);
    const r = preToolUse("Write", { file_path: "/a/b.go", content: "x" });
    expect(r).toBe(RAG_INJECTION);
  });

  it("passes for Read (no content)", () => {
    mockedSpawn.mockReturnValue(makeSpawnResult(RAG_INJECTION) as ReturnType<typeof child_process.spawnSync>);
    const r = preToolUse("Read", { file_path: "/a/b.go" });
    expect(r).toBe(RAG_INJECTION);
  });

  it("skips non-watched tool (Bash)", () => {
    const r = preToolUse("Bash", { command: "ls" });
    expect(r).toBeUndefined();
    expect(mockedSpawn).not.toHaveBeenCalled();
  });
});

// --- preToolUse: path filtering ---

describe("preToolUse path filtering", () => {
  it("skips node_modules paths", () => {
    const r = preToolUse("Read", { file_path: "/proj/node_modules/foo/index.js" });
    expect(r).toBeUndefined();
    expect(mockedSpawn).not.toHaveBeenCalled();
  });

  it("skips .git paths", () => {
    const r = preToolUse("Read", { file_path: "/proj/.git/COMMIT_EDITMSG" });
    expect(r).toBeUndefined();
    expect(mockedSpawn).not.toHaveBeenCalled();
  });

  it("skips binary extensions", () => {
    const r = preToolUse("Read", { file_path: "/proj/image.png" });
    expect(r).toBeUndefined();
    expect(mockedSpawn).not.toHaveBeenCalled();
  });

  it("skips missing file_path", () => {
    const r = preToolUse("Edit", { new_string: "x" });
    expect(r).toBeUndefined();
    expect(mockedSpawn).not.toHaveBeenCalled();
  });
});

// --- preToolUse: subprocess args ---

describe("preToolUse subprocess args", () => {
  it("passes correct args for Edit with content", () => {
    mockedSpawn.mockReturnValue(makeSpawnResult(RAG_INJECTION) as ReturnType<typeof child_process.spawnSync>);
    preToolUse("Edit", { file_path: "/src/foo.go", new_string: "hello world" });
    expect(mockedSpawn).toHaveBeenCalledWith(
      "ailang",
      ["micro-rag", "context", "--tool", "Edit", "--file", "/src/foo.go", "--content", "hello world"],
      expect.objectContaining({ timeout: 3000 })
    );
  });

  it("omits --content for Read", () => {
    mockedSpawn.mockReturnValue(makeSpawnResult(RAG_INJECTION) as ReturnType<typeof child_process.spawnSync>);
    preToolUse("Read", { file_path: "/src/foo.go" });
    const args = mockedSpawn.mock.calls[0][1] as string[];
    expect(args).not.toContain("--content");
  });

  it("truncates content to 4096 chars", () => {
    mockedSpawn.mockReturnValue(makeSpawnResult(RAG_INJECTION) as ReturnType<typeof child_process.spawnSync>);
    const longContent = "x".repeat(10000);
    preToolUse("Write", { file_path: "/a.go", content: longContent });
    const args = mockedSpawn.mock.calls[0][1] as string[];
    const contentIdx = args.indexOf("--content");
    expect(contentIdx).toBeGreaterThan(-1);
    expect(args[contentIdx + 1].length).toBe(4096);
  });
});

// --- preToolUse: MultiEdit ---

describe("preToolUse MultiEdit", () => {
  it("extracts file_path from first edit", () => {
    mockedSpawn.mockReturnValue(makeSpawnResult(RAG_INJECTION) as ReturnType<typeof child_process.spawnSync>);
    preToolUse("MultiEdit", {
      edits: [
        { file_path: "/src/a.go", new_string: "a" },
        { file_path: "/src/b.go", new_string: "b" },
      ],
    });
    const args = mockedSpawn.mock.calls[0][1] as string[];
    expect(args).toContain("/src/a.go");
  });

  it("joins new_string from all edits for content", () => {
    mockedSpawn.mockReturnValue(makeSpawnResult(RAG_INJECTION) as ReturnType<typeof child_process.spawnSync>);
    preToolUse("MultiEdit", {
      edits: [
        { file_path: "/a.go", new_string: "hello" },
        { file_path: "/a.go", new_string: "world" },
      ],
    });
    const args = mockedSpawn.mock.calls[0][1] as string[];
    const contentIdx = args.indexOf("--content");
    expect(args[contentIdx + 1]).toBe("hello\nworld");
  });
});

// --- preToolUse: failure / no result ---

describe("preToolUse subprocess failure", () => {
  it("returns undefined when spawnSync exits non-zero", () => {
    mockedSpawn.mockReturnValue(makeSpawnResult(null) as ReturnType<typeof child_process.spawnSync>);
    const r = preToolUse("Edit", { file_path: "/a.go", new_string: "x" });
    expect(r).toBeUndefined();
  });

  it("returns undefined when injection_text missing from JSON", () => {
    mockedSpawn.mockReturnValue({
      status: 0,
      stdout: JSON.stringify({ injection: {} }),
      stderr: "",
    } as ReturnType<typeof child_process.spawnSync>);
    const r = preToolUse("Edit", { file_path: "/a.go", new_string: "x" });
    expect(r).toBeUndefined();
  });

  it("returns undefined on invalid JSON stdout", () => {
    mockedSpawn.mockReturnValue({
      status: 0,
      stdout: "not json",
      stderr: "",
    } as ReturnType<typeof child_process.spawnSync>);
    const r = preToolUse("Edit", { file_path: "/a.go", new_string: "x" });
    expect(r).toBeUndefined();
  });
});

// --- postToolUse: builtin-lint ---

function makeLintResult(nudges: string[]) {
  return {
    status: 0,
    stdout: JSON.stringify({
      nudges: nudges.map((t) => ({ injection_text: t })),
    }),
    stderr: "",
  };
}

describe("postToolUse builtin-lint", () => {
  it("returns joined nudge text for .ail Edit", () => {
    mockedSpawn.mockReturnValue(
      makeLintResult(["═ μRAG/builtin: httpGet ═", "═ μRAG/builtin: parseJSON ═"]) as ReturnType<typeof child_process.spawnSync>
    );
    const r = postToolUse("Edit", { file_path: "/src/foo.ail", new_string: "httpGet(url)" }, undefined);
    expect(r).toBe("═ μRAG/builtin: httpGet ═\n═ μRAG/builtin: parseJSON ═");
    expect(mockedSpawn).toHaveBeenCalledWith(
      "ailang",
      expect.arrayContaining(["micro-rag", "lint-builtin", "--file", "/src/foo.ail"]),
      expect.objectContaining({ timeout: 3000 })
    );
  });

  it("returns undefined for non-.ail files", () => {
    const r = postToolUse("Write", { file_path: "/src/foo.go", content: "x" }, undefined);
    expect(r).toBeUndefined();
    expect(mockedSpawn).not.toHaveBeenCalled();
  });

  it("returns undefined for Read tool (no new code)", () => {
    const r = postToolUse("Read", { file_path: "/src/foo.ail" }, undefined);
    expect(r).toBeUndefined();
    expect(mockedSpawn).not.toHaveBeenCalled();
  });

  it("returns undefined when AILANG_MICRORAG_ENABLED=0", () => {
    process.env["AILANG_MICRORAG_ENABLED"] = "0";
    const r = postToolUse("Edit", { file_path: "/src/foo.ail", new_string: "x" }, undefined);
    expect(r).toBeUndefined();
    expect(mockedSpawn).not.toHaveBeenCalled();
  });

  it("returns undefined when nudges array is empty", () => {
    mockedSpawn.mockReturnValue(makeLintResult([]) as ReturnType<typeof child_process.spawnSync>);
    const r = postToolUse("Edit", { file_path: "/src/foo.ail", new_string: "x" }, undefined);
    expect(r).toBeUndefined();
  });

  it("returns undefined when subprocess fails", () => {
    mockedSpawn.mockReturnValue({ status: 1, stdout: "", stderr: "boom" } as ReturnType<typeof child_process.spawnSync>);
    const r = postToolUse("Edit", { file_path: "/src/foo.ail", new_string: "x" }, undefined);
    expect(r).toBeUndefined();
  });

  it("truncates code to 8192 chars before invoking subprocess", () => {
    mockedSpawn.mockReturnValue(makeLintResult(["nudge"]) as ReturnType<typeof child_process.spawnSync>);
    const longCode = "x".repeat(20000);
    postToolUse("Write", { file_path: "/a.ail", content: longCode }, undefined);
    const args = mockedSpawn.mock.calls[0][1] as string[];
    const codeIdx = args.indexOf("--code");
    expect(args[codeIdx + 1].length).toBe(8192);
  });
});
