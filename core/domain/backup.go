package domain

import "time"

// BackupVersion is the layout of a Backup: a file of another version is refused rather than half read.
const BackupVersion = 1

var ErrInvalidBackup = Invalid("not a callback backup of this version")

// Backup is everything the account wrote, as one JSON file: GET /api/export gives it, POST /api/import reads it.
// The settings, the tokens and the sessions stay out: they belong to the base, not to the data.
type Backup struct {
	Version       int                  `json:"version" example:"1"`
	ExportedAt    time.Time            `json:"exported_at"`
	Applications  []Application        `json:"applications"`
	Deleted       []DeletedApplication `json:"deleted"`
	StatusChanges []StatusChange       `json:"status_changes"`
	FollowUps     []FollowUp           `json:"follow_ups"`
}
