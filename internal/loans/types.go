// Package loans provides the NBFC loan offer service, EMI calculator, and offer ranking.
package loans

// LoanOffer represents an NBFC loan product.
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

// EMIResult holds the output of an EMI calculation.
type EMIResult struct {
	MonthlyEMI    int `json:"monthlyEMI"`    // INR (not paise, for demo readability)
	TotalPayable  int `json:"totalPayable"`  // INR
	TotalInterest int `json:"totalInterest"` // INR
}

// RankedOffer is a loan offer augmented with student-specific calculations.
type RankedOffer struct {
	LoanOffer
	RecommendedLoanLakh int     `json:"recommendedLoanLakh"`
	IllustrativeEMI     int     `json:"illustrativeEMI"` // monthly EMI in INR
	MatchScore          float64 `json:"matchScore"`
}

// OffersResponse is the response envelope for the loan offers endpoint.
type OffersResponse struct {
	Locked        bool          `json:"locked"`
	RequiredScore int           `json:"requiredScore,omitempty"`
	Offers        []RankedOffer `json:"offers,omitempty"`
}
