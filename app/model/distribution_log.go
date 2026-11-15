package model

import "time"

// DistributionLog merepresentasikan catatan audit jejak status kupon distribusi.
type DistributionLog struct {
	ID             int       `json:"id"`
	DistributionID int       `json:"distribution_id"`
	ActorID        int       `json:"actor_id"`
	FromStatus     *string   `json:"from_status,omitempty"`
	ToStatus       string    `json:"to_status"`
	Note           *string   `json:"note,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}
