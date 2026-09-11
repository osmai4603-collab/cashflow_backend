package livechat

// Rating constants
const (
	RatingSatisfied    = 5
	RatingNeutral      = 3
	RatingDissatisfied = 1
)

// SessionRating summarizes the rating given to a chat session.
type SessionRating struct {
	SessionID int64  `json:"session_id"`
	Score     int    `json:"score"`
	Comment   string `json:"comment"`
}
