package domain

import (
	"time"

	"github.com/uptrace/bun"
)

// StatusChange is an application reaching a status: written with the application and at each change of its status,
// deleted with it. The funnel and the response delays of the dashboard come from them.
type StatusChange struct {
	bun.BaseModel `bun:"table:status_changes,alias:sc" json:"-"`

	ID            int64     `bun:",pk,autoincrement" json:"-"`
	ApplicationID string    `bun:",notnull" json:"application_id"`
	Status        Status    `bun:",notnull" json:"status" enums:"applied,responded,interview,offer,rejected"`
	At            time.Time `bun:",notnull" json:"at"`
}
