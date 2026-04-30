# DISHA AI — Backend API

**Know Your Funding First. Then Dream Bigger.**

Go backend powering the DISHA AI education finance platform — ECP scoring, university matching, loan marketplace simulation, ROI projections, and AI counselling.

## Quick Start

```bash
# Clone and enter the project
cd disha-backend

# Install dependencies
go mod download

# Run the server
go run ./cmd/server/

# Or with Make
make run
```

The server starts at `http://localhost:8080` with 3 pre-loaded demo students.

## Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `GOOGLE_API_KEY` | No | - | Gemini API key for AI chat (falls back to smart responses without it) |
| `PORT` | No | 8080 | Server port |
| `ENV` | No | development | `development` or `production` |
| `RATE_LIMIT_REQUESTS` | No | 20 | Max chat messages per student per hour |
| `CORS_ALLOWED_ORIGINS` | No | * | Allowed CORS origins |

## API Endpoints

### Health Check
```bash
curl http://localhost:8080/api/v1/health
```

### ECP Score Calculation
```bash
curl -X POST http://localhost:8080/api/v1/ecp/calculate \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Priya Sharma",
    "cgpa": 7.8,
    "greScore": 312,
    "familyIncome": "8-20L",
    "hasCoApplicant": true,
    "coApplicantIncome": "8-20L",
    "targetCountry": "USA",
    "targetProgram": "MS",
    "intake": "Sep2026"
  }'
```

### Get Demo Student
```bash
curl http://localhost:8080/api/v1/students/STU-PRIYA001
curl http://localhost:8080/api/v1/students/STU-ROHAN002
curl http://localhost:8080/api/v1/students/STU-ANJALI003
```

### University Matching
```bash
curl "http://localhost:8080/api/v1/universities/match?studentId=STU-PRIYA001"
curl "http://localhost:8080/api/v1/universities/match?studentId=STU-PRIYA001&country=USA&program=MS"
```

### Loan Offers
```bash
curl "http://localhost:8080/api/v1/loans/offers?studentId=STU-ROHAN002"
```

### EMI Calculator
```bash
curl -X POST http://localhost:8080/api/v1/loans/emi \
  -H "Content-Type: application/json" \
  -d '{"loanAmountLakh": 50, "annualRatePercent": 10.5, "repaymentYears": 10}'
```

### ROI Calculation
```bash
curl -X POST http://localhost:8080/api/v1/roi/calculate \
  -H "Content-Type: application/json" \
  -d '{"studentId": "STU-PRIYA001", "universityId": "US-GATECH-MS", "loanAmountLakh": 50, "annualRatePercent": 10.5, "currentSalaryLPA": 5}'
```

### AI Chat (SSE Streaming)
```bash
curl -X POST http://localhost:8080/api/v1/chat \
  -H "Content-Type: application/json" \
  -d '{"studentId": "STU-PRIYA001", "messages": [{"role": "user", "content": "Which university should I apply to?"}]}'
```

### Funding Passport
```bash
curl http://localhost:8080/api/v1/funding-passport/STU-PRIYA001
```

### Dream Gap Analysis
```bash
curl "http://localhost:8080/api/v1/dream-gap?studentId=STU-PRIYA001&universityId=US-MIT-MS"
```

### Loan Application
```bash
curl -X POST http://localhost:8080/api/v1/applications \
  -H "Content-Type: application/json" \
  -d '{"studentId": "STU-PRIYA001", "loanOfferId": "HDFC-CREDILA", "requestedAmountLakh": 50, "phone": "9876543210", "panLastFour": "1234", "targetUniversityId": "US-GATECH-MS"}'
```

## Demo Students

| ID | Name | ECP Score | Tier |
|---|---|---|---|
| `STU-PRIYA001` | Priya Sharma | 88 | Green |
| `STU-ROHAN002` | Rohan Mehta | 93 | Green |
| `STU-ANJALI003` | Anjali Nair | 70 | Green |

## Tech Stack

- **Language:** Go 1.22+
- **Router:** chi v5 (100% net/http compatible)
- **AI:** Google Gemini 2.5 Flash (free tier) via `google.golang.org/genai`
- **Storage:** In-memory with `sync.RWMutex` (24h TTL)
- **Data:** 50 universities + 3 NBFCs embedded via `embed.FS`

## Deployment

```bash
# Docker
make docker-build
make docker-run

# Render.com — push to GitHub and connect render.yaml
```

> **Note:** University cost data is approximate (2024-2025 estimates) for demonstration purposes.
