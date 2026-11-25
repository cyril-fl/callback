package domain

import (
	"time"

	"github.com/uptrace/bun"
)

// FollowUp is the user following an application up (a call, an e-mail), written by an update with followed_up and
// deleted with the application. The application keeps the last one in followed_up_at.
type FollowUp struct {
	bun.BaseModel `bun:"table:follow_ups,alias:fu" json:"-"`

	ID            int64     `bun:",pk,autoincrement" json:"-"`
	ApplicationID string    `bun:",notnull" json:"application_id"`
	At            time.Time `bun:",notnull" json:"at"`
}
