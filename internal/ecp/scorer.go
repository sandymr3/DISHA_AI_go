// Package ecp implements the Education Credit Profile scoring engine.
// The ECP Score is a 0–100 fundability metric computed from a student's
// academic profile, family income, co-applicant data, and target program.
package ecp

import (
	"errors"
	"fmt"
	"math"
)

// --- Sentinel errors ---

var (
	// ErrInvalidInput indicates a validation failure in the student profile.
	ErrInvalidInput = errors.New("invalid input")
)

// --- Domain types ---

// IncomeBand represents a family income bracket in Lakhs INR per annum.
type IncomeBand string

const (
	IncomeLess3L  IncomeBand = "<3L"
	Income3To8L   IncomeBand = "3-8L"
	Income8To20L  IncomeBand = "8-20L"
	Income20LPlus IncomeBand = "20L+"
)

// ProgramType represents the target postgraduate program.
type ProgramType string

const (
	ProgramMBA  ProgramType = "MBA"
	ProgramMS   ProgramType = "MS"
	ProgramMiM  ProgramType = "MiM"
	ProgramPhD  ProgramType = "PhD"
	ProgramMArch ProgramType = "MArch"
	ProgramMPH  ProgramType = "MPH"
)

// Country represents the target study destination.
type Country string

const (
	CountryUSA       Country = "USA"
	CountryUK        Country = "UK"
	CountryCanada    Country = "Canada"
	CountryGermany   Country = "Germany"
	CountryAustralia Country = "Australia"
	CountryIndia     Country = "India"
)

// IntakePeriod represents the planned enrollment intake.
type IntakePeriod string

const (
	IntakeJan2026 IntakePeriod = "Jan2026"
	IntakeSep2026 IntakePeriod = "Sep2026"
	IntakeJan2027 IntakePeriod = "Jan2027"
)

// StudentProfile is the input to the ECP scoring engine.
type StudentProfile struct {
	Name              string      `json:"name"`
	CGPA              float64     `json:"cgpa"`
	GREScore          int         `json:"greScore"`
	FamilyIncome      IncomeBand  `json:"familyIncome"`
	HasCoApplicant    bool        `json:"hasCoApplicant"`
	CoApplicantIncome IncomeBand  `json:"coApplicantIncome"`
	TargetCountry     Country     `json:"targetCountry"`
	TargetProgram     ProgramType `json:"targetProgram"`
	Intake            IntakePeriod `json:"intake"`
}

// SubScores holds the three component sub-scores of the ECP.
type SubScores struct {
	Academic      int `json:"academic"`
	Financial     int `json:"financial"`
	LoanReadiness int `json:"loanReadiness"`
}

// ImprovementTip is a suggested action to improve the ECP score.
type ImprovementTip struct {
	Action        string `json:"action"`
	PotentialGain int    `json:"potentialGain"`
	Effort        string `json:"effort"` // "Low", "Medium", "High"
}

// ECPResult is the output of the ECP scoring engine.
type ECPResult struct {
	Score            int              `json:"score"`
	Tier             string           `json:"tier"`
	FundingBandLower int              `json:"fundingBandLower"`
	FundingBandUpper int              `json:"fundingBandUpper"`
	SubScores        SubScores        `json:"subScores"`
	ImprovementTips  []ImprovementTip `json:"improvementTips"`
}

// --- Lookup tables ---

var incomeMap = map[IncomeBand]int{
	IncomeLess3L:  8,
	Income3To8L:   18,
	Income8To20L:  28,
	Income20LPlus: 36,
}

var programMap = map[ProgramType]int{
	ProgramMBA:  10,
	ProgramMS:   9,
	ProgramMiM:  7,
	ProgramPhD:  8,
	ProgramMArch: 6,
	ProgramMPH:  7,
}

var countryMap = map[Country]int{
	CountryUSA:       10,
	CountryUK:        9,
	CountryCanada:    9,
	CountryGermany:   7,
	CountryAustralia: 8,
	CountryIndia:     6,
}

var intakeMap = map[IntakePeriod]int{
	IntakeJan2026: 10,
	IntakeSep2026: 8,
	IntakeJan2027: 5,
}

var incomeBaseLakh = map[IncomeBand]int{
	IncomeLess3L:  15,
	Income3To8L:   30,
	Income8To20L:  50,
	Income20LPlus: 75,
}

// validIncomeBands is the set of valid income band values.
var validIncomeBands = map[IncomeBand]bool{
	IncomeLess3L: true, Income3To8L: true, Income8To20L: true, Income20LPlus: true,
}

var validPrograms = map[ProgramType]bool{
	ProgramMBA: true, ProgramMS: true, ProgramMiM: true,
	ProgramPhD: true, ProgramMArch: true, ProgramMPH: true,
}

var validCountries = map[Country]bool{
	CountryUSA: true, CountryUK: true, CountryCanada: true,
	CountryGermany: true, CountryAustralia: true, CountryIndia: true,
}

var validIntakes = map[IntakePeriod]bool{
	IntakeJan2026: true, IntakeSep2026: true, IntakeJan2027: true,
}

// --- Validation ---

// Validate checks that all fields of the StudentProfile are within valid ranges.
func (p StudentProfile) Validate() error {
	if p.CGPA < 4.0 || p.CGPA > 10.0 {
		return fmt.Errorf("ecp.Validate: CGPA %.1f out of range [4.0, 10.0]: %w", p.CGPA, ErrInvalidInput)
	}
	if p.GREScore != 0 && (p.GREScore < 260 || p.GREScore > 340) {
		return fmt.Errorf("ecp.Validate: GRE score %d out of range [260, 340] or 0: %w", p.GREScore, ErrInvalidInput)
	}
	if !validIncomeBands[p.FamilyIncome] {
		return fmt.Errorf("ecp.Validate: invalid family income band %q: %w", p.FamilyIncome, ErrInvalidInput)
	}
	if p.HasCoApplicant && p.CoApplicantIncome != "" && !validIncomeBands[p.CoApplicantIncome] {
		return fmt.Errorf("ecp.Validate: invalid co-applicant income band %q: %w", p.CoApplicantIncome, ErrInvalidInput)
	}
	if !validPrograms[p.TargetProgram] {
		return fmt.Errorf("ecp.Validate: invalid program type %q: %w", p.TargetProgram, ErrInvalidInput)
	}
	if !validCountries[p.TargetCountry] {
		return fmt.Errorf("ecp.Validate: invalid country %q: %w", p.TargetCountry, ErrInvalidInput)
	}
	if !validIntakes[p.Intake] {
		return fmt.Errorf("ecp.Validate: invalid intake period %q: %w", p.Intake, ErrInvalidInput)
	}
	return nil
}

// --- Scoring functions ---

// academicScore computes the Academic Strength sub-score (0–30).
func academicScore(cgpa float64, greScore int) int {
	cgpaScore := int(math.Round((cgpa / 10.0) * 20.0))

	var greBonus int
	switch {
	case greScore >= 325:
		greBonus = 10
	case greScore >= 315:
		greBonus = 8
	case greScore >= 305:
		greBonus = 6
	case greScore >= 295:
		greBonus = 4
	case greScore > 0:
		greBonus = 2
	default:
		// GRE = 0 means plans to take
		greBonus = 1
	}

	return min(30, cgpaScore+greBonus)
}

// financialScore computes the Financial Foundation sub-score (0–40).
func financialScore(familyIncome IncomeBand, hasCoApplicant bool, coApplicantIncome IncomeBand) int {
	baseIncome := incomeMap[familyIncome]

	if !hasCoApplicant {
		return min(40, baseIncome)
	}

	// Default co-applicant income to "<3L" if not specified
	coIncome := coApplicantIncome
	if coIncome == "" {
		coIncome = IncomeLess3L
	}

	coIncomeScore := incomeMap[coIncome]
	financial := baseIncome + int(math.Round(float64(coIncomeScore)*0.25)) + 4

	return min(40, financial)
}

// loanReadinessScore computes the Loan Readiness sub-score (0–30).
func loanReadinessScore(program ProgramType, country Country, intake IntakePeriod) int {
	p := programMap[program]
	c := countryMap[country]
	i := intakeMap[intake]
	return min(30, p+c+i)
}

// tier returns the funding tier based on the total ECP score.
func tier(score int) string {
	switch {
	case score >= 70:
		return "Green"
	case score >= 50:
		return "Amber"
	default:
		return "Red"
	}
}

// fundingBand computes the estimated funding range in Lakhs INR.
func fundingBand(score int, familyIncome IncomeBand, hasCoApplicant bool) (lower, upper int) {
	base := incomeBaseLakh[familyIncome]
	scoreModifier := int(math.Round(float64(score) / 100.0 * 20.0))
	coApplicantBoost := 0
	if hasCoApplicant {
		coApplicantBoost = 10
	}

	lower = int(math.Round(float64(base)*0.7)) + scoreModifier
	upper = base + scoreModifier + coApplicantBoost

	return lower, upper
}

// improvementTips generates actionable suggestions based on the weakest areas.
func improvementTips(p StudentProfile) []ImprovementTip {
	var tips []ImprovementTip

	if !p.HasCoApplicant {
		tips = append(tips, ImprovementTip{
			Action:        "Add co-applicant with income above ₹8L",
			PotentialGain: 12,
			Effort:        "Medium",
		})
	}

	if p.GREScore == 0 {
		tips = append(tips, ImprovementTip{
			Action:        "Appear for GRE and score above 305",
			PotentialGain: 6,
			Effort:        "High",
		})
	}

	if p.CGPA < 8.0 {
		tips = append(tips, ImprovementTip{
			Action:        "A strong LOR can partially offset CGPA below 8.0",
			PotentialGain: 3,
			Effort:        "Low",
		})
	}

	if p.FamilyIncome == IncomeLess3L {
		tips = append(tips, ImprovementTip{
			Action:        "Apply for CSIS subsidy scheme",
			PotentialGain: 5,
			Effort:        "Low",
		})
	}

	return tips
}

// --- Public API ---

// Calculate computes the full ECP result for a validated student profile.
// This is a pure function with no I/O — it runs in <1ms for any input.
func Calculate(profile StudentProfile) (ECPResult, error) {
	if err := profile.Validate(); err != nil {
		return ECPResult{}, fmt.Errorf("ecp.Calculate: %w", err)
	}

	academic := academicScore(profile.CGPA, profile.GREScore)
	financial := financialScore(profile.FamilyIncome, profile.HasCoApplicant, profile.CoApplicantIncome)
	loanReady := loanReadinessScore(profile.TargetProgram, profile.TargetCountry, profile.Intake)

	totalScore := min(100, academic+financial+loanReady)
	t := tier(totalScore)
	lower, upper := fundingBand(totalScore, profile.FamilyIncome, profile.HasCoApplicant)
	tips := improvementTips(profile)

	return ECPResult{
		Score:            totalScore,
		Tier:             t,
		FundingBandLower: lower,
		FundingBandUpper: upper,
		SubScores: SubScores{
			Academic:      academic,
			Financial:     financial,
			LoanReadiness: loanReady,
		},
		ImprovementTips: tips,
	}, nil
}
