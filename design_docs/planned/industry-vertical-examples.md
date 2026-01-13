# Industry Vertical Example Projects for AILANG

**Status:** Planned
**Target:** v0.7.0 (Public Website Launch)
**Priority:** P0 (High)
**Estimated:** 4 weeks (distributed across public GitHub repos)
**Dependencies:** Website redesign (v0.7.0), BigQuery integration (M-CLOUD-STORAGE), Module system (v0.6.3+)
**Created:** 2026-01-13

---

## Problem Statement

Currently, AILANG showcases only toy examples (hello world, factorial, ADTs). This limits its visibility in key markets:

1. **Data Engineering**: No pipeline examples → can't demonstrate data transformation
2. **AI/ML**: No LLM integration examples → looks like a toy language
3. **Web Development**: No backend examples → can't show real-world API servers
4. **Business Value**: No industry-specific use cases → hard to pitch to companies

**Current Impact:**
- Website examples don't showcase AILANG's real strengths:
  - Deterministic execution (perfect for reproducible pipelines)
  - Effect typing (transparent about side effects)
  - AI-friendly semantics (optimal for agent coding)
- Developers can't see how to use AILANG in their own projects
- Limited SEO/discoverability for industry-specific searches

**Goals:**
- Create 3 public GitHub repositories demonstrating AILANG in different industries
- Each includes ≥1 of: data pipeline, AI API calls, BigQuery, React backend
- Serve as templates for users building similar projects
- Improve SEO with industry-specific content
- Showcase AILANG's deterministic semantics in real contexts

---

## Solution Design

### Three Industry Verticals

#### 1. **Financial Services** (`ailang-fintech-pipeline`)
**Focus:** Deterministic data transformations + AI analysis

**Use Case:** Stock price analysis pipeline
- **Data Pipeline:** Fetch stock data → clean → transform → store
- **AI Integration:** Use Claude to analyze stock trends + generate trading signals
- **BigQuery:** Store historical prices, signals, predictions
- **Backend:** FastAPI/Flask serving ML predictions (Python, not AILANG)

**Key Features:**
- Deterministic AILANG modules for data transformation
- Effect types show exactly what data is read/written
- BigQuery integration for audit trail
- Reproducible analysis (same input = same output)

**Example Workflow:**
```ailang
-- AILANG deterministic pipeline
module fintech/price_analyzer

import std/json (decode, encode)
import std/io (readLine)

-- Deterministic transformation (no side effects)
func analyzePrice(price: float, movingAvg: float) -> string {
  if price > movingAvg * 1.05 then
    "BUY"
  else if price < movingAvg * 0.95 then
    "SELL"
  else
    "HOLD"
}

-- Effect-typed I/O (transparent about what we do)
func fetchAndAnalyze(ticker: string) -> string ! {IO, Net} {
  let response = httpGet(concat_String("https://api.example.com/prices/", ticker));
  let price = parsePrice(response);
  -- Signal handler does AI analysis (Claude)
  analyzePrice(price, movingAvg)
}
```

#### 2. **Healthcare** (`ailang-healthcare-system`)
**Focus:** Data privacy + deterministic validation

**Use Case:** Patient health record processing + analysis
- **Data Pipeline:** Ingest HL7/FHIR → validate → anonymize → analyze
- **AI Integration:** Claude analyzes patient records for insights (with compliance)
- **BigQuery:** Store anonymized data for epidemiology research
- **Backend:** GraphQL API exposing sanitized records (Python/Node.js)

**Key Features:**
- Effect types enforce data access policies
- Deterministic anonymization (HIPAA compliance)
- Audit trails built into AILANG execution traces
- Type-safe schema validation

**Example Workflow:**
```ailang
module healthcare/fhir_processor

-- Effect type shows exactly what we access
func validateFHIRRecord(record: string) -> bool ! {IO} {
  -- AILANG validates deterministically
  let parsed = parseFHIR(record);
  requiredFieldsPresent(parsed)
}

func anonymizeRecord(record: FHIRRecord) -> FHIRRecord ! {IO} {
  -- Remove PII, keep clinical data
  -- AI handler (Claude) determines what's safe to keep
  {
    patientId: generateUUID(),
    age: record.age,
    conditions: record.conditions
  }
}
```

#### 3. **E-Commerce** (`ailang-ecommerce-analytics`)
**Focus:** Real-time data streams + React frontend

**Use Case:** Product recommendation engine + analytics dashboard
- **Data Pipeline:** Process purchase events → aggregate → generate recommendations
- **AI Integration:** Claude generates product descriptions + analyzes customer sentiment
- **BigQuery:** Store events, products, recommendations, metrics
- **Backend:** Node.js/Python API server querying BigQuery + calling Claude
- **Frontend:** React dashboard showing recommendations, trends, analytics

**Key Features:**
- AILANG analyzes and transforms event streams
- Deterministic recommendation algorithm (reproducible)
- Real-time dashboard with React frontend
- BigQuery integration for analytics

**Example Workflow:**
```ailang
module ecommerce/recommendation_engine

-- Pure recommendation algorithm (deterministic)
func scoreProduct(product: Product, userHistory: [Product]) -> float {
  let similarity = calculateSimilarity(product, userHistory);
  let popularity = getPopularity(product);
  similarity * 0.7 + popularity * 0.3
}

-- Stream processing with effects
func processEventsAndRecommend(events: [PurchaseEvent]) -> [Recommendation] ! {IO, Net} {
  let aggregated = aggregateEvents(events);
  let scores = map(scoreProduct(_, aggregated.userHistory), allProducts);
  let topProducts = sortByScore(scores);

  -- AI handler generates descriptions via Claude
  generateDescriptions(topProducts)
}
```

### Repository Structure

Each repository follows this pattern:

```
ailang-INDUSTRY-VERTICAL/
├── .github/
│   ├── workflows/
│   │   ├── test.yml
│   │   ├── deploy.yml
│   │   └── bigquery-sync.yml
│   └── ISSUE_TEMPLATE/
│       └── feature_request.md
├── ailang/                          # AILANG deterministic modules
│   ├── data_pipeline.ail            # Core transformation logic
│   ├── validation.ail               # Schema/data validation
│   ├── analysis.ail                 # Analysis algorithms
│   └── effects_demo.ail             # Effect types showcase
├── backend/                         # Python/Node.js API server
│   ├── api/
│   │   ├── routes.py
│   │   └── models.py
│   ├── bigquery/
│   │   ├── schema.sql
│   │   └── migrations/
│   └── ai_handlers/
│       └── claude_integration.py    # Calls Claude for AI tasks
├── frontend/                        # React dashboard (ecommerce only)
│   ├── src/
│   │   ├── components/
│   │   ├── pages/
│   │   └── services/
│   └── package.json
├── data/                            # Sample data
│   ├── sample.json
│   └── schema.json
├── examples/
│   ├── simple.ail                   # Hello world equivalent
│   └── full_pipeline.ail            # Complete example
├── README.md                        # Setup + usage
├── ARCHITECTURE.md                  # Design decisions
├── Makefile                         # Build automation
└── docker-compose.yml               # Local dev environment

# Key files for website
├── docs/
│   ├── tutorial.md                  # Getting started
│   ├── data-pipeline.md             # Data transformation guide
│   ├── ai-integration.md            # How to call Claude/GPT
│   ├── bigquery-setup.md            # BigQuery configuration
│   └── screenshots/                 # Demo screenshots
└── DEMO_GUIDE.md                    # How to run locally
```

### Implementation Plan

#### Phase 1: Foundation (Week 1)
**Goal:** Create repositories and basic structure

- [ ] Create 3 GitHub repositories (private initially)
- [ ] Set up issue templates and contribution guidelines
- [ ] Add GitHub Actions for CI/CD
- [ ] Create Makefile with standard targets (build, test, deploy)
- [ ] Add Docker Compose for local development
- [ ] Write skeleton README for each repo

**Files:** ~50 files, ~2,000 LOC (mostly config)

#### Phase 2: AILANG Modules (Week 2)
**Goal:** Implement core deterministic logic

- [ ] **Fintech:** `price_analyzer.ail`, `signal_generator.ail`, `data_cleaner.ail`
- [ ] **Healthcare:** `fhir_processor.ail`, `anonymizer.ail`, `validator.ail`
- [ ] **E-Commerce:** `recommendation_engine.ail`, `event_processor.ail`, `scorer.ail`
- [ ] Add comprehensive examples showing effect types
- [ ] Write tests for AILANG modules (`ailang test`)
- [ ] Document AILANG code with inline examples

**Files:** ~30 AILANG modules, ~3,000 LOC total

**Key Insight:** Use AILANG for **deterministic core logic only**:
- ✅ Transformations (pure functions)
- ✅ Validation (exhaustive pattern matching)
- ✅ Analysis algorithms (reproducible scoring)
- ❌ NOT I/O, NOT AI calls (those go in Python/Node.js handlers)

#### Phase 3: Backend Integration (Week 3)
**Goal:** Connect AILANG to API servers and databases

- [ ] **Fintech:** Python Flask server
  - Endpoint: `POST /analyze` → calls AILANG pipeline → returns signal
  - Endpoint: `GET /history` → BigQuery queries
- [ ] **Healthcare:** Python FastAPI server
  - Endpoint: `POST /validate` → calls AILANG validator
  - Endpoint: `GET /anonymized-records` → BigQuery
- [ ] **E-Commerce:** Node.js Express server
  - Endpoint: `POST /recommend` → calls AILANG engine
  - Endpoint: `GET /analytics` → BigQuery
- [ ] Implement BigQuery integration
- [ ] Add API documentation (OpenAPI/Swagger)
- [ ] Write integration tests

**Files:** ~50 backend files, ~4,000 LOC

**BigQuery Integration Pattern:**
```python
# Python example (shared across repos)
from google.cloud import bigquery

async def store_result(ailang_output: dict):
    """Store AILANG deterministic output to BigQuery"""
    client = bigquery.Client()
    table = client.get_table("project.dataset.results")
    rows_to_insert = [ailang_output]
    errors = client.insert_rows_json(table, rows_to_insert)
    # Audit trail: AILANG output → BigQuery
```

#### Phase 4: React Frontend (E-Commerce Only) (Week 3)
**Goal:** Build analytics dashboard

- [ ] Create React app with Vite
- [ ] Dashboard components (charts, tables, filters)
- [ ] Real-time data refresh (WebSocket or polling)
- [ ] BigQuery data integration
- [ ] Mobile responsive design

**Files:** ~30 React components, ~2,500 LOC

#### Phase 5: Documentation & Examples (Week 4)
**Goal:** Make it easy for developers to learn and fork

**For Each Repository:**
- [ ] Tutorial: "Building a pipeline with AILANG"
- [ ] Setup guide: Local development (Docker)
- [ ] API documentation (OpenAPI)
- [ ] AILANG code walkthrough
- [ ] Deployment guide (Cloud Run, Heroku, etc.)
- [ ] Screenshot/video demo

**Website Integration:**
- [ ] Add project cards to `/docs/examples/industry-verticals`
- [ ] Link to GitHub repos
- [ ] Add to navigation menu
- [ ] Create comparison table (AILANG vs alternatives)

**Files:** ~15 markdown docs, ~5,000 words

#### Phase 6: Public Launch & SEO (Week 4)
**Goal:** Make repositories public and discoverable

- [ ] Make repos public
- [ ] Add GitHub topics: `ailang`, `data-pipeline`, `python-integration`, etc.
- [ ] Submit to GitHub topics + trending
- [ ] Create blog posts on Sunholo blog:
  - "Building Deterministic Data Pipelines with AILANG"
  - "Why Determinism Matters for Financial Services"
  - "AILANG + Claude: AI-Powered Analytics"
- [ ] Add links from website to repos
- [ ] Create social media posts

**Timeline:** 1 week (overlaps with Phase 5)

---

## Examples

### Example 1: Fintech Pipeline Architecture

**AILANG Core (Pure, Deterministic):**
```ailang
module fintech/analysis

-- Pure transformation: no side effects
func calculateRSI(prices: [float]) -> float {
  let n = 14;
  let gains = sumPositive(calculateChanges(prices, n));
  let losses = sumNegative(calculateChanges(prices, n));
  let rs = gains / losses;
  100.0 - (100.0 / (1.0 + rs))
}

-- Result type: make success/failure explicit
func validateTradeSignal(rsi: float, price: float, threshold: float) -> Result {
  if rsi > threshold then
    Ok({signal: "BUY", confidence: rsi})
  else
    Err({reason: "RSI below threshold", value: rsi})
}
```

**Python Backend (Effects & I/O):**
```python
# backend/api/trade_signal.py
from google.cloud import bigquery
import asyncio
import subprocess
import json

async def analyze_stock(ticker: str):
    # 1. Fetch data (I/O effect)
    prices = await fetch_prices_from_api(ticker)

    # 2. Call AILANG for deterministic analysis
    ailang_result = await run_ailang_analysis(prices)

    # 3. Store in BigQuery (I/O effect)
    await store_result_to_bigquery(ticker, ailang_result)

    # 4. Call Claude for narrative analysis (AI effect)
    narrative = await claude.analyze(ailang_result)

    return {
        "ticker": ticker,
        "signal": ailang_result["signal"],
        "narrative": narrative,
        "timestamp": datetime.now()
    }

async def run_ailang_analysis(prices):
    """Execute AILANG module deterministically"""
    # Write prices to temp file
    with open("/tmp/prices.json", "w") as f:
        json.dump(prices, f)

    # Run AILANG
    result = subprocess.run(
        ["ailang", "run", "--caps", "IO,FS", "--entry", "main",
         "ailang/analysis.ail"],
        capture_output=True,
        text=True
    )

    return json.loads(result.stdout)
```

**Benefits:**
- **Determinism:** Same prices always produce same signal (reproducible)
- **Auditability:** BigQuery stores complete trace
- **Composability:** AILANG logic is pure, tests are simple
- **Transparency:** Effect types show I/O, network, AI calls clearly

### Example 2: Healthcare Record Processing

**AILANG Validation (Type-Safe):**
```ailang
module healthcare/fhir

-- ADT for valid record states
type PatientRecord =
  | Valid(patient: Patient, observations: [Observation])
  | Invalid(reason: string)

-- Deterministic validation
func validateFHIR(json: string) -> PatientRecord {
  match parseFHIR(json) {
    Ok(record) =>
      if hasRequiredFields(record) then
        Valid(record.patient, record.observations)
      else
        Invalid("Missing required FHIR fields")
    Err(msg) =>
      Invalid(concat_String("Parse error: ", msg))
  }
}

-- Anonymization (deterministic, reproducible)
func anonymize(record: PatientRecord) -> PatientRecord {
  match record {
    Valid(patient, obs) =>
      let anon_patient = {
        id: generateDeterministicID(patient.ssn),
        age: patient.dob_to_age(),
        gender: patient.gender
      };
      Valid(anon_patient, obs)
    Invalid(reason) =>
      Invalid(reason)
  }
}
```

**Python Backend (Compliance):**
```python
# backend/api/patient_handler.py
from google.cloud import bigquery
import subprocess

async def process_hl7_record(hl7_data: str):
    # 1. Validate with AILANG (deterministic)
    validation = await run_fhir_validation(hl7_data)

    if validation["status"] == "invalid":
        return {"error": validation["reason"]}

    # 2. Anonymize with AILANG (deterministic)
    anonymized = await run_anonymization(validation["record"])

    # 3. Store to BigQuery (immutable audit trail)
    await bigquery_client.insert_rows_json(
        table="healthcare.patients_anonymized",
        rows=[{
            "ailang_output": anonymized,
            "processed_at": datetime.now().isoformat(),
            "version": "v1.0"  # For reproducibility
        }]
    )

    return {"status": "success"}
```

**Benefits:**
- **HIPAA Compliance:** Deterministic anonymization, audit trails
- **Reproducibility:** Same record always produces same anonymized version
- **Type Safety:** Pattern matching ensures all cases handled

### Example 3: E-Commerce Dashboard Architecture

**AILANG Recommendation Engine:**
```ailang
module ecommerce/recommendations

-- Scoring algorithm (deterministic)
func scoreProduct(product: Product, user: UserProfile) -> float {
  let category_match = if memberOf(product.category, user.interests) then 1.0 else 0.5;
  let price_match = 1.0 - (abs(product.price - user.avg_spend) / user.avg_spend);
  let popularity = product.rating / 5.0;

  category_match * 0.4 + price_match * 0.3 + popularity * 0.3
}

-- Top-K recommendations (with confidence scores)
func recommendProducts(products: [Product], user: UserProfile) -> [Recommendation] {
  let scored = map(\p. {product: p, score: scoreProduct(p, user)}, products);
  let sorted = sortBy(\r. r.score, scored);
  take(10, sorted)  -- Top 10
}
```

**Node.js Backend + React:**
```javascript
// backend/api/recommendations.ts
import { BigQuery } from "@google-cloud/bigquery";

async function getRecommendations(userId: string) {
  // 1. Fetch user profile from BigQuery
  const user = await bigquery.query(
    `SELECT * FROM ecommerce.users WHERE id = @userId`,
    { userId }
  );

  // 2. Call AILANG for deterministic scoring
  const ailangRecommendations = await callAilang({
    products: allProducts,
    user: user[0]
  });

  // 3. Enrich with Claude-generated descriptions
  const enriched = await Promise.all(
    ailangRecommendations.map(async (rec) => ({
      ...rec,
      description: await claude.generateDescription(rec.product),
      personalized_reason: await claude.generateReason(rec, user)
    }))
  );

  // 4. Store to BigQuery for analytics
  await bigquery.table("ecommerce.recommendations").insert([{
    user_id: userId,
    recommendations: enriched,
    created_at: new Date(),
    algorithm_version: "ailang-v1.0"  // Reproducibility
  }]);

  return enriched;
}
```

**React Dashboard:**
```jsx
// frontend/src/pages/Dashboard.tsx
import { useEffect, useState } from "react";
import { BigQuery } from "@google-cloud/bigquery";

export function Dashboard() {
  const [recommendations, setRecommendations] = useState([]);
  const [trends, setTrends] = useState([]);

  useEffect(() => {
    // Real-time recommendations
    const interval = setInterval(async () => {
      const rec = await fetch("/api/recommendations/me").then(r => r.json());
      setRecommendations(rec);
    }, 5000);

    // Analytics dashboard
    const trends = await fetch("/api/analytics/trends").then(r => r.json());
    setTrends(trends);

    return () => clearInterval(interval);
  }, []);

  return (
    <div className="dashboard">
      <h1>Personalized Recommendations</h1>
      <ProductGrid products={recommendations} />

      <h2>Analytics</h2>
      <TrendChart data={trends} />
    </div>
  );
}
```

**Benefits:**
- **Deterministic Scoring:** Same user always gets same recommendations (reproducible)
- **Real-Time Dashboard:** React shows latest data + AI insights
- **Auditability:** BigQuery stores all recommendations for analysis

---

## Success Criteria

### Metrics to Track

1. **Code Quality**
   - [ ] All AILANG code passes linting (`make lint`)
   - [ ] Test coverage > 80% for AILANG modules
   - [ ] Backend integration tests pass in CI
   - [ ] Frontend passes accessibility checks (a11y)

2. **Documentation**
   - [ ] README has clear setup instructions (<5 min to run locally)
   - [ ] Every AILANG function documented with examples
   - [ ] Tutorial is completable in 30 minutes
   - [ ] API documentation is 100% complete

3. **Usability**
   - [ ] Each repo has a `make demo` target that runs example
   - [ ] Docker Compose starts all services with one command
   - [ ] Setup script handles API key configuration
   - [ ] Error messages guide users to solutions

4. **Website Integration**
   - [ ] Projects appear on `/docs/examples/industry-verticals`
   - [ ] Each project has a dedicated page with screenshots
   - [ ] At least 100 GitHub stars combined across 3 repos
   - [ ] Repositories appear in GitHub trending (optional)

5. **SEO & Discovery**
   - [ ] Blog posts published to Sunholo blog
   - [ ] Social media posts reach >5K impressions each
   - [ ] Google search results for "AILANG data pipeline" show repos
   - [ ] Repos indexed in awesome-lists (Python, Node.js, Finance, Healthcare)

### Acceptance Tests

- [ ] All 3 repositories are public and fully functional
- [ ] Each runs locally with `docker-compose up && make demo`
- [ ] AILANG modules are well-documented and reusable
- [ ] Backends successfully call AILANG and BigQuery
- [ ] E-Commerce repo has working React dashboard
- [ ] Documentation covers setup, architecture, and contribution
- [ ] Website updated with links and screenshots
- [ ] No critical security issues (pass OWASP top 10)

---

## Timeline

| Week | Fintech | Healthcare | E-Commerce | Website |
|------|---------|-----------|-----------|---------|
| **1** | Setup repos | Setup repos | Setup repos | Plan |
| **2** | AILANG modules | AILANG modules | AILANG + React | Design |
| **3** | Backend | Backend | Backend | Content |
| **4** | Testing + docs | Testing + docs | Testing + docs | Launch |

**Total Effort:** 4 weeks, 1 FTE (full-time engineer)

**Can be parallelized:** All 3 repos can be built simultaneously if 3 engineers available → 2 weeks

---

## Dependencies

### Must-Have (Blockers)

1. **Website v0.7.0 redesign** - Need new `/docs/examples/` section
2. **AILANG v0.6.3+ maturity** - Modules must work reliably
3. **Documentation** - `ailang prompt` must be up to date

### Nice-to-Have

1. **BigQuery integration guide** - M-CLOUD-STORAGE (v0.7.0)
2. **Cloud deployment docs** - Cloud Run, Heroku guides
3. **Performance benchmarks** - For comparison with Python

---

## Related Documents

- [M-CLOUD-STORAGE](./m-cloud-storage.md) - BigQuery/Firestore integration
- [design_docs/implemented/v0_3_14/](../implemented/v0_3_14/) - Stdlib + examples
- [design_docs/implemented/v0_6_3/](../implemented/v0_6_3/) - Module system maturity
- [README.md](../../README.md) - Current example status

---

## Risks & Mitigations

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|-----------|
| AILANG modules too slow | Projects seem inefficient | Medium | Use profiling, optimize early |
| BigQuery costs high | Project becomes expensive | Low | Use simulator for dev, cache queries |
| React dashboard complexity | Takes too long | Medium | Use existing dashboard template |
| Attracting users fails | Low GitHub stars | Low | Market aggressively on Twitter/Reddit |

---

## Future Extensions

**Phase 2 (v0.8.0+):**
- Add SQL Query Builder example (AILANG → SQL generation)
- Add time-series analysis example (financial data)
- Add mobile React Native version of e-commerce app
- Add Kubernetes deployment guides

**Phase 3 (v0.9.0+):**
- Add ML model serving example (AILANG + TensorFlow)
- Add real-time stream processing (Kafka, Pub/Sub)
- Add GraphQL federation across multiple services

---

## Author Notes

**Key Insight:** Don't try to build everything in AILANG. AILANG excels at:
- ✅ **Deterministic transformations** - pure functions
- ✅ **Schema validation** - pattern matching
- ✅ **Type-safe analysis** - algebraic types

But use existing ecosystems for:
- ❌ **Web servers** - Use Python/Node.js (mature, fast)
- ❌ **UI** - Use React/Vue (mature, polished)
- ❌ **Database operations** - Use SQLAlchemy/ORM (well-tested)

**The sweet spot:** Call AILANG as a subprocess for core logic, chain effects together in Python/Node.js.

**Axiom Compliance:**
- A1 (Determinism): ✅ AILANG modules are pure, deterministic
- A3 (Effect Legibility): ✅ Python/Node.js backends show all effects clearly
- A7 (Machines First): ✅ Structured JSON for AI analysis
- A9 (Cost Visibility): ✅ BigQuery shows exact query costs
- A12 (System Boundary): ✅ Clear separation between AILANG core and external services
