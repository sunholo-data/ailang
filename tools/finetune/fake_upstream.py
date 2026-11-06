#!/usr/bin/env python3
"""Scripted OpenAI-compatible model server: captures an agent harness's exact wire format, no GPU.

Each /v1/chat/completions request gets the next scripted reply (a tool call or a final
text). The harness really executes the tools, so the NEXT request carries its real tool
results. Put tools/finetune/trace_proxy.py in front of it to log every request.

Script file: JSON list of steps, each {"tool": name, "args": {...}} or {"text": "..."}.
The string "$SOLUTION" in args is replaced with the solution path parsed from the first
user message ("Write your complete solution to: <path>").

    python3 tools/finetune/fake_upstream.py --port 11436 --script steps.json
"""
import argparse
import http.server
import json
import re
import socketserver
import time

STATE = {"i": 0, "solution": None}


def reply_for(req, steps):
    msgs = req.get("messages") or []
    if STATE["solution"] is None:
        for m in msgs:
            c = m.get("content")
            text = c if isinstance(c, str) else json.dumps(c)
            mm = re.search(r"Write your complete solution to: \*{0,2}(\S+?\.ail)", text or "")
            if mm:
                STATE["solution"] = mm.group(1)
    i = min(STATE["i"], len(steps) - 1)
    STATE["i"] += 1
    st = json.loads(json.dumps(steps[i]).replace("$SOLUTION", STATE["solution"] or "benchmark/solution.ail"))
    return st


def make(steps):
    class H(http.server.BaseHTTPRequestHandler):
        protocol_version = "HTTP/1.1"

        def log_message(self, *a):
            pass

        def do_GET(self):
            body = json.dumps({"object": "list", "data": []}).encode()
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

        def do_POST(self):
            n = int(self.headers.get("Content-Length") or 0)
            req = json.loads(self.rfile.read(n) or b"{}")
            st = reply_for(req, steps)
            cid, now, model = f"chatcmpl-{STATE['i']}", int(time.time()), req.get("model", "fake")
            if "tool" in st:
                msg = {"role": "assistant", "content": None, "tool_calls": [{"id": f"call_{STATE['i']}", "type": "function",
                       "function": {"name": st["tool"], "arguments": json.dumps(st["args"])}}]}
                finish = "tool_calls"
            else:
                msg = {"role": "assistant", "content": st["text"]}
                finish = "stop"
            usage = {"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2}
            if req.get("stream"):
                self.send_response(200)
                self.send_header("Content-Type", "text/event-stream")
                self.send_header("Cache-Control", "no-cache")
                self.end_headers()
                delta = dict(msg)
                if "tool_calls" in delta:
                    delta["tool_calls"] = [dict(delta["tool_calls"][0], index=0)]
                chunks = [{"id": cid, "object": "chat.completion.chunk", "created": now, "model": model,
                           "choices": [{"index": 0, "delta": delta, "finish_reason": None}]},
                          {"id": cid, "object": "chat.completion.chunk", "created": now, "model": model,
                           "choices": [{"index": 0, "delta": {}, "finish_reason": finish}], "usage": usage}]
                for c in chunks:
                    self.wfile.write(f"data: {json.dumps(c)}\n\n".encode())
                self.wfile.write(b"data: [DONE]\n\n")
                self.wfile.flush()
                self.close_connection = True
                return
            body = json.dumps({"id": cid, "object": "chat.completion", "created": now, "model": model,
                               "choices": [{"index": 0, "message": msg, "finish_reason": finish}], "usage": usage}).encode()
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

    return H


class Server(socketserver.ThreadingMixIn, http.server.HTTPServer):
    daemon_threads = True


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--port", type=int, default=11436)
    ap.add_argument("--script", required=True)
    a = ap.parse_args()
    Server(("127.0.0.1", a.port), make(json.load(open(a.script)))).serve_forever()


if __name__ == "__main__":
    main()
