package domain

import "slices"

// The plans of the web app: base keeps the interface to the list and its form, pro adds the dashboard and the tools.
const (
	PlanBase = "base"
	PlanPro  = "pro"
)

// Limits of FollowUpDays.
const (
	MinFollowUpDays = 1
	MaxFollowUpDays = 90
	MaxNoReplyDays  = 365
	MinTrashDays    = 1
	MaxTrashDays    = 365
)

var (
	ErrInvalidPlan     = Invalid("plan is base or pro")
	ErrInvalidFollowUp = Invalid("follow_up_days is between 1 and 90")
	ErrInvalidNoReply  = Invalid("no_reply_days is between 0 and 365")
	ErrInvalidTrash    = Invalid("trash_days is between 1 and 365")
	ErrLongDefault     = Invalid("default_location has at most 200 characters")
	ErrInvalidTheme    = Invalid("theme is lilac, pink, blue or green")
	ErrInvalidMode     = Invalid("color_mode is system, light or dark")
)

// Themes repeats the ids of web/app/utils/themes.ts (preferences_test.go checks it) and ColorModes the modes of the
// web app, the first one by default.
var (
	Themes     = []string{"lilac", "pink", "blue", "green"}
	ColorModes = []string{"system", "light", "dark"}
)

// Preferences are the settings of an account that follow it on every device.
type Preferences struct {
	Plan string `json:"plan" enums:"base,pro"`
	// Theme and ColorMode are the colors of the web app.
	Theme     string `json:"theme" enums:"lilac,pink,blue,green"`
	ColorMode string `json:"color_mode" enums:"system,light,dark"`
	// DefaultLocation and DefaultCountry (a code of Countries) fill a new application of the web app; both may be empty.
	DefaultLocation string `json:"default_location"`
	DefaultCountry  string `json:"default_country"`
	// FollowUpDays is how long an application waits without news before it is to follow up.
	FollowUpDays int `json:"follow_up_days"`
	// NoReplyDays is how long a waiting application stays without news (since its last follow-up, status change or
	// interview, else the day it applied) before the server moves it to no_reply; 0 never does.
	NoReplyDays int `json:"no_reply_days"`
	// TrashDays is how long a deleted application can be restored, before it is gone with its history.
	TrashDays int `json:"trash_days"`
}

// DefaultPreferences are the preferences of an account that never saved any.
var DefaultPreferences = Preferences{Plan: PlanBase, Theme: Themes[0], ColorMode: ColorModes[0], FollowUpDays: 7, NoReplyDays: 10, TrashDays: 30}

// Validate checks preferences as they will be stored.
func (p Preferences) Validate() error {
	switch {
	case p.Plan != PlanBase && p.Plan != PlanPro:
		return ErrInvalidPlan
	case !slices.Contains(Themes, p.Theme):
		return ErrInvalidTheme
	case !slices.Contains(ColorModes, p.ColorMode):
		return ErrInvalidMode
	case p.FollowUpDays < MinFollowUpDays || p.FollowUpDays > MaxFollowUpDays:
		return ErrInvalidFollowUp
	case p.NoReplyDays < 0 || p.NoReplyDays > MaxNoReplyDays:
		return ErrInvalidNoReply
	case p.TrashDays < MinTrashDays || p.TrashDays > MaxTrashDays:
		return ErrInvalidTrash
	case runes(p.DefaultLocation) > MaxShort:
		return ErrLongDefault
	case !IsCountry(p.DefaultCountry):
		return ErrInvalidCountry
	}
	return nil
}

// PreferenceChanges are the preferences an update sets; one left out keeps its value.
type PreferenceChanges struct {
	Plan            *string `json:"plan,omitempty" binding:"optional" enums:"base,pro"`
	Theme           *string `json:"theme,omitempty" binding:"optional" enums:"lilac,pink,blue,green"`
	ColorMode       *string `json:"color_mode,omitempty" binding:"optional" enums:"system,light,dark"`
	DefaultLocation *string `json:"default_location,omitempty" binding:"optional"`
	DefaultCountry  *string `json:"default_country,omitempty" binding:"optional"`
	FollowUpDays    *int    `json:"follow_up_days,omitempty" binding:"optional"`
	NoReplyDays     *int    `json:"no_reply_days,omitempty" binding:"optional"`
	TrashDays       *int    `json:"trash_days,omitempty" binding:"optional"`
}

// Apply writes the changes into p.
func (c PreferenceChanges) Apply(p *Preferences) {
	set(&p.Plan, c.Plan)
	set(&p.Theme, c.Theme)
	set(&p.ColorMode, c.ColorMode)
	set(&p.DefaultLocation, c.DefaultLocation)
	set(&p.DefaultCountry, c.DefaultCountry)
	p.DefaultCountry = CountryCode(p.DefaultCountry)
	set(&p.FollowUpDays, c.FollowUpDays)
	set(&p.NoReplyDays, c.NoReplyDays)
	set(&p.TrashDays, c.TrashDays)
}
