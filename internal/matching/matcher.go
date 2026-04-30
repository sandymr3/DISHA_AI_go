package matching

import (
	"disha-backend/internal/ecp"
	"math"
	"sort"
)

// Matcher holds the university dataset and performs matching.
type Matcher struct {
	universities []University
}

// NewMatcher creates a new Matcher with the given university dataset.
func NewMatcher(universities []University) *Matcher {
	return &Matcher{universities: universities}
}

// GetByID returns a single university by ID, or nil if not found.
func (m *Matcher) GetByID(id string) *University {
	for i := range m.universities {
		if m.universities[i].ID == id {
			return &m.universities[i]
		}
	}
	return nil
}

// AllUniversities returns the full dataset.
func (m *Matcher) AllUniversities() []University {
	return m.universities
}

// Match finds and ranks universities for a student profile.
// Returns up to 8 matches sorted by funding status tier then match score.
func (m *Matcher) Match(profile ecp.StudentProfile, ecpResult ecp.ECPResult, filters Filters) []MatchedUniversity {
	fundingUpperINR := int64(ecpResult.FundingBandUpper) * 100000 * 100 // Lakhs → paise

	var matches []MatchedUniversity

	for _, uni := range m.universities {
		// Filter by country
		if filters.Country != "" && uni.Country != filters.Country {
			continue
		}
		// Filter by program type — match at student profile level if no explicit filter
		if filters.Program != "" {
			if uni.ProgramType != filters.Program {
				continue
			}
		} else {
			// Default: match on student's target country AND program type
			if uni.Country != profile.TargetCountry || uni.ProgramType != profile.TargetProgram {
				continue
			}
		}

		// Coverage percentage
		var coveragePercent int
		if uni.TotalCostINR > 0 {
			coveragePercent = int(math.Round(float64(fundingUpperINR) / float64(uni.TotalCostINR) * 100))
		}

		// Funding status
		var fundingStatus string
		switch {
		case coveragePercent >= 90:
			fundingStatus = "Within Band"
		case coveragePercent >= 70:
			fundingStatus = "Stretch Goal"
		default:
			fundingStatus = "Out of Range"
		}

		// Admit probability based on CGPA band
		admitProb := getAdmitProbability(profile.CGPA, uni.AdmitProbability)

		// ROI score: min(100, (1/roiYears)*300)
		roiScore := 100.0
		if uni.ROIYears > 0 {
			roiScore = math.Min(100, (1.0/uni.ROIYears)*300.0)
		}

		// Composite match score
		matchScore := (roiScore * 0.40) + (float64(coveragePercent) * 0.35) + (admitProb * 100.0 * 0.25)
		matchScore = math.Round(matchScore*100) / 100

		matches = append(matches, MatchedUniversity{
			University:       uni,
			CoveragePercent:  coveragePercent,
			FundingStatus:    fundingStatus,
			AdmitProbability: admitProb,
			ROIScore:         math.Round(roiScore*100) / 100,
			MatchScore:       matchScore,
		})
	}

	// Sort: Within Band first, then Stretch Goal, then Out of Range;
	// within each tier by matchScore descending.
	sort.Slice(matches, func(i, j int) bool {
		ti := fundingStatusRank(matches[i].FundingStatus)
		tj := fundingStatusRank(matches[j].FundingStatus)
		if ti != tj {
			return ti < tj
		}
		return matches[i].MatchScore > matches[j].MatchScore
	})

	// Return top 8
	if len(matches) > 8 {
		matches = matches[:8]
	}

	return matches
}

// getAdmitProbability looks up the admit probability based on the student's CGPA.
func getAdmitProbability(cgpa float64, probMap map[string]float64) float64 {
	switch {
	case cgpa >= 9.0:
		return probMap["cgpa_9plus"]
	case cgpa >= 8.0:
		return probMap["cgpa_8to9"]
	case cgpa >= 7.0:
		return probMap["cgpa_7to8"]
	default:
		return probMap["cgpa_below7"]
	}
}

// fundingStatusRank maps funding status to a sort rank (lower = better).
func fundingStatusRank(status string) int {
	switch status {
	case "Within Band":
		return 0
	case "Stretch Goal":
		return 1
	default:
		return 2
	}
}
