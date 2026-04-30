// Package matching provides the university matching engine.
package matching

import (
	"disha-backend/internal/ecp"
)

// University represents a university program loaded from embedded JSON.
type University struct {
	ID                 string             `json:"id"`
	Name               string             `json:"name"`
	Country            ecp.Country        `json:"country"`
	ProgramType        ecp.ProgramType    `json:"programType"`
	ProgramName        string             `json:"programName"`
	TotalCostUSD       int                `json:"totalCostUSD"`
	TotalCostINR       int64              `json:"totalCostINR"`
	PostStudySalaryUSD int                `json:"postStudySalaryUSD"`
	ROIYears           float64            `json:"roiYears"`
	AdmitProbability   map[string]float64 `json:"admitProbability"`
}

// MatchedUniversity is a university augmented with student-specific match data.
type MatchedUniversity struct {
	University
	CoveragePercent  int     `json:"coveragePercent"`
	FundingStatus    string  `json:"fundingStatus"` // "Within Band", "Stretch Goal", "Out of Range"
	AdmitProbability float64 `json:"admitProbabilityForStudent"`
	ROIScore         float64 `json:"roiScore"`
	MatchScore       float64 `json:"matchScore"`
}

// Filters defines optional filters for university matching.
type Filters struct {
	Country ecp.Country     `json:"country"`
	Program ecp.ProgramType `json:"program"`
}
