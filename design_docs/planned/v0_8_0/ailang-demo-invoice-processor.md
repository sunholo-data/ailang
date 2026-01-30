# AILANG Demo: Invoice Processor with WebAssembly

**Status**: Planned
**Target Repo**: `ailang_demo` (separate repository)
**Deployment**: GitHub Pages
**Timeline**: 2 weeks

---

## Goal

Create a standalone business application demo that showcases AILANG's WebAssembly capabilities by building a real-world invoice validation and processing system. The demo will emphasize type safety catching errors that would cause runtime crashes in JavaScript/Python.

---

## User Requirements

- **Application Type**: Invoice/Document Processor
- **User Interaction**: Upload/paste JSON invoices, see validation results + calculations
- **Key Message**: Type safety catches errors before production
- **Deployment**: GitHub Pages (static site, no server required)
- **Repository**: `ailang_demo` (separate from main AILANG repo)

---

## Application Overview

**Invoice Processor** - A business application that:
1. Accepts invoice data (paste JSON or upload file)
2. Validates structure and data types using AILANG's type system
3. Calculates totals, taxes, discounts with type-safe arithmetic
4. Detects common errors (missing fields, wrong types, invalid amounts)
5. Displays results with clear error messages

**Key Differentiator**: Shows how AILANG's type system catches errors **at parse time** that would cause runtime crashes in JavaScript/Python.

---

## Repository Structure

```
ailang_demo/
├── index.html                 # Main demo page (~250 LOC)
├── css/
│   └── styles.css            # UI styling (~300 LOC)
├── js/
│   ├── app.js                # Main application logic (~300 LOC)
│   ├── ailang-wrapper.js     # WASM integration layer (~150 LOC)
│   └── examples.js           # Sample invoice data (~100 LOC)
├── wasm/
│   ├── ailang.wasm           # AILANG interpreter (from main repo releases)
│   ├── wasm_exec.js          # Go WASM runtime (from main repo releases)
│   └── invoice_processor.ail # Business logic in AILANG (~200 LOC)
├── assets/
│   ├── logo.svg              # AILANG logo
│   └── demo-screenshot.png   # For README
├── README.md                 # Setup and deployment instructions (~100 LOC)
└── .github/
    └── workflows/
        └── deploy.yml        # Auto-deploy to GitHub Pages (~40 LOC)
```

**Total new code**: ~1,440 LOC
**Assets from main repo**: `ailang.wasm`, `wasm_exec.js` (downloaded from releases)

---

## Core AILANG Module

### File: `wasm/invoice_processor.ail` (~200 LOC)

This module contains the business logic written in AILANG:

```ailang
module invoice_processor

-- Invoice line item type
type LineItem = {
  description: string,
  quantity: int,
  unit_price: float,
  tax_rate: float
}

-- Invoice type
type Invoice = {
  invoice_number: string,
  customer_name: string,
  date: string,
  line_items: [LineItem],
  discount_percent: float
}

-- Result type for validation
type ValidationResult =
  | Valid(Invoice)
  | Invalid(string)

-- Calculate line item total
pure func lineItemTotal(item: LineItem) -> float =
  let subtotal = float(item.quantity) * item.unit_price;
  let tax = subtotal * item.tax_rate;
  subtotal + tax

-- Calculate invoice subtotal (before discount)
pure func calculateSubtotal(items: [LineItem]) -> float =
  match items {
    [] => 0.0,
    item :: rest => lineItemTotal(item) + calculateSubtotal(rest)
  }

-- Calculate final total with discount
pure func calculateTotal(invoice: Invoice) -> float =
  let subtotal = calculateSubtotal(invoice.line_items);
  let discount = subtotal * (invoice.discount_percent / 100.0);
  subtotal - discount

-- Validate invoice structure and data
pure func validateInvoice(data: {invoice_number: string, customer_name: string,
                                 date: string, line_items: [{description: string,
                                 quantity: int, unit_price: float,
                                 tax_rate: float}], discount_percent: float})
  -> ValidationResult =
  -- Check required fields
  if data.invoice_number == "" then
    Invalid("Missing invoice_number")
  else if data.customer_name == "" then
    Invalid("Missing customer_name")
  else if data.discount_percent < 0.0 || data.discount_percent > 100.0 then
    Invalid("Discount must be between 0 and 100")
  else
    match data.line_items {
      [] => Invalid("Invoice must have at least one line item"),
      _ => validateLineItems(data.line_items, 1)
    }

-- Validate all line items
pure func validateLineItems(items: [LineItem], lineNum: int) -> ValidationResult =
  match items {
    [] => Valid(...), -- Create valid invoice
    item :: rest =>
      if item.quantity <= 0 then
        Invalid("Line " ++ intToString(lineNum) ++ ": quantity must be positive")
      else if item.unit_price < 0.0 then
        Invalid("Line " ++ intToString(lineNum) ++ ": price cannot be negative")
      else if item.tax_rate < 0.0 || item.tax_rate > 1.0 then
        Invalid("Line " ++ intToString(lineNum) ++ ": tax rate must be 0-1")
      else
        validateLineItems(rest, lineNum + 1)
  }

-- Main processing function (exposed to JavaScript)
export func processInvoice(jsonString: string) -> string =
  -- Parse JSON, validate, calculate totals
  -- Returns JSON result with validation errors or totals
  ...
```

**Key Features Demonstrated:**
- ✅ **Type Safety**: Record types ensure required fields
- ✅ **Pattern Matching**: ADTs for Result type, list processing
- ✅ **Recursion**: Process line items recursively
- ✅ **Pure Functions**: No side effects, testable logic
- ✅ **Error Handling**: Explicit validation with helpful messages

---

## JavaScript Integration

### File: `js/ailang-wrapper.js` (~150 LOC)

Wraps AILANG WASM and provides high-level API:

```javascript
class InvoiceProcessor {
  constructor() {
    this.ready = false;
    this.wasmModule = null;
  }

  async init() {
    // Load WASM module
    const go = new Go();
    const result = await WebAssembly.instantiateStreaming(
      fetch('wasm/ailang.wasm'),
      go.importObject
    );
    go.run(result.instance);

    // Load invoice processor module
    await this.loadModule('wasm/invoice_processor.ail');
    this.ready = true;
  }

  async processInvoice(invoiceData) {
    if (!this.ready) throw new Error('Not initialized');

    // Call AILANG function via WASM
    const jsonInput = JSON.stringify(invoiceData);
    const result = window.ailangEval(
      `processInvoice("${escapeJson(jsonInput)}")`
    );

    return JSON.parse(result);
  }
}
```

### File: `js/app.js` (~300 LOC)

Main application logic:

```javascript
// Initialize processor
const processor = new InvoiceProcessor();
await processor.init();

// Handle form submission
document.getElementById('processBtn').addEventListener('click', async () => {
  const input = document.getElementById('invoiceInput').value;

  try {
    const invoiceData = JSON.parse(input);
    const result = await processor.processInvoice(invoiceData);

    if (result.valid) {
      displaySuccess(result);
    } else {
      displayErrors(result.errors);
    }
  } catch (e) {
    displayError('Invalid JSON: ' + e.message);
  }
});
```

---

## User Interface

### File: `index.html` (~250 LOC)

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>AILANG Invoice Processor Demo</title>
  <link rel="stylesheet" href="css/styles.css">
  <script src="wasm/wasm_exec.js"></script>
</head>
<body>
  <header>
    <h1>🧾 AILANG Invoice Processor</h1>
    <p>Type-safe business logic with WebAssembly</p>
  </header>

  <main>
    <!-- Split panel UI -->
    <div class="panel-container">
      <!-- Left: Input -->
      <div class="input-panel">
        <h2>Invoice Data (JSON)</h2>
        <div class="examples">
          <button data-example="valid">✅ Valid Invoice</button>
          <button data-example="missing-field">❌ Missing Field</button>
          <button data-example="negative-price">❌ Negative Price</button>
          <button data-example="invalid-discount">❌ Invalid Discount</button>
        </div>
        <textarea id="invoiceInput" rows="20"></textarea>
        <button id="processBtn" class="primary">Process Invoice</button>
      </div>

      <!-- Right: Output -->
      <div class="output-panel">
        <h2>Results</h2>
        <div id="results">
          <p class="placeholder">Paste invoice data and click "Process Invoice"</p>
        </div>
      </div>
    </div>

    <!-- Feature highlights -->
    <section class="features">
      <h2>Why AILANG for Business Logic?</h2>
      <div class="feature-grid">
        <div class="feature">
          <h3>🛡️ Type Safety</h3>
          <p>Catch errors at compile time, not in production</p>
        </div>
        <div class="feature">
          <h3>⚡ Fast</h3>
          <p>Runs entirely in browser via WebAssembly</p>
        </div>
        <div class="feature">
          <h3>🔧 Maintainable</h3>
          <p>Clear, declarative code with pattern matching</p>
        </div>
        <div class="feature">
          <h3>📦 Portable</h3>
          <p>Same code runs client-side or server-side</p>
        </div>
      </div>
    </section>
  </main>

  <script type="module" src="js/app.js"></script>
</body>
</html>
```

---

## Example Invoices

### File: `js/examples.js` (~100 LOC)

```javascript
export const examples = {
  valid: {
    invoice_number: "INV-2024-001",
    customer_name: "Acme Corp",
    date: "2024-01-15",
    line_items: [
      {
        description: "Widget A",
        quantity: 10,
        unit_price: 25.00,
        tax_rate: 0.08
      },
      {
        description: "Service B",
        quantity: 5,
        unit_price: 100.00,
        tax_rate: 0.08
      }
    ],
    discount_percent: 10.0
  },

  "missing-field": {
    invoice_number: "",  // ❌ Empty required field
    customer_name: "Test Co",
    date: "2024-01-15",
    line_items: [],
    discount_percent: 0
  },

  "negative-price": {
    invoice_number: "INV-002",
    customer_name: "Test Co",
    date: "2024-01-15",
    line_items: [
      {
        description: "Item",
        quantity: 1,
        unit_price: -50.00,  // ❌ Negative price
        tax_rate: 0.08
      }
    ],
    discount_percent: 0
  },

  "invalid-discount": {
    invoice_number: "INV-003",
    customer_name: "Test Co",
    date: "2024-01-15",
    line_items: [
      {
        description: "Item",
        quantity: 1,
        unit_price: 100.00,
        tax_rate: 0.08
      }
    ],
    discount_percent: 150.0  // ❌ Discount > 100%
  }
};
```

---

## Deployment: GitHub Pages

### File: `.github/workflows/deploy.yml` (~40 LOC)

```yaml
name: Deploy to GitHub Pages

on:
  push:
    branches: [main]
  workflow_dispatch:

permissions:
  contents: read
  pages: write
  id-token: write

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Download AILANG WASM
        run: |
          # Get latest release from main AILANG repo
          wget https://github.com/sunholo-data/ailang/releases/latest/download/ailang.wasm \
            -O wasm/ailang.wasm
          wget https://github.com/sunholo-data/ailang/releases/latest/download/wasm_exec.js \
            -O wasm/wasm_exec.js

      - name: Setup Pages
        uses: actions/configure-pages@v4

      - name: Upload artifact
        uses: actions/upload-pages-artifact@v3
        with:
          path: '.'

      - name: Deploy to GitHub Pages
        uses: actions/deploy-pages@v4
```

**Result**: Demo automatically deploys to `https://[org].github.io/ailang_demo/`

---

## README.md Template

### File: `README.md` (~100 LOC)

```markdown
# AILANG Invoice Processor Demo

A production-ready invoice validation and processing application powered by AILANG WebAssembly.

## 🎯 What This Demonstrates

- **Type Safety**: AILANG's type system catches errors at compile time
- **Business Logic**: Complex calculations with pattern matching and recursion
- **WebAssembly**: Full AILANG interpreter running in the browser
- **Zero Server**: Entire application runs client-side

## 🚀 Live Demo

**[Try it now →](https://sunholo-data.github.io/ailang_demo/)**

## 📦 Local Development

```bash
# Clone repo
git clone https://github.com/sunholo-data/ailang_demo.git
cd ailang_demo

# Serve locally
python3 -m http.server 8000
# Open http://localhost:8000
```

## 🏗️ How It Works

1. **User pastes JSON invoice** into textarea
2. **AILANG validates structure** using type system
3. **Calculates totals** with type-safe arithmetic
4. **Returns results** with detailed error messages

## 🧠 AILANG Code

The business logic is written in AILANG (`wasm/invoice_processor.ail`):
- Pattern matching for list processing
- ADTs for validation results
- Pure functions for testability
- Type-safe record access

## 📚 Learn More

- [AILANG Documentation](https://ailang-docs.com)
- [Main Repository](https://github.com/sunholo-data/ailang)
- [WASM Integration Guide](https://ailang-docs.com/docs/guides/wasm-integration)
```

---

## Implementation Plan

### Phase 1: Core Functionality (Week 1)

**Goal**: Working invoice processor with basic UI

**Tasks**:
1. Set up `ailang_demo` repository (if not already exists)
2. Write `invoice_processor.ail` module
3. Test AILANG code locally with `ailang run`
4. Create basic HTML/CSS UI
5. Integrate WASM with JavaScript wrapper
6. Add 4 example invoices

**Deliverable**: Working demo that validates invoices and calculates totals

**Files to Create**:
- `wasm/invoice_processor.ail`
- `index.html`
- `css/styles.css`
- `js/app.js`
- `js/ailang-wrapper.js`
- `js/examples.js`

---

### Phase 2: Polish & Deploy (Week 2)

**Goal**: Production-ready demo on GitHub Pages

**Tasks**:
1. Improve UI styling (professional, modern)
2. Add loading states and animations
3. Write comprehensive README
4. Set up GitHub Actions for auto-deployment
5. Add analytics (optional)
6. Create demo video/GIF

**Deliverable**: Live demo at `https://sunholo-data.github.io/ailang_demo/`

**Files to Create**:
- `.github/workflows/deploy.yml`
- `README.md`
- `assets/logo.svg`
- `assets/demo-screenshot.png`

---

### Phase 3: Advanced Features (Optional)

**Goal**: Show more AILANG capabilities

**Tasks**:
1. Add more validation rules (date formats, tax calculations)
2. Export results to PDF (using client-side library)
3. Comparison view (JavaScript vs AILANG code side-by-side)
4. Performance metrics display

---

## Key Selling Points

### For the Demo Page

**Hero Section**:
> "Business logic that catches errors **before** they reach production"
>
> See how AILANG's type system validates invoice data with zero runtime overhead.

**Comparison Box**:
```
❌ JavaScript/Python:
- Missing fields crash at runtime
- Type errors discovered by users
- No guarantees about data structure

✅ AILANG:
- Type system ensures all fields present
- Errors caught at compile time
- Guaranteed data structure correctness
```

**Call-to-Action**:
- "Try the Live Demo"
- "View Source Code"
- "Read Integration Guide"
- "Download AILANG"

---

## Technical Details

### WASM Integration

**How it works**:
1. User loads page → Browser fetches `ailang.wasm` (~2MB gzipped)
2. Go runtime (`wasm_exec.js`) initializes WebAssembly
3. AILANG REPL starts, loads `invoice_processor.ail`
4. JavaScript calls `ailangEval("processInvoice(...)")` via WASM bridge
5. AILANG validates, calculates, returns JSON string
6. JavaScript parses JSON and displays results

**No server required** - everything runs client-side:
- WASM binary is cached by browser
- Works offline after first load
- Zero API costs
- Instant feedback (< 100ms processing)

### Type Safety Examples

**Missing Field Error**:
```
❌ JavaScript: invoice.customer_name undefined (runtime crash)
✅ AILANG: "Missing customer_name" (validation error)
```

**Wrong Type Error**:
```
❌ JavaScript: NaN (quantity * price with string quantity)
✅ AILANG: Type error - expected int, got string (compile time)
```

**Invalid Data Error**:
```
❌ JavaScript: Negative total (no validation)
✅ AILANG: "Line 1: price cannot be negative" (validation error)
```

---

## Success Metrics

- ✅ Demo loads in < 3 seconds
- ✅ All 4 example invoices work correctly
- ✅ Type errors are clearly displayed
- ✅ UI is professional and responsive
- ✅ README explains value proposition
- ✅ Deploys automatically via GitHub Actions

---

## Files Summary

### To Create in `ailang_demo` Repo

| File | LOC | Purpose |
|------|-----|---------|
| `index.html` | ~250 | Main UI |
| `css/styles.css` | ~300 | Styling |
| `js/app.js` | ~300 | Application logic |
| `js/ailang-wrapper.js` | ~150 | WASM integration |
| `js/examples.js` | ~100 | Sample data |
| `wasm/invoice_processor.ail` | ~200 | **Core business logic** |
| `.github/workflows/deploy.yml` | ~40 | Auto-deployment |
| `README.md` | ~100 | Documentation |

**Total**: ~1,440 LOC

### To Copy from Main AILANG Repo

1. `wasm/ailang.wasm` - Binary from latest release
2. `wasm/wasm_exec.js` - Go runtime from latest release

---

## Next Steps

1. Create design doc in main AILANG repo ✅
2. User implements in `ailang_demo` repo
3. Test with sample invoices
4. Deploy to GitHub Pages
5. Share demo link!

---

## References

- AILANG WASM docs: `docs/docs/guides/wasm-integration.md`
- Existing playground: `docs/docs/playground.mdx`
- WASM build: `make build-wasm` in main repo
- Release workflow: `.github/workflows/release.yml`
