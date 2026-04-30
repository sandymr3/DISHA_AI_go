// Package roi implements the Return on Investment projection engine for education programs.
package roi

import (
	"math"
)

// Params holds the inputs for an ROI calculation.
type Params struct {
	TotalCostINR       int64   `json:"totalCostINR"`       // total program cost in paise
	PostStudySalaryUSD int     `json:"postStudySalaryUSD"` // expected annual salary post-graduation in USD
	LoanAmountLakh     float64 `json:"loanAmountLakh"`     // loan principal in Lakhs INR
	AnnualRatePercent  float64 `json:"annualRatePercent"`  // annual interest rate as percentage
	RepaymentYears     int     `json:"repaymentYears"`     // loan tenure in years
	CurrentSalaryLPA   float64 `json:"currentSalaryLPA"`   // current annual salary in Lakhs
}

// DataPoint represents one year in the ROI projection.
type DataPoint struct {
	Year              int     `json:"year"`
	WithoutDegreeLakh float64 `json:"withoutDegreeLakh"`
	WithDegreeLakh    float64 `json:"withDegreeLakh"`
}

// Result is the output of the ROI calculation.
type Result struct {
	DataPoints     []DataPoint `json:"dataPoints"`
	BreakEvenYear  *int        `json:"breakEvenYear"`  // nil if never breaks even within 10 years
	TenYearGainLakh float64    `json:"tenYearGainLakh"`
}

// CalculateEMI computes the monthly EMI using the standard formula:
// EMI = P × r × (1+r)^n / ((1+r)^n - 1)
// where P = principal (in Lakhs), r = monthly rate, n = total months.
// Returns monthly EMI in Lakhs.
func CalculateEMI(principalLakh float64, annualRatePercent float64, repaymentYears int) float64 {
	if principalLakh <= 0 || annualRatePercent <= 0 || repaymentYears <= 0 {
		return 0
	}

	r := annualRatePercent / 12.0 / 100.0 // monthly rate as decimal
	n := float64(repaymentYears * 12)      // total months

	// EMI formula: P * r * (1+r)^n / ((1+r)^n - 1)
	power := math.Pow(1+r, n)
	emi := principalLakh * r * power / (power - 1)

	return emi
}

// Calculate projects ROI over 10 years for a given education program and loan.
func Calculate(params Params) Result {
	// Convert post-study salary from USD to LPA (Lakhs Per Annum)
	// Using 83 INR/USD exchange rate as per PRD
	postStudySalaryLPA := (float64(params.PostStudySalaryUSD) * 83.0) / 100000.0

	// Total program cost in Lakhs
	totalCostLakh := float64(params.TotalCostINR) / 100.0 / 100000.0 // paise → INR → Lakhs

	// Monthly EMI in Lakhs
	monthlyEMI := CalculateEMI(params.LoanAmountLakh, params.AnnualRatePercent, params.RepaymentYears)
	annualEMILPA := monthlyEMI * 12.0

	dataPoints := make([]DataPoint, 11) // years 0 through 10
	var breakEvenYear *int

	for year := 0; year <= 10; year++ {
		// Without degree: cumulative earnings growing at 6% per year
		var withoutDegree float64
		if year > 0 {
			// Cumulative earnings with 6% annual growth
			for y := 1; y <= year; y++ {
				withoutDegree += params.CurrentSalaryLPA * math.Pow(1.06, float64(y-1))
			}
		}

		// With degree
		var withDegree float64
		if year <= 2 {
			// During study period: accumulating cost
			withDegree = -(totalCostLakh) * (float64(year) / 2.0)
		} else {
			// Post-graduation: earning - EMI - total cost
			workYears := year - 2

			// Cumulative earnings with 7% annual growth
			var earnings float64
			for y := 1; y <= workYears; y++ {
				earnings += postStudySalaryLPA * math.Pow(1.07, float64(y-1))
			}

			// EMI paid (only during repayment period)
			emiYears := workYears
			if emiYears > params.RepaymentYears {
				emiYears = params.RepaymentYears
			}
			emiPaid := float64(emiYears) * annualEMILPA

			withDegree = earnings - emiPaid - totalCostLakh
		}

		dataPoints[year] = DataPoint{
			Year:              year,
			WithoutDegreeLakh: math.Round(withoutDegree*100) / 100,
			WithDegreeLakh:    math.Round(withDegree*100) / 100,
		}

		// Detect break-even: first year after study period where withDegree > withoutDegree
		if year > 2 && breakEvenYear == nil && withDegree > withoutDegree {
			y := year
			breakEvenYear = &y
		}
	}

	tenYearGain := dataPoints[10].WithDegreeLakh - dataPoints[10].WithoutDegreeLakh
	tenYearGain = math.Round(tenYearGain*100) / 100

	return Result{
		DataPoints:     dataPoints,
		BreakEvenYear:  breakEvenYear,
		TenYearGainLakh: tenYearGain,
	}
}
