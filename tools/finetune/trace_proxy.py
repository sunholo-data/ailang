#!/usr/bin/env python3
"""Logging reverse proxy between an agent harness and Ollama (fine-tune diagnostics).

pi keeps no transcript for eval runs, and the banked result holds only tool-call
counts. This proxy records every request body and the full response (streamed or
not) so a run can be replayed turn by turn: what system prompt and tools pi sent,
and exactly what the model emitted.

    python3 tools/finetune/trace_proxy.py --listen 11435 --upstream http://127.0.0.1:11434 \
        --out ~/models/finetune/traces/run.jsonl

Point a pi provider's baseUrl at http://127.0.0.1:11435/v1. Headers (including
X-Rig-Lease) are forwarded unchanged; nothing is logged from request headers.
"""
import argparse
import http.server
import json
import os
import socketserver
import threading
import time
import urllib.request

LOCK = threading.Lock()


def make_handler(upstream, out):
    class H(http.server.BaseHTTPRequestHandler):
        protocol_version = "HTTP/1.1"

        def log_message(self, *a):
            pass

        def _proxy(self):
            n = int(self.headers.get("Content-Length") or 0)
            body = self.rfile.read(n) if n else None
            hdrs = {k: v for k, v in self.headers.items() if k.lower() not in ("host", "content-length", "accept-encoding")}
            req = urllib.request.Request(upstream + self.path, data=body, headers=hdrs, method=self.command)
            t0 = time.time()
            chunks = []
            try:
                resp = urllib.request.urlopen(req, timeout=3600)
                status, rh = resp.status, resp.headers
            except urllib.error.HTTPError as e:
                resp, status, rh = e, e.code, e.headers
            self.send_response(status)
            for k, v in rh.items():
                if k.lower() not in ("transfer-encoding", "content-length", "connection"):
                    self.send_header(k, v)
            self.send_header("Transfer-Encoding", "chunked")
            self.end_headers()
            while True:
                c = resp.read1(65536) if hasattr(resp, "read1") else resp.read(65536)
                if not c:
                    break
                chunks.append(c)
                self.wfile.write(b"%x\r\n%s\r\n" % (len(c), c))
                self.wfile.flush()
            self.wfile.write(b"0\r\n\r\n")
            rec = {"t": t0, "secs": round(time.time() - t0, 2), "method": self.command, "path": self.path,
                   "status": status, "request": _json(body), "response": b"".join(chunks).decode("utf-8", "replace")}
            with LOCK, open(out, "a") as f:
                f.write(json.dumps(rec) + "\n")

        do_GET = do_POST = _proxy

    return H


def _json(b):
    if not b:
        return None
    try:
        return json.loads(b)
    except ValueError:
        return b.decode("utf-8", "replace")


class Server(socketserver.ThreadingMixIn, http.server.HTTPServer):
    daemon_threads = True


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--listen", type=int, default=11435)
    ap.add_argument("--upstream", default="http://127.0.0.1:11434")
    ap.add_argument("--out", required=True)
    a = ap.parse_args()
    os.makedirs(os.path.dirname(os.path.expanduser(a.out)), exist_ok=True)
    Server(("127.0.0.1", a.listen), make_handler(a.upstream, os.path.expanduser(a.out))).serve_forever()


if __name__ == "__main__":
    main()
