package ai

import (
	"fmt"
	"strings"

	"disha-backend/internal/ecp"
	"disha-backend/internal/matching"
)

// FallbackContext holds the data needed to populate fallback response templates.
type FallbackContext struct {
	Profile   ecp.StudentProfile
	ECPResult ecp.ECPResult
	TopUnis   []matching.MatchedUniversity
}

// GetFallbackResponse returns a keyword-matched fallback response when Gemini is unavailable.
func GetFallbackResponse(lastMessage string, fc FallbackContext) string {
	lower := strings.ToLower(lastMessage)

	topUniName := "your top-matched program"
	if len(fc.TopUnis) > 0 {
		topUniName = fc.TopUnis[0].Name
	}
	matchCount := len(fc.TopUnis)
	admitPct := 0
	if len(fc.TopUnis) > 0 {
		admitPct = int(fc.TopUnis[0].AdmitProbability * 100)
	}
	fundingStatus := "Within Band"
	if len(fc.TopUnis) > 0 {
		fundingStatus = fc.TopUnis[0].FundingStatus
	}

	score := fc.ECPResult.Score
	lb := fc.ECPResult.FundingBandLower
	ub := fc.ECPResult.FundingBandUpper
	country := string(fc.Profile.TargetCountry)
	program := string(fc.Profile.TargetProgram)
	cgpa := fc.Profile.CGPA
	gre := fc.Profile.GREScore

	// Keywords: university, school, apply, which
	if containsAny(lower, "university", "school", "apply", "which", "college") {
		return fmt.Sprintf(
			"Based on your ECP Score of %d and funding band of ₹%dL–₹%dL, I'd focus on programs where total cost stays within ₹%dL. Your profile shows strongest admit probability at %s. Would you like me to detail the loan structure for that program?",
			score, lb, ub, ub, topUniName,
		)
	}

	// Keywords: loan, borrow, EMI, repay, lender
	if containsAny(lower, "loan", "borrow", "emi", "repay", "lender", "nbfc") {
		return fmt.Sprintf(
			"With your ECP Score of %d, you qualify for non-collateral loans up to ₹%dL. HDFC Credila currently offers the most competitive rate at 9.85%% for your profile tier. I'd recommend exploring a ₹%dL loan over 12 years for the most comfortable EMI.",
			score, ub, ub,
		)
	}

	// Keywords: GRE, IELTS, GMAT, score, test
	if containsAny(lower, "gre", "ielts", "gmat", "score", "test", "exam") {
		greStr := fmt.Sprintf("%d", gre)
		if gre == 0 {
			greStr = "not yet taken"
		}
		improved := score + 6
		if improved > 100 {
			improved = 100
		}
		return fmt.Sprintf(
			"Your GRE score of %s is noted for %s programs in %s. Combined with your CGPA of %.1f, your admit probability at target programs is strong. Improving GRE to 320+ would push your ECP from %d to approximately %d and unlock better loan rates.",
			greStr, program, country, cgpa, score, improved,
		)
	}

	// Keywords: ROI, salary, worth, return, investment
	if containsAny(lower, "roi", "salary", "worth", "return", "investment", "earning") {
		return fmt.Sprintf(
			"The ROI analysis for your target %s program shows strong long-term returns. With average post-graduation salaries in %s being significantly higher than domestic alternatives, the 10-year cumulative gain typically exceeds your loan commitment of ₹%dL — making it a sound financial investment.",
			program, country, ub,
		)
	}

	// Default fallback
	return fmt.Sprintf(
		"Great question! Based on your ECP Score of %d and ₹%dL–₹%dL funding band, DISHA has matched you with %d programs across %s. Your strongest option right now is %s — %s, %d%% admit probability for your profile. What aspect would you like to explore deeper?",
		score, lb, ub, matchCount, country, topUniName, fundingStatus, admitPct,
	)
}

func containsAny(s string, keywords ...string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}
