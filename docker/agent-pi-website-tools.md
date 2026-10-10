# Website tools in the Pi executor

`agent-pi` (and its `agent-pi-go` child) includes Python 3, PyYAML,
virtual environments and Chromium. Tasks still run as the non-root `ailang`
user. No repository permissions or task policy change.

Run the repository's official build, for example `python build.py`. For extra
Python packages, create a workspace environment rather than changing system
Python:

```sh
python3 -m venv .venv
.venv/bin/python -m pip install -r requirements.txt
```

A new venv isolates system packages, including PyYAML: install the repository's
requirements in it, or use `--system-site-packages` when the build needs the
image's PyYAML and has no requirements file.

For repositories using Puppeteer, run their existing `npm ci`. The image sets
`PUPPETEER_SKIP_DOWNLOAD=true` and
`PUPPETEER_EXECUTABLE_PATH=/usr/bin/chromium`, so Puppeteer uses the installed
browser. `puppeteer-core` requires an explicit `executablePath` instead.
For browser launches in the executor container, use explicit arguments:

```js
const browser = await puppeteer.launch({
  headless: true,
  args: ['--no-sandbox', '--disable-dev-shm-usage'],
});
```

Executor jobs do not request Chromium namespace sandbox privileges; the Cloud
Run container remains the task isolation boundary. These flags are local to
the browser invocation. The
presentations repository already uses `--no-sandbox` in its PDF exporter.

Record actual build and browser results separately from source inspection.
`docker/test-agent-pi.sh` gates dev and release builds on YAML parsing, a
non-root venv/pip, and real Chromium JavaScript, screenshot and PDF probes.

References: [Puppeteer configuration](https://pptr.dev/api/puppeteer.configuration),
[installation](https://pptr.dev/guides/installation) and
[container troubleshooting](https://pptr.dev/troubleshooting).
