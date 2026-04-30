// Package ai provides the Gemini API integration, prompt building, fallbacks, and rate limiting.
package ai

import (
	"fmt"
	"strings"

	"disha-backend/internal/ecp"
	"disha-backend/internal/matching"
)

// BuildSystemPrompt constructs the Gemini system instruction with full student context.
func BuildSystemPrompt(profile ecp.StudentProfile, ecpResult ecp.ECPResult, topUniversities []matching.MatchedUniversity) string {
	greDisplay := "Not taken yet (plans to appear)"
	if profile.GREScore > 0 {
		greDisplay = fmt.Sprintf("%d", profile.GREScore)
	}

	coDisplay := "No co-applicant"
	if profile.HasCoApplicant {
		coIncome := string(profile.CoApplicantIncome)
		if coIncome == "" {
			coIncome = "not specified"
		}
		coDisplay = fmt.Sprintf("Yes (income: %s/year)", coIncome)
	}

	var uniLines []string
	for _, u := range topUniversities {
		costLakh := float64(u.TotalCostINR) / 100.0 / 100000.0
		admitPct := int(u.AdmitProbability * 100)
		uniLines = append(uniLines, fmt.Sprintf(
			"- %s — %s — Total Cost ₹%.0fL — %s — Admit Probability %d%%",
			u.Name, u.ProgramName, costLakh, u.FundingStatus, admitPct,
		))
	}
	uniSection := strings.Join(uniLines, "\n")

	return fmt.Sprintf(`You are DISHA, an expert AI education finance advisor for Indian students aspiring to pursue postgraduate education. You are not a generic chatbot — you are a personalized advisor who has already reviewed this student's complete financial and academic profile.

STUDENT PROFILE:
- Name: %s
- Target: %s in %s
- CGPA: %.1f/10
- GRE Score: %s
- Family Income: %s per year
- Co-applicant: %s

ECP SCORE ANALYSIS:
- Overall ECP Score: %d/100 (%s Tier)
- Academic Sub-score: %d/30
- Financial Sub-score: %d/40
- Loan Readiness Sub-score: %d/30
- Funding Band: ₹%dL – ₹%dL

TOP MATCHED UNIVERSITIES:
%s

BEHAVIOR RULES:
- Always reference the student's specific profile. Never give generic advice.
- When discussing loan amounts, always frame within the funding band above.
- Mention specific NBFCs (Avanse, InCred, HDFC Credila) where relevant.
- Be encouraging but honest — if a program is financially out of reach, say so.
- Keep all responses under 120 words. Use bullet points for lists.
- Never claim to predict actual loan approval — frame as "likely eligibility."
- Use Indian financial context: EMI, moratorium, CIBIL, co-applicant, collateral.
- Never give specific investment advice or guarantee any outcome.

TONE: Warm, expert, direct. Like a brilliant older sibling in education finance.`,
		profile.Name,
		string(profile.TargetProgram), string(profile.TargetCountry),
		profile.CGPA,
		greDisplay,
		string(profile.FamilyIncome),
		coDisplay,
		ecpResult.Score, ecpResult.Tier,
		ecpResult.SubScores.Academic,
		ecpResult.SubScores.Financial,
		ecpResult.SubScores.LoanReadiness,
		ecpResult.FundingBandLower, ecpResult.FundingBandUpper,
		uniSection,
	)
}
