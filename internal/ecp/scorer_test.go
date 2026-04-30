package ecp

import (
	"testing"
)

func TestCalculate(t *testing.T) {
	tests := []struct {
		name          string
		profile       StudentProfile
		wantScoreMin  int
		wantScoreMax  int
		wantTier      string
		wantAcademic  int
		wantFinancial int
		wantLoanReady int
		wantErr       bool
	}{
		{
			name: "Perfect profile — max everything",
			profile: StudentProfile{
				Name: "Perfect", CGPA: 10.0, GREScore: 340,
				FamilyIncome: Income20LPlus, HasCoApplicant: true, CoApplicantIncome: Income20LPlus,
				TargetCountry: CountryUSA, TargetProgram: ProgramMBA, Intake: IntakeJan2026,
			},
			// academic: min(30, 20+10)=30, financial: min(40, 36+round(36*0.25)+4)=min(40,36+9+4)=40 (capped)
			// loanReadiness: min(30, 10+10+10)=30 → total=min(100,100)=100
			wantScoreMin: 100, wantScoreMax: 100,
			wantTier: "Green", wantAcademic: 30, wantFinancial: 40, wantLoanReady: 30,
		},
		{
			name: "Minimal profile — lowest everything",
			profile: StudentProfile{
				Name: "Minimal", CGPA: 4.0, GREScore: 0,
				FamilyIncome: IncomeLess3L, HasCoApplicant: false,
				TargetCountry: CountryIndia, TargetProgram: ProgramMArch, Intake: IntakeJan2027,
			},
			// academic: min(30, round(4/10*20)+1) = min(30,8+1)=9
			// financial: min(40, 8)=8
			// loanReadiness: min(30, 6+6+5)=17
			// total: 9+8+17=34
			wantScoreMin: 30, wantScoreMax: 40,
			wantTier: "Red", wantAcademic: 9, wantFinancial: 8, wantLoanReady: 17,
		},
		{
			name: "Priya — CGPA 7.8, GRE 312, 8-20L, co-applicant",
			profile: StudentProfile{
				Name: "Priya Sharma", CGPA: 7.8, GREScore: 312,
				FamilyIncome: Income8To20L, HasCoApplicant: true, CoApplicantIncome: Income8To20L,
				TargetCountry: CountryUSA, TargetProgram: ProgramMS, Intake: IntakeSep2026,
			},
			// academic: min(30, round(7.8/10*20)+6) = min(30,16+6)=22
			// financial: min(40, 28+round(28*0.25)+4) = min(40,28+7+4)=39
			// loanReadiness: min(30, 9+10+8)=27
			// total: 22+39+27=88
			wantScoreMin: 85, wantScoreMax: 92,
			wantTier: "Green", wantAcademic: 22, wantFinancial: 39, wantLoanReady: 27,
		},
		{
			name: "Rohan — CGPA 8.1, GRE 318, MBA, USA, Jan2026",
			profile: StudentProfile{
				Name: "Rohan Mehta", CGPA: 8.1, GREScore: 318,
				FamilyIncome: Income8To20L, HasCoApplicant: true, CoApplicantIncome: Income8To20L,
				TargetCountry: CountryUSA, TargetProgram: ProgramMBA, Intake: IntakeJan2026,
			},
			// academic: min(30, round(8.1/10*20)+8) = min(30,16+8)=24
			// financial: same as Priya = 39
			// loanReadiness: min(30, 10+10+10)=30
			// total: 24+39+30=93
			wantScoreMin: 90, wantScoreMax: 96,
			wantTier: "Green", wantAcademic: 24, wantFinancial: 39, wantLoanReady: 30,
		},
		{
			name: "Anjali — CGPA 8.5, no GRE, no co-applicant, Canada, MiM",
			profile: StudentProfile{
				Name: "Anjali Nair", CGPA: 8.5, GREScore: 0,
				FamilyIncome: Income8To20L, HasCoApplicant: false,
				TargetCountry: CountryCanada, TargetProgram: ProgramMiM, Intake: IntakeSep2026,
			},
			// academic: min(30, round(8.5/10*20)+1) = min(30,17+1)=18
			// financial: min(40, 28)=28
			// loanReadiness: min(30, 7+9+8)=24
			// total: 18+28+24=70
			wantScoreMin: 67, wantScoreMax: 73,
			wantTier: "Green", wantAcademic: 18, wantFinancial: 28, wantLoanReady: 24,
		},
		{
			name: "Edge — CGPA exactly 8.0, GRE exactly 305, no co-applicant",
			profile: StudentProfile{
				Name: "Edge Case 1", CGPA: 8.0, GREScore: 305,
				FamilyIncome: Income8To20L, HasCoApplicant: false,
				TargetCountry: CountryUK, TargetProgram: ProgramMS, Intake: IntakeSep2026,
			},
			// academic: min(30, round(8/10*20)+6) = min(30,16+6)=22
			// financial: 28
			// loanReadiness: min(30, 9+9+8)=26
			// total: 22+28+26=76
			wantScoreMin: 73, wantScoreMax: 79,
			wantTier: "Green", wantAcademic: 22, wantFinancial: 28, wantLoanReady: 26,
		},
		{
			name: "Edge — co-applicant with empty income defaults to <3L",
			profile: StudentProfile{
				Name: "Edge Case 2", CGPA: 7.0, GREScore: 300,
				FamilyIncome: Income3To8L, HasCoApplicant: true, CoApplicantIncome: "",
				TargetCountry: CountryGermany, TargetProgram: ProgramMiM, Intake: IntakeJan2027,
			},
			// academic: min(30, round(7/10*20)+4) = min(30,14+4)=18
			// financial: min(40, 18+round(8*0.25)+4) = min(40,18+2+4)=24
			// loanReadiness: min(30, 7+7+5)=19
			// total: 18+24+19=61
			wantScoreMin: 58, wantScoreMax: 64,
			wantTier: "Amber", wantAcademic: 18, wantFinancial: 24, wantLoanReady: 19,
		},
		{
			name: "Edge — GRE = 0 (plans to take)",
			profile: StudentProfile{
				Name: "Edge Case 3", CGPA: 9.0, GREScore: 0,
				FamilyIncome: Income20LPlus, HasCoApplicant: true, CoApplicantIncome: Income20LPlus,
				TargetCountry: CountryUSA, TargetProgram: ProgramMS, Intake: IntakeJan2026,
			},
			// academic: min(30, round(9/10*20)+1) = min(30,18+1)=19
			// financial: min(40, 36+9+4)=40 (capped)
			// loanReadiness: min(30, 9+10+10)=29
			// total: 19+40+29=88
			wantScoreMin: 85, wantScoreMax: 92,
			wantTier: "Green", wantAcademic: 19, wantFinancial: 40, wantLoanReady: 29,
		},
		{
			name: "Edge — income <3L, no co-applicant",
			profile: StudentProfile{
				Name: "Low Income", CGPA: 6.0, GREScore: 280,
				FamilyIncome: IncomeLess3L, HasCoApplicant: false,
				TargetCountry: CountryIndia, TargetProgram: ProgramMPH, Intake: IntakeJan2027,
			},
			// academic: min(30, round(6/10*20)+2) = min(30,12+2)=14
			// financial: 8
			// loanReadiness: min(30, 7+6+5)=18
			// total: 14+8+18=40
			wantScoreMin: 37, wantScoreMax: 43,
			wantTier: "Red", wantAcademic: 14, wantFinancial: 8, wantLoanReady: 18,
		},
		{
			name: "Edge — max everything except low income",
			profile: StudentProfile{
				Name: "High Talent Low Income", CGPA: 10.0, GREScore: 340,
				FamilyIncome: IncomeLess3L, HasCoApplicant: false,
				TargetCountry: CountryUSA, TargetProgram: ProgramMBA, Intake: IntakeJan2026,
			},
			// academic: min(30, 20+10)=30
			// financial: 8
			// loanReadiness: min(30, 10+10+10)=30
			// total: 30+8+30=68
			wantScoreMin: 65, wantScoreMax: 71,
			wantTier: "Amber", wantAcademic: 30, wantFinancial: 8, wantLoanReady: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Calculate(tt.profile)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Score < tt.wantScoreMin || result.Score > tt.wantScoreMax {
				t.Errorf("score = %d, want [%d, %d]", result.Score, tt.wantScoreMin, tt.wantScoreMax)
			}
			if result.Tier != tt.wantTier {
				t.Errorf("tier = %q, want %q (score=%d)", result.Tier, tt.wantTier, result.Score)
			}
			if result.SubScores.Academic != tt.wantAcademic {
				t.Errorf("academic = %d, want %d", result.SubScores.Academic, tt.wantAcademic)
			}
			if result.SubScores.Financial != tt.wantFinancial {
				t.Errorf("financial = %d, want %d", result.SubScores.Financial, tt.wantFinancial)
			}
			if result.SubScores.LoanReadiness != tt.wantLoanReady {
				t.Errorf("loanReadiness = %d, want %d", result.SubScores.LoanReadiness, tt.wantLoanReady)
			}

			// Funding band sanity: lower < upper, both > 0
			if result.FundingBandLower <= 0 || result.FundingBandUpper <= 0 {
				t.Errorf("funding band must be positive: [%d, %d]", result.FundingBandLower, result.FundingBandUpper)
			}
			if result.FundingBandLower > result.FundingBandUpper {
				t.Errorf("funding band inverted: lower=%d > upper=%d", result.FundingBandLower, result.FundingBandUpper)
			}
		})
	}
}

func TestValidation(t *testing.T) {
	tests := []struct {
		name    string
		profile StudentProfile
		wantErr bool
	}{
		{
			name: "CGPA too low",
			profile: StudentProfile{
				CGPA: 3.9, GREScore: 300, FamilyIncome: Income3To8L,
				TargetCountry: CountryUSA, TargetProgram: ProgramMS, Intake: IntakeJan2026,
			},
			wantErr: true,
		},
		{
			name: "CGPA too high",
			profile: StudentProfile{
				CGPA: 10.1, GREScore: 300, FamilyIncome: Income3To8L,
				TargetCountry: CountryUSA, TargetProgram: ProgramMS, Intake: IntakeJan2026,
			},
			wantErr: true,
		},
		{
			name: "GRE out of range (low)",
			profile: StudentProfile{
				CGPA: 8.0, GREScore: 259, FamilyIncome: Income3To8L,
				TargetCountry: CountryUSA, TargetProgram: ProgramMS, Intake: IntakeJan2026,
			},
			wantErr: true,
		},
		{
			name: "GRE out of range (high)",
			profile: StudentProfile{
				CGPA: 8.0, GREScore: 341, FamilyIncome: Income3To8L,
				TargetCountry: CountryUSA, TargetProgram: ProgramMS, Intake: IntakeJan2026,
			},
			wantErr: true,
		},
		{
			name: "Invalid income band",
			profile: StudentProfile{
				CGPA: 8.0, GREScore: 300, FamilyIncome: "50L+",
				TargetCountry: CountryUSA, TargetProgram: ProgramMS, Intake: IntakeJan2026,
			},
			wantErr: true,
		},
		{
			name: "Invalid program type",
			profile: StudentProfile{
				CGPA: 8.0, GREScore: 300, FamilyIncome: Income3To8L,
				TargetCountry: CountryUSA, TargetProgram: "BBA", Intake: IntakeJan2026,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Calculate(tt.profile)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
