# DISHA AI — Go Backend Implementation Plan

## Analysis

### 1. Frontend Feature → Backend API Mapping

| Frontend Feature | Backend Endpoint | Priority |
|---|---|---|
| ECP Score Calculator | `POST /api/v1/ecp/calculate` | P0 |
| "What If" Improvement Slider | `POST /api/v1/ecp/simulate` | P0 |
| Student Onboarding | `POST /api/v1/students` | P0 |
| Profile View | `GET /api/v1/students/:studentId` | P0 |
| Profile Update → Score Recalc | `PUT /api/v1/students/:studentId` | P1 |
| University Match Cards | `GET /api/v1/universities/match?studentId=` | P0 |
| University Detail Expansion | `GET /api/v1/universities/:universityId` | P1 |
| ROI Projection Chart | `POST /api/v1/roi/calculate` | P0 |
| Loan Offer Comparison | `GET /api/v1/loans/offers?studentId=` | P0 |
| EMI Calculator Widget | `POST /api/v1/loans/emi` | P1 |
| AI Counsellor Chat | `POST /api/v1/chat` | P0 |
| Funding Passport Card | `GET /api/v1/funding-passport/:studentId` | P1 |
| Dream Gap Analysis | `GET /api/v1/dream-gap?universityId=&studentId=` | P1 |
| Loan Application | `POST /api/v1/applications` | P1 |
| Application Status | `GET /api/v1/applications/:applicationId` | P1 |
| Health Check / Deploy Probe | `GET /api/v1/health` | P0 |

### 2. Top 3 Highest-Risk Architectural Decisions

#### Risk 1: Gemini API Streaming + SSE Relay
**Decision:** Use the official Google GenAI Go SDK `google.golang.org/genai` for streaming. The SDK provides `client.Models.GenerateContentStream()` which returns an iterator we consume chunk-by-chunk. We'll read each chunk in the HTTP handler goroutine, extract `resp.Candidates[].Content.Parts[].Text`, and write it as an SSE event to the `ResponseWriter` via `http.Flusher`. A `context.WithTimeout(ctx, 30*time.Second)` wraps the entire call. Client disconnect detection via `ctx.Done()` cancels the API call immediately.

**Model:** `gemini-2.5-flash` — available on Gemini's **FREE tier** (no billing required for hackathon). Up to ~1000 RPD, 15 RPM on the free plan.

**Fallback:** If Gemini API errors or times out, we stream one of 5 pre-written responses (keyword-matched) so the demo never shows an error screen.

#### Risk 2: Monetary Value Precision
**Decision:** All internal monetary arithmetic uses `int64` paise. The PRD's funding band calculation uses Lakhs (whole numbers), so those stay as `int` since they're rounded estimates. EMI calculations use `math.Round()` at each step boundary to avoid float drift. JSON serialization converts paise → INR `float64` only at the handler boundary via custom `MarshalJSON` or explicit conversion in response structs.

**Exception:** The `POST /api/v1/loans/emi` endpoint returns INR (not paise) per PRD spec for demo readability.

#### Risk 3: In-Memory Store Concurrency
**Decision:** Use `sync.RWMutex`-protected `map[string]*StudentRecord` (not `sync.Map`, because we need TTL cleanup and iteration). A background goroutine runs every 5 minutes to evict entries older than 24 hours. All store methods accept `context.Context` and check `ctx.Done()` before returning, even though in-memory ops are fast — this keeps the interface consistent for a future database migration.

### 3. Folder Structure

```
disha-backend/
├── cmd/
│   └── server/
│       └── main.go              # Entry point: config load, DI wiring, server start, graceful shutdown
├── internal/
│   ├── ecp/
│   │   ├── scorer.go            # ECP scoring engine — all types + Calculate() + FundingBand() + Tier()
│   │   └── scorer_test.go       # Table-driven tests: 10 profiles covering all edge cases
│   ├── matching/
│   │   ├── matcher.go           # University matching: filter → score → sort → top 8
│   │   └── types.go             # MatchedUniversity, MatchResult types
│   ├── roi/
│   │   └── calculator.go        # ROI projection: 10-year data points, break-even, gain
│   ├── loans/
│   │   ├── offers.go            # Offer filtering, ranking, EMI calc, illustrative EMI
│   │   └── types.go             # LoanOffer, EMIResult, OfferMatch types
│   ├── ai/
│   │   ├── client.go            # Google GenAI SDK streaming client wrapper (Gemini 2.5 Flash)
│   │   ├── prompt.go            # System prompt builder with full profile injection
│   │   ├── fallback.go          # 5 keyword-matched fallback responses
│   │   └── ratelimiter.go       # Per-student token bucket rate limiter
│   ├── students/
│   │   └── store.go             # Thread-safe in-memory student store with TTL
│   ├── applications/
│   │   └── store.go             # Thread-safe in-memory loan application store
│   ├── handlers/
│   │   ├── ecp.go               # POST /ecp/calculate, POST /ecp/simulate
│   │   ├── students.go          # POST/GET/PUT /students
│   │   ├── universities.go      # GET /universities/match, GET /universities/:id
│   │   ├── loans.go             # GET /loans/offers, POST /loans/emi, POST+GET /applications
│   │   ├── chat.go              # POST /chat (SSE streaming)
│   │   ├── health.go            # GET /health
│   │   ├── passport.go          # GET /funding-passport/:studentId
│   │   ├── dreamgap.go          # GET /dream-gap
│   │   └── response.go          # Envelope helper: success/error response builders
│   └── middleware/
│       ├── cors.go              # CORS headers for Next.js on Vercel
│       ├── logging.go           # Structured request logging (method, path, duration, status)
│       ├── recovery.go          # Panic recovery → sanitized 500
│       └── requestid.go         # UUID injection into request context
├── data/
│   ├── universities.json        # 50 university records (embedded via embed.FS)
│   └── loan_offers.json         # 3 NBFC offer records (embedded via embed.FS)
├── config/
│   └── config.go                # Env var loading with defaults
├── Dockerfile                   # Multi-stage: golang:1.22-alpine → alpine:3.19
├── render.yaml                  # Render.com deployment config
├── Makefile                     # build, test, run, lint, docker-build, docker-run
├── .env.example                 # All env vars documented
├── go.mod
├── go.sum
└── README.md                    # Complete setup + all endpoint cURL examples
```

**Justification:**
- `cmd/server/` — Standard Go layout. `main.go` is the only place that wires dependencies; it imports nothing from `handlers` except to register routes.
- `internal/` — Prevents external packages from importing our domain logic. Every sub-package has a single domain responsibility.
- `internal/handlers/` — Thin HTTP adapters. Each handler parses request, delegates to domain, writes response. No business logic here.
- `internal/middleware/` — Cross-cutting concerns. Composable via `chi.Use()`.
- `data/` — Embedded JSON. Using `embed.FS` means the binary is self-contained — no file system access at runtime.
- `config/` — Single source of truth for env vars. All env vars have documented defaults.
- No `pkg/` directory — nothing in this project needs to be importable by external packages.

### 4. Data Models

#### StudentProfile (input)
```go
type StudentProfile struct {
    Name              string      `json:"name"`
    CGPA              float64     `json:"cgpa"`              // 4.0–10.0
    GREScore          int         `json:"greScore"`          // 0 or 260–340
    FamilyIncome      IncomeBand  `json:"familyIncome"`      // "<3L", "3-8L", "8-20L", "20L+"
    HasCoApplicant    bool        `json:"hasCoApplicant"`
    CoApplicantIncome IncomeBand  `json:"coApplicantIncome"` // same enum, defaults "<3L"
    TargetCountry     Country     `json:"targetCountry"`
    TargetProgram     ProgramType `json:"targetProgram"`
    Intake            IntakePeriod `json:"intake"`
}
```

#### ECPResult (output)
```go
type ECPResult struct {
    Score           int             `json:"score"`
    Tier            string          `json:"tier"`              // "Green", "Amber", "Red"
    FundingBandLower int            `json:"fundingBandLower"`  // Lakhs
    FundingBandUpper int            `json:"fundingBandUpper"`  // Lakhs
    SubScores       SubScores       `json:"subScores"`
    ImprovementTips []ImprovementTip `json:"improvementTips"`
}
```

#### University (embedded data)
```go
type University struct {
    ID              string      `json:"id"`
    Name            string      `json:"name"`
    Country         Country     `json:"country"`
    ProgramType     ProgramType `json:"programType"`
    ProgramName     string      `json:"programName"`
    TotalCostUSD    int         `json:"totalCostUSD"`    // total 2-year cost
    TotalCostINR    int64       `json:"totalCostINR"`    // paise
    PostStudySalaryUSD int      `json:"postStudySalaryUSD"`
    ROIYears        float64     `json:"roiYears"`
    AdmitProbability map[string]float64 `json:"admitProbability"`
    // keys: "cgpa_9plus", "cgpa_8to9", "cgpa_7to8", "cgpa_below7"
}
```

#### LoanOffer (embedded data)
```go
type LoanOffer struct {
    ID                   string  `json:"id"`
    LenderName           string  `json:"lenderName"`
    MaxAmountLakh        int     `json:"maxAmountLakh"`
    RateMin              float64 `json:"rateMin"`
    RateMax              float64 `json:"rateMax"`
    MoratoriumMonths     int     `json:"moratoriumMonths"`
    RepaymentYears       int     `json:"repaymentYears"`
    ProcessingFeePercent float64 `json:"processingFeePercent"`
    CollateralRequired   bool    `json:"collateralRequired"`
    USP                  string  `json:"usp"`
    ProcessingDays       int     `json:"processingDays"`
}
```

### 5. Build Sequence

```
Phase A: Core Domain (pure logic, no dependencies)
  A1. config/config.go
  A2. internal/ecp/scorer.go + scorer_test.go
  A3. internal/roi/calculator.go
  A4. internal/loans/offers.go + types.go

Phase B: Data Layer (stores + embedded data)
  B1. data/universities.json (50 records)
  B2. data/loan_offers.json (3 records)
  B3. internal/matching/matcher.go + types.go
  B4. internal/students/store.go
  B5. internal/applications/store.go

Phase C: AI Layer
  C1. internal/ai/prompt.go
  C2. internal/ai/client.go
  C3. internal/ai/fallback.go
  C4. internal/ai/ratelimiter.go

Phase D: HTTP Layer
  D1. internal/middleware/ (cors, logging, recovery, requestid)
  D2. internal/handlers/response.go
  D3. internal/handlers/ (ecp, students, universities, loans, chat, health, passport, dreamgap)

Phase E: Wiring + Deployment
  E1. cmd/server/main.go
  E2. go.mod, Dockerfile, render.yaml, Makefile, .env.example, README.md
```

### 6. Concurrency Strategy

| Location | Primitive | Justification |
|---|---|---|
| Student Store | `sync.RWMutex` | Multiple concurrent reads (GET student) with occasional writes (POST/PUT). RWMutex allows read parallelism. |
| Application Store | `sync.RWMutex` | Same pattern as student store. |
| TTL Cleanup | `time.Ticker` + goroutine | Background eviction every 5 min. Runs under write lock, short duration. |
| Rate Limiter | `sync.Mutex` per bucket map | Token bucket state must be atomically updated per request. |
| Rate Limiter Cleanup | `time.Ticker` + goroutine | Evict stale buckets (>2 hrs) every 10 min. |
| Chat SSE Streaming | goroutine + `context.Cancel` | Gemini API stream runs in handler goroutine. Client disconnect → context cancel → API call abort. No extra goroutine needed since the handler itself IS the goroutine. |
| University Matching | **No fan-out** for hackathon | 50 records is trivially fast (<1ms). Fan-out adds complexity without benefit. The PRD says "when dataset > 100 records" — we have 50. Matching remains sequential. |
| Graceful Shutdown | `os.Signal` + `http.Server.Shutdown` | 30-second drain. `Shutdown` waits for in-flight requests. SSE connections get canceled via context. |

**Not using concurrency where it doesn't help:** University matching (50 records), ECP calculation (pure arithmetic), ROI calculation (10 iterations). Adding goroutines here would be premature complexity.

### 7. Error Taxonomy

#### User-Facing (4xx)
```go
var (
    ErrInvalidInput    = errors.New("invalid input")        // 400
    ErrNotFound        = errors.New("not found")            // 404
    ErrRateLimited     = errors.New("rate limit exceeded")  // 429
    ErrValidation      = errors.New("validation failed")    // 422
)
```
- Always include actionable `message` in response envelope
- Include `code` field for frontend switching: `"INVALID_CGPA"`, `"STUDENT_NOT_FOUND"`, `"RATE_LIMITED"`

#### Internal (5xx — logged, sanitized to client)
```go
var (
    ErrStoreInternal   = errors.New("store operation failed")
    ErrAIClient        = errors.New("ai client error")
    ErrDataLoad        = errors.New("data loading failed")
)
```
- Full error logged with `slog.Error()` including request ID
- Client sees: `"An internal error occurred. Please try again."`

#### Fatal (startup only)
- Missing `GOOGLE_API_KEY` → soft warning (chat uses fallbacks); NOT fatal
- Failed to parse embedded JSON → `log.Fatal`
- Port already in use → `log.Fatal`

### 8. Hackathon Mocking Strategy

| Component | Real vs. Mock | Reasoning |
|---|---|---|
| ECP Scoring | **Real** | Pure computation, fully implemented per PRD formula |
| University Matching | **Real** | Algorithm runs on embedded JSON, fully implemented |
| ROI Calculator | **Real** | Pure math, fully implemented |
| EMI Calculator | **Real** | Standard EMI formula, trivial |
| Gemini AI Chat | **Real** (with fallback) | Uses Gemini 2.5 Flash (FREE tier); 5 fallback responses if API fails |
| Student Store | **In-memory** (mock DB) | HashMap with TTL — structurally identical to a DB-backed store |
| Application Store | **In-memory** (mock DB) | Same pattern. Application IDs are real, status is simulated |
| Loan Application Status | **Mocked** | Always returns "Submitted" — no actual NBFC integration |
| NBFC Marketplace | **Static data** | 3 hardcoded offers per PRD |
| Document Verification | **Mocked** | Checklist generated but no actual verification |
| University Data | **Realistic static** | 50 real universities with approximate real costs/salaries |

---

## Proposed Changes

### Phase A — Core Domain

#### [NEW] [config.go](file:///c:/Users/gobesh%20j/Desktop/Disha%20AI/config/config.go)
- Loads env vars: `GOOGLE_API_KEY`, `PORT`, `ENV`, `RATE_LIMIT_REQUESTS`, `RATE_LIMIT_WINDOW_HOURS`, `CORS_ALLOWED_ORIGINS`
- All vars have sensible defaults; `GOOGLE_API_KEY` is optional (chat falls back to keyword-matched responses if absent)
- Uses `os.Getenv` with fallbacks — no external config libraries

#### [NEW] [scorer.go](file:///c:/Users/gobesh%20j/Desktop/Disha%20AI/internal/ecp/scorer.go)
- Complete ECP scoring engine implementing the exact formula from PRD
- Strong types: `IncomeBand`, `ProgramType`, `Country`, `IntakePeriod` as `type X string` with const blocks
- `Calculate(profile StudentProfile) ECPResult` — pure function, no I/O
- Sub-score functions: `academicScore()`, `financialScore()`, `loanReadinessScore()`
- `fundingBand()` and `tier()` functions
- `improvementTips()` — dynamic tips based on weakest sub-score

#### [NEW] [scorer_test.go](file:///c:/Users/gobesh%20j/Desktop/Disha%20AI/internal/ecp/scorer_test.go)
- 10 table-driven test cases:
  1. Perfect profile → score 100, Green
  2. Minimal profile → score < 30, Red
  3. Priya (CGPA 7.8, GRE 312, 8-20L, co-applicant) → 72-76, Green
  4. Rohan (CGPA 8.1, GRE 318, MBA) → ~82, Green
  5. Anjali (CGPA 8.5, no GRE, no co-applicant) → ~58, Amber
  6. Edge: CGPA 8.0, GRE 305, no co-applicant
  7. Edge: co-applicant with empty income
  8. Edge: GRE = 0 (plans to take)
  9. Edge: income "<3L", no co-applicant
  10. Edge: max everything except low income

#### [NEW] [calculator.go](file:///c:/Users/gobesh%20j/Desktop/Disha%20AI/internal/roi/calculator.go)
- `Calculate(params ROIParams) ROIResult`
- 10-year projection with compound growth
- EMI calculation using standard formula
- Break-even year detection
- 10-year gain calculation

#### [NEW] [offers.go](file:///c:/Users/gobesh%20j/Desktop/Disha%20AI/internal/loans/offers.go) + [types.go](file:///c:/Users/gobesh%20j/Desktop/Disha%20AI/internal/loans/types.go)
- `FilterAndRankOffers(ecpResult, allOffers) []RankedOffer`
- Score lock check (score < 60 → locked)
- `CalculateEMI(amount, rate, years) EMIResult`
- Illustrative EMI augmentation per offer

---

### Phase B — Data Layer

#### [NEW] [universities.json](file:///c:/Users/gobesh%20j/Desktop/Disha%20AI/data/universities.json)
- 50 university records: 25 USA, 8 UK, 6 Canada, 5 Germany, 3 Australia, 3 India
- Real program names, approximate 2024-2025 costs, realistic admit probability bands
- Post-graduation salary estimates, ROI years

#### [NEW] [loan_offers.json](file:///c:/Users/gobesh%20j/Desktop/Disha%20AI/data/loan_offers.json)
- 3 NBFC records: HDFC Credila, Avanse Financial, InCred Finance
- Exact data from PRD

#### [NEW] [matcher.go](file:///c:/Users/gobesh%20j/Desktop/Disha%20AI/internal/matching/matcher.go) + [types.go](file:///c:/Users/gobesh%20j/Desktop/Disha%20AI/internal/matching/types.go)
- Loads universities from `embed.FS` at init
- `Match(profile, ecpResult, filters) []MatchedUniversity`
- Coverage %, funding status, admit probability, ROI score, composite match score
- Sorted: Within Band > Stretch Goal > Out of Range, then by matchScore desc
- Returns top 8

#### [NEW] [store.go](file:///c:/Users/gobesh%20j/Desktop/Disha%20AI/internal/students/store.go)
- `type Store struct` with `sync.RWMutex` + `map[string]*StudentRecord`
- CRUD: `Create`, `Get`, `Update`
- `StudentRecord` contains `StudentProfile`, `ECPResult`, `CreatedAt`
- TTL: 24-hour expiry, background cleanup goroutine
- 3 demo profiles pre-loaded at startup

#### [NEW] [store.go](file:///c:/Users/gobesh%20j/Desktop/Disha%20AI/internal/applications/store.go)
- Thread-safe application store
- `Create`, `Get` methods
- Application ID format: `"DISHA-2025-XXXXXX"`
- Document checklist generation

---

### Phase C — AI Layer

#### [NEW] [prompt.go](file:///c:/Users/gobesh%20j/Desktop/Disha%20AI/internal/ai/prompt.go)
- `BuildSystemPrompt(profile, ecpResult, topUniversities) string`
- Full profile injection per PRD template
- Behavior rules and tone embedded

#### [NEW] [client.go](file:///c:/Users/gobesh%20j/Desktop/Disha%20AI/internal/ai/client.go)
- Wraps `google.golang.org/genai` client with `genai.BackendGeminiAPI`
- Model: `gemini-2.5-flash` (FREE tier, no billing required)
- `StreamChat(ctx, systemPrompt, messages, writer) error`
- Uses `client.Models.GenerateContentStream()` → iterates `stream.Next()` → extracts `part.Text`
- SSE format: `data: {"type":"delta","text":"..."}\n\n`
- Context deadline: 30 seconds
- Graceful error → fallback delegation

#### [NEW] [fallback.go](file:///c:/Users/gobesh%20j/Desktop/Disha%20AI/internal/ai/fallback.go)
- 5 keyword-matched fallback responses
- Template variables populated from student profile
- `GetFallbackResponse(lastMessage, profile, ecpResult) string`

#### [NEW] [ratelimiter.go](file:///c:/Users/gobesh%20j/Desktop/Disha%20AI/internal/ai/ratelimiter.go)
- Token bucket per student ID: 20 tokens/hour
- `Allow(studentID) (bool, time.Duration)` — returns whether allowed + retry-after
- Background cleanup of stale buckets (>2 hours since last access)

---

### Phase D — HTTP Layer

#### [NEW] Middleware files
- `cors.go` — Permissive CORS, OPTIONS preflight → 204
- `logging.go` — Structured `slog` logging: method, path, duration, status, request ID
- `recovery.go` — Panic recovery → 500 with sanitized message
- `requestid.go` — UUID from `crypto/rand`, injected into `context.Context`

#### [NEW] [response.go](file:///c:/Users/gobesh%20j/Desktop/Disha%20AI/internal/handlers/response.go)
- `type APIResponse[T any] struct` — the standard envelope
- `WriteSuccess(w, data, statusCode)`
- `WriteError(w, code, message, statusCode)`
- Request ID and timestamp auto-populated from context

#### [NEW] Handler files
- Each handler file contains a handler struct with dependencies injected via constructor
- All handlers are thin: parse → validate → delegate → respond
- Error responses use consistent envelope format

---

### Phase E — Wiring + Deployment

#### [NEW] [main.go](file:///c:/Users/gobesh%20j/Desktop/Disha%20AI/cmd/server/main.go)
- Loads config
- Initializes all stores and services
- Pre-loads 3 demo students
- Sets up chi router with middleware stack
- Registers all routes
- Starts HTTP server with graceful shutdown (SIGTERM/SIGINT, 30s drain)

#### [NEW] Deployment files
- `Dockerfile` — multi-stage, final image < 20MB
- `render.yaml` — Render.com deployment
- `Makefile` — build, test, run, lint, docker-build, docker-run
- `.env.example` — documented env vars
- `README.md` — complete setup + cURL examples

---

## Router Choice: chi v5

**Why chi over gorilla/mux:**
1. gorilla/mux is **archived** (maintenance mode since 2022). chi is actively maintained.
2. chi is 100% compatible with `net/http` — uses standard `http.Handler` signatures.
3. chi has built-in middleware composition via `r.Use()` and route grouping via `r.Route()`.
4. chi's `chi.URLParam(r, "id")` is cleaner than gorilla's `mux.Vars(r)`.
5. Go 1.22's new ServeMux routing patterns are close but lack middleware composition and sub-router mounting that chi provides out of the box.
6. chi adds zero magic — it's a thin router, not a framework.

---

## Open Questions

> [!IMPORTANT]
> **Gemini API Key:** The chat endpoint uses `GOOGLE_API_KEY` (free tier). Should the server start without it (all non-chat endpoints work, chat returns keyword-matched fallback responses)? Or should it refuse to start?  
> **My default:** Start without it — chat endpoint returns fallback responses. This lets the frontend team test all other features without any API key.

> [!IMPORTANT]
> **University cost data:** The PRD says "real approximate costs (2024-2025)." I'll use publicly available tuition data for 50 programs. These are estimates — should I document them as "approximate, for demo purposes" in the README?  
> **My default:** Yes, with a disclaimer.

---

## Verification Plan

### Automated Tests
1. `go test ./internal/ecp/... -v` — 10 table-driven ECP scoring tests
2. `go build ./cmd/server/` — compile check
3. `go vet ./...` — static analysis
4. Manual cURL tests against all 16 endpoints after server start

### Manual Verification
1. Start server with `make run`
2. Hit `/api/v1/health` — verify 200 OK
3. Get demo student: `GET /api/v1/students/STU-PRIYA001` — verify ECP score ~74
4. Match universities: `GET /api/v1/universities/match?studentId=STU-PRIYA001` — verify 8 results
5. Calculate ROI: `POST /api/v1/roi/calculate` with Priya's profile
6. Get loan offers: `GET /api/v1/loans/offers?studentId=STU-ROHAN002` — verify 3 offers
7. Chat endpoint: `POST /api/v1/chat` — verify SSE streaming via Gemini 2.5 Flash (or fallback if no key)
8. Create application: `POST /api/v1/applications` — verify DISHA-2025-XXXXXX ID
