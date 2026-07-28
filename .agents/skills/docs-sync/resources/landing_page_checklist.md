# Landing Page Checklist

Landing pages are the first thing users see. They must be accurate and up-to-date.

## Critical Landing Pages

### 1. intro.mdx (/)

**Purpose**: First page users see, overview of AILANG

**Required Elements**:
- [ ] Current version from `STABLE_RELEASE` constant
- [ ] Working Hello World example (via raw-loader)
- [ ] Working factorial/recursion example (via raw-loader)
- [ ] Links to latest teaching prompt version
- [ ] Accurate "Core Features" list (only implemented features)
- [ ] "Coming Soon" section separate from current features

**Common Issues**:
- "Coming in v0.4" when we're at v0.5.x
- Teaching prompt references to old versions
- Embedded code examples that may be broken

**Validation**:
```bash
# Check version references
grep -n "v0\.[0-9]" docs/docs/intro.mdx

# Check raw-loader imports exist
grep -n "raw-loader" docs/docs/intro.mdx

# Verify imported examples work
ailang run --caps IO --entry main examples/runnable/hello.ail
```

---

### 2. vision.mdx

**Purpose**: Explain AILANG's design philosophy and roadmap

**Required Elements**:
- [ ] Accurate benchmark numbers (verify against latest.json)
- [ ] Current timeline with correct version statuses
- [ ] Clear separation of implemented vs planned
- [ ] Links to design docs for planned features

**Common Issues**:
- Timeline showing "In Progress" for completed versions
- Planned features described as current
- Benchmark numbers from old releases

**Validation**:
```bash
# Check timeline accuracy
grep -A5 "Timeline" docs/docs/vision.mdx

# Verify benchmark references
cat docs/static/benchmarks/latest.json | jq '.versions | keys'
```

---

### 3. why-ailang.mdx

**Purpose**: Convince users why to use AILANG

**Required Elements**:
- [ ] Accurate capability comparison table
- [ ] Working code examples
- [ ] Honest assessment of current vs future features

**Common Issues**:
- Claiming "hot-swappable logic" (not implemented)
- Claiming "safe sandbox" without caveats
- Outdated status indicators in tables

**Validation**:
```bash
# Check for unimplemented claims
grep -n "hot.*swap\|sandbox\|mod.*friendly" docs/docs/why-ailang.mdx
```

---

## Example Validation Rules

### Rule 1: All examples must use raw-loader

```mdx
// CORRECT
import HelloExample from '!!raw-loader!@site/../examples/runnable/hello.ail';
<CodeBlock language="typescript" title="examples/runnable/hello.ail">
  {HelloExample}
</CodeBlock>

// INCORRECT - embedded code will drift
```typescript
module examples/hello
export func main() -> () ! {IO} {
  println("Hello")
}
```
```

### Rule 2: Examples must be tested

Before release, run:
```bash
# Test all runnable examples
for f in examples/runnable/*.ail; do
  echo "Testing: $f"
  ailang run --caps IO --entry main "$f" || echo "FAILED: $f"
done
```

### Rule 3: Example paths must match

The import path must match an actual file:
```mdx
// This import:
import HelloExample from '!!raw-loader!@site/../examples/runnable/hello.ail';

// Requires this file to exist:
// examples/runnable/hello.ail
```

---

## Version Reference Rules

### Rule 1: Use constants, not hardcoded versions

```mdx
// CORRECT
import { STABLE_RELEASE, ACTIVE_PROMPT } from '@site/src/constants/version';

Current version: {STABLE_RELEASE}
Teaching prompt: {ACTIVE_PROMPT}

// INCORRECT
Current version: v0.4.4  // Will go stale!
```

### Rule 2: Keep constants updated

After every release:
```bash
# Get actual version
git describe --tags --abbrev=0

# Get latest prompt
ls prompts/v*.md | sort -V | tail -1

# Update constants
cat > docs/src/constants/version.js << EOF
export const STABLE_RELEASE = 'v0.5.6';
export const ACTIVE_PROMPT = 'v0.5.2';
EOF
```

---

## Pre-Publish Checklist

Before publishing website updates:

### Version Accuracy
- [ ] `docs/src/constants/version.js` matches `git describe --tags`
- [ ] All hardcoded version references updated
- [ ] Teaching prompt links point to latest version

### Example Accuracy
- [ ] All raw-loader imports point to existing files
- [ ] `make verify-examples` passes
- [ ] Example output in docs matches actual output

### Feature Claims
- [ ] No planned features described as current
- [ ] All features in "Current" section actually work
- [ ] Roadmap section has clear version targets

### Build Verification
- [ ] `cd docs && npm run build` succeeds
- [ ] No broken links (check console warnings)
- [ ] Key pages render correctly

---

## Quick Fix Commands

### Update version constants:
```bash
VERSION=$(git describe --tags --abbrev=0)
PROMPT=$(ls prompts/v*.md | sort -V | tail -1 | xargs basename | sed 's/.md//')

cat > docs/src/constants/version.js << EOF
export const STABLE_RELEASE = '$VERSION';
export const ACTIVE_PROMPT = '$PROMPT';
EOF
```

### Find stale version references:
```bash
# Find all version references
grep -rn "v0\.[0-9]\+\.[0-9]\+" docs/docs/ | grep -v node_modules
```

### Test a specific example:
```bash
ailang run --caps IO --entry main examples/runnable/hello.ail
```
