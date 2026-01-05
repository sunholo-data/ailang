# Gemini CLI Reference Guide

## Installation

### Prerequisites
- Node.js v20 or higher (v22 recommended)
- Google Cloud project with Cloud Trace API enabled
- Google Cloud credentials (ADC or service account)

### Install via nvm (Recommended)
```bash
# Install Node v22
nvm install 22
nvm use 22

# Install Gemini CLI globally
npm install -g @google/gemini-cli

# Verify installation
gemini --version
```

### Install via Homebrew
```bash
brew install node@22
/opt/homebrew/opt/node@22/bin/npm install -g @google/gemini-cli
```

## Path Configuration

### Finding the Correct Paths

```bash
# Find Node installation
which node
# Example: /Users/mark/.nvm/versions/node/v22.20.0/bin/node

# Find Gemini CLI
which gemini
# Example: /Users/mark/.nvm/versions/node/v22.20.0/bin/gemini

# Check symlink target
ls -la $(which gemini)
# Example: -> ../lib/node_modules/@google/gemini-cli/dist/index.js
```

### Full Path Pattern

For nvm installations:
```
Node: ~/.nvm/versions/node/v22.x.x/bin/node
Gemini: ~/.nvm/versions/node/v22.x.x/lib/node_modules/@google/gemini-cli/dist/index.js
```

For Homebrew installations:
```
Node: /opt/homebrew/opt/node@22/bin/node
Gemini: /opt/homebrew/opt/node@22/lib/node_modules/@google/gemini-cli/dist/index.js
```

## GCP Telemetry Configuration

### Environment Variables
```bash
# Required: GCP project for Cloud Trace export
export GOOGLE_CLOUD_PROJECT=multivac-internal-dev

# Optional: Use a different project for telemetry
export OTLP_GOOGLE_CLOUD_PROJECT=my-telemetry-project
```

### How Telemetry Works

1. **Gemini CLI** exports traces to GCP Cloud Trace
2. **AILANG Observatory** imports traces from GCP via composite backend
3. **Dashboard** shows unified view of local + GCP traces

### Viewing Traces

**GCP Cloud Trace Console:**
```
https://console.cloud.google.com/traces/list?project=YOUR_PROJECT
```

**AILANG Observatory:**
```bash
# List recent traces
curl -s "http://localhost:1957/api/observatory/traces?limit=20" | jq '.[] | {trace_id, service_name, source}'

# Filter by source
curl -s "http://localhost:1957/api/observatory/traces?limit=50" | jq '[.[] | select(.source == "gcp")]'
```

## Troubleshooting

### Error: Invalid regular expression flags
```
SyntaxError: Invalid regular expression flags
```

**Cause:** Using Node.js < v20. The Gemini CLI uses regex features requiring v20+.

**Fix:**
```bash
# Check current Node version
node --version

# If < v20, use full path to v22
/Users/mark/.nvm/versions/node/v22.20.0/bin/node \
  /Users/mark/.nvm/versions/node/v22.20.0/lib/node_modules/@google/gemini-cli/dist/index.js \
  --version
```

### Error: MODULE_NOT_FOUND
```
Error: Cannot find module '/path/to/bin/cli.mjs'
```

**Cause:** Using wrong path. Gemini CLI entry point changed.

**Fix:** Use `dist/index.js` not `bin/cli.mjs`:
```bash
# WRONG
.../gemini-cli/bin/cli.mjs

# CORRECT
.../gemini-cli/dist/index.js
```

### Error: No traces appearing in Observatory
1. Check `GOOGLE_CLOUD_PROJECT` is set
2. Verify server started with composite backend:
   ```bash
   head -30 ~/.ailang/logs/server.log | grep -i "composite"
   ```
3. GCP quota: 300 reads/minute limit. Cache TTL is 60s.

### Gemini CLI hangs/no output
- Default timeout is 120 seconds
- Use `--timeout` flag to extend:
  ```bash
  .claude/skills/gemini-cli-helper/scripts/gemini_run.sh "Complex prompt" --timeout 300
  ```

## CLI Options Reference

```
gemini [options]

Options:
  -p, --prompt <prompt>     Run with prompt (non-interactive)
  --output-format <format>  Output format: text, json, markdown
  --version                 Show version
  --help                    Show help

Examples:
  gemini                           # Interactive mode
  gemini -p "Hello"                # Single prompt
  gemini -p "Hello" --output-format json
```

## Integration with AILANG

### Coordinator Task Execution

The coordinator can use Gemini CLI for documentation and research tasks:

```yaml
# ~/.ailang/config.yaml
coordinator:
  agents:
    - id: docs-writer
      provider: gemini
      # Uses gemini-cli-helper skill scripts internally
```

### Trace Correlation

When Gemini executes ailang commands, traces are correlated:
1. Gemini CLI trace (source: gcp)
2. AILANG execution trace (source: local)
3. Both visible in Observatory with same time window
