package domain

import (
	"time"

	"github.com/uptrace/bun"
)

// StatusChange is an application reaching a status: written with the application and at each change of its status,
// deleted with it. The funnel and the response delays of the dashboard come from them.
type StatusChange struct {
	bun.BaseModel `bun:"table:status_changes,alias:sc" json:"-"`

	ID            int64  `bun:",pk,autoincrement" json:"-"`
	ApplicationID string `bun:",notnull" json:"application_id"`
	// Company is read from the application, to tell the changes apart without reading the applications.
	Company string `bun:",scanonly" json:"company,omitempty" binding:"optional"`
	// From is the status before the change, empty when the application was created with Status.
	From   Status    `bun:",nullzero" json:"from,omitempty" enums:"applied,responded,interview,offer,rejected"`
	Status Status    `bun:",notnull" json:"status" enums:"applied,responded,interview,offer,rejected"`
	At     time.Time `bun:",notnull" json:"at"`
}
