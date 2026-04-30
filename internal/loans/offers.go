package loans

import (
	"math"
	"sort"
)

// CalculateEMI computes the EMI for a loan using the standard reducing-balance formula.
// Returns values in INR (not paise) for demo readability.
// Validation: amount 1-100L, rate 5-25%, years 1-20.
func CalculateEMI(loanAmountLakh float64, annualRatePercent float64, repaymentYears int) EMIResult {
	if loanAmountLakh <= 0 || annualRatePercent <= 0 || repaymentYears <= 0 {
		return EMIResult{}
	}

	principal := loanAmountLakh * 100000.0 // Lakhs → INR
	r := annualRatePercent / 12.0 / 100.0  // monthly rate as decimal
	n := float64(repaymentYears * 12)      // total months

	power := math.Pow(1+r, n)
	monthlyEMI := principal * r * power / (power - 1)

	totalPayable := monthlyEMI * float64(repaymentYears*12)
	totalInterest := totalPayable - principal

	return EMIResult{
		MonthlyEMI:    int(math.Round(monthlyEMI)),
		TotalPayable:  int(math.Round(totalPayable)),
		TotalInterest: int(math.Round(totalInterest)),
	}
}

// FilterAndRankOffers returns loan offers filtered and ranked for a student's profile.
// If ECP score < 60, returns a locked response.
func FilterAndRankOffers(ecpScore int, fundingBandLower int, fundingBandUpper int, allOffers []LoanOffer) OffersResponse {
	if ecpScore < 60 {
		return OffersResponse{
			Locked:        true,
			RequiredScore: 60,
		}
	}

	var ranked []RankedOffer

	for _, offer := range allOffers {
		// Filter: only offers where maxAmountLakh >= lower bound of funding band
		if offer.MaxAmountLakh < fundingBandLower {
			continue
		}

		// Recommended loan = min(fundingBandUpper, offer.MaxAmountLakh)
		recommended := fundingBandUpper
		if offer.MaxAmountLakh < recommended {
			recommended = offer.MaxAmountLakh
		}

		// Illustrative EMI at midpoint rate and max tenure
		midRate := (offer.RateMin + offer.RateMax) / 2.0
		emi := CalculateEMI(float64(recommended), midRate, offer.RepaymentYears)

		// Match score: higher maxAmount + lower rate + faster processing = better
		// Normalize each factor to 0-1 and weight
		amountScore := float64(offer.MaxAmountLakh) / 100.0          // max 100L → 1.0
		rateScore := 1.0 - (offer.RateMin / 25.0)                   // lower rate → higher score
		speedScore := 1.0 - (float64(offer.ProcessingDays) / 30.0)  // faster → higher score
		feeScore := 1.0 - (offer.ProcessingFeePercent / 5.0)        // lower fee → higher score

		matchScore := (amountScore * 0.35) + (rateScore * 0.30) + (speedScore * 0.20) + (feeScore * 0.15)
		matchScore = math.Round(matchScore*100) / 100

		ranked = append(ranked, RankedOffer{
			LoanOffer:           offer,
			RecommendedLoanLakh: recommended,
			IllustrativeEMI:     emi.MonthlyEMI,
			MatchScore:          matchScore,
		})
	}

	// Sort by matchScore descending
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].MatchScore > ranked[j].MatchScore
	})

	return OffersResponse{
		Locked: false,
		Offers: ranked,
	}
}
