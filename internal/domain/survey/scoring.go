package survey

type ScoringEngine struct{}

func NewScoringEngine() *ScoringEngine {
	return &ScoringEngine{}
}

type ScoreSummary struct {
	CSAT float64 `json:"csat"` // Percentage of 4-5 stars or positive answers
	NPS  float64 `json:"nps"`  // net promoter score: % promoters - % detractors
}

func (e *ScoringEngine) CalculateCSATAndNPS(lines []InputLine) ScoreSummary {
	var totalRatings int
	var positiveRatings int // For CSAT: 4 or 5
	var promoters int       // For NPS: 9 or 10
	var detractors int      // For NPS: 0 to 6

	for _, line := range lines {
		// Assuming standard rating fields or score values are stored in ValueScore
		if line.ValueScore > 0 {
			totalRatings++
			// CSAT assume scale of 1-5
			if line.ValueScore >= 4 && line.ValueScore <= 5 {
				positiveRatings++
			}
			// NPS assume scale of 0-10
			if line.ValueScore >= 9 {
				promoters++
			} else if line.ValueScore <= 6 {
				detractors++
			}
		}
	}

	summary := ScoreSummary{}
	if totalRatings > 0 {
		summary.CSAT = (float64(positiveRatings) / float64(totalRatings)) * 100.0
		promoPct := (float64(promoters) / float64(totalRatings)) * 100.0
		detractPct := (float64(detractors) / float64(totalRatings)) * 100.0
		summary.NPS = promoPct - detractPct
	}

	return summary
}
