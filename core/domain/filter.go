package domain

import (
	"slices"
	"time"
)

// MaxIDs is the most ids one request names.
const MaxIDs = 500

var (
	ErrTooManyIDs    = Invalid("at most 500 ids per request")
	ErrInvalidFilter = Invalid("applied_before, applied_after and untouched_since look like 2026-01-31")
	ErrNoFilter      = Invalid("name ids or at least one filter: an empty filter would match every application")
)

// Filter picks applications. Every field left out matches all; together they all must match.
type Filter struct {
	// IDs are the applications by id.
	IDs []string `json:"ids,omitempty" binding:"optional"`
	// Status is any of these statuses.
	Status []Status `json:"status,omitempty" binding:"optional"`
	Source Source   `json:"source,omitempty" binding:"optional" enums:"spontaneous,job_post,recruiter"`
	// Q is text found in the company, location, website, source_detail or comment, case-insensitive.
	Q string `json:"q,omitempty" binding:"optional"`
	// AppliedBefore and AppliedAfter are strict: applied_at < before, applied_at > after.
	AppliedBefore string `json:"applied_before,omitempty" binding:"optional" example:"2026-01-31"`
	AppliedAfter  string `json:"applied_after,omitempty" binding:"optional" example:"2026-01-01"`
	// UntouchedSince is the applications not updated since this day.
	UntouchedSince string `json:"untouched_since,omitempty" binding:"optional" example:"2026-01-01"`
}

// Empty tells a filter that matches every application.
func (f Filter) Empty() bool {
	return len(f.IDs) == 0 && len(f.Status) == 0 && f.Source == "" && f.Q == "" && f.AppliedBefore == "" && f.AppliedAfter == "" && f.UntouchedSince == ""
}

// Validate checks the values of the filter.
func (f Filter) Validate() error {
	if len(f.IDs) > MaxIDs {
		return ErrTooManyIDs
	}
	for _, status := range f.Status {
		if !slices.Contains(Statuses, status) {
			return ErrInvalidStatus
		}
	}
	if f.Source != "" && !slices.Contains(Sources, f.Source) {
		return ErrInvalidSource
	}
	for _, date := range []string{f.AppliedBefore, f.AppliedAfter, f.UntouchedSince} {
		if date != "" && !IsDate(date) {
			return ErrInvalidFilter
		}
	}
	return nil
}

// UntouchedTime is the start of the UntouchedSince day, in UTC.
func (f Filter) UntouchedTime() time.Time {
	day, _ := time.Parse(DateLayout, f.UntouchedSince)
	return day
}
