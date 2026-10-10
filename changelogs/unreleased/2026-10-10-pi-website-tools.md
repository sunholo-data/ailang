### Fixed — Pi website executors lacked build and browser dependencies (2026-10-10)

Website tasks could create a PR but the official Python build failed to import
PyYAML, and presentation checks could only inspect source because no browser
was installed. The Pi image now includes Python/PyYAML, virtual environments
and Chromium with fonts. Puppeteer uses the installed browser without another
download. The existing model-free image acceptance gate now parses YAML,
creates a venv with pip as the runtime user, executes page JavaScript, renders
a PNG and prints a PDF. See `docker/agent-pi-website-tools.md` for task usage.
