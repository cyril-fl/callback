// Package domain holds the models: their bun tags are the schema, their json tags the API's JSON and the web app's types.
package domain

import (
	"errors"
	"net/mail"
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/uptrace/bun"
)

// Source is how the application started: one of the known sources below, or any name the user gives (Facebook,
// a temp agency...).
type Source string

// The known sources, whose source_detail is checked: a job post's URL, a recruiter's e-mail, nothing when spontaneous.
const (
	SourceSpontaneous Source = "spontaneous"
	SourceJobPost     Source = "job_post"
	SourceRecruiter   Source = "recruiter"
)

var Sources = []Source{SourceSpontaneous, SourceJobPost, SourceRecruiter}

// MaxSource is the most characters of a source's name.
const MaxSource = 60

// Status is where the application stands in the pipeline.
type Status string

const (
	StatusApplied   Status = "applied"
	StatusResponded Status = "responded"
	StatusInterview Status = "interview"
	StatusOffer     Status = "offer"
	StatusRejected  Status = "rejected"
)

var Statuses = []Status{StatusApplied, StatusResponded, StatusInterview, StatusOffer, StatusRejected}

// DateLayout is the layout of applied_at and of the date filters.
const DateLayout = time.DateOnly

// DateTimeLayout is the layout of interview_at: a local date and time, no zone.
const DateTimeLayout = "2006-01-02T15:04"

// Limits of the free text, in runes.
const (
	MaxShort   = 200
	MaxComment = 10000
)

var (
	ErrNoCompany        = Invalid("company is required")
	ErrNoLocation       = Invalid("location (the city) is required")
	ErrLong             = Invalid("company, location, street, postal_code, country, website, source_detail, contact_name and contact_email have at most 200 characters, comment 10000")
	ErrInvalidWebsite   = Invalid("website must be an http(s) URL or an e-mail")
	ErrInvalidSource    = Invalid("source is spontaneous, job_post, recruiter or a name of at most 60 characters")
	ErrInvalidDetail    = Invalid("source_detail must be the job post's http(s) URL for job_post, the recruiter's e-mail for recruiter, empty for spontaneous; free for another source")
	ErrInvalidStatus    = Invalid("status must be one of applied, responded, interview, offer, rejected")
	ErrInvalidRating    = Invalid("rating must be between 0 and 5")
	ErrInvalidDate      = Invalid("dates must look like 2026-01-31")
	ErrInvalidInterview = Invalid("interview_at must look like 2026-01-31T14:30")
	ErrInvalidContact   = Invalid("contact_email must be an e-mail")
)

// Application is a job application. Text is stored raw: the web app escapes it when it shows it.
// Every field but id may be left out of a response that asks for some fields only (?fields=).
// The columns have no SQL default: a batch insert leaves out a defaulted column when its first row has the zero value.
type Application struct {
	bun.BaseModel `bun:"table:applications,alias:a" json:"-"`

	ID      string `bun:",pk" json:"id"`
	Company string `bun:",notnull" json:"company,omitempty" binding:"optional"`
	// Location is the city; Street, PostalCode and Country complete the address when known.
	Location   string `bun:",notnull" json:"location,omitempty" binding:"optional"`
	Street     string `bun:",nullzero" json:"street,omitempty" binding:"optional"`
	PostalCode string `bun:",nullzero" json:"postal_code,omitempty" binding:"optional"`
	Country    string `bun:",nullzero" json:"country,omitempty" binding:"optional"`
	// Website is the company's site, or an e-mail to write to when it has none.
	Website      string `bun:",notnull" json:"website,omitempty" binding:"optional"`
	Source       Source `bun:",notnull" json:"source,omitempty" binding:"optional" example:"job_post" swaggertype:"string"`
	SourceDetail string `bun:",notnull" json:"source_detail,omitempty" binding:"optional"`
	Status       Status `bun:",notnull" json:"status,omitempty" binding:"optional" enums:"applied,responded,interview,offer,rejected"`
	Rating       int    `bun:",notnull" json:"rating,omitempty" binding:"optional"`
	Comment      string `bun:",notnull" json:"comment,omitempty" binding:"optional"`
	AppliedAt    string `bun:",notnull" json:"applied_at,omitempty" binding:"optional" example:"2026-01-31"`
	// InterviewAt is the next interview's local date and time.
	InterviewAt  string    `bun:",nullzero" json:"interview_at,omitempty" binding:"optional" example:"2026-01-31T14:30"`
	ContactName  string    `bun:",nullzero" json:"contact_name,omitempty" binding:"optional"`
	ContactEmail string    `bun:",nullzero" json:"contact_email,omitempty" binding:"optional"`
	CreatedAt    time.Time `bun:",notnull" json:"created_at,omitzero" binding:"optional"`
	UpdatedAt    time.Time `bun:",notnull" json:"updated_at,omitzero" binding:"optional"`
	// FollowedUpAt is the last follow-up, set by an update with followed_up.
	FollowedUpAt time.Time `bun:",nullzero" json:"followed_up_at,omitzero" binding:"optional"`
	// Truncated tells that comment was cut at max_chars.
	Truncated bool `bun:"-" json:"truncated,omitempty" binding:"optional"`
}

// Columns are the columns ?fields= may ask for.
var Columns = []string{"id", "company", "location", "street", "postal_code", "country", "website", "source", "source_detail", "status", "rating", "comment", "applied_at", "interview_at", "contact_name", "contact_email", "created_at", "updated_at", "followed_up_at"}

// Normalize trims the text and fills the defaults: status applied, applied today.
func (a *Application) Normalize() {
	a.Company = strings.TrimSpace(a.Company)
	a.Location = strings.TrimSpace(a.Location)
	a.Street = strings.TrimSpace(a.Street)
	a.PostalCode = strings.TrimSpace(a.PostalCode)
	a.Country = strings.TrimSpace(a.Country)
	a.Source = Source(strings.TrimSpace(string(a.Source)))
	a.Website = strings.TrimSpace(a.Website)
	a.SourceDetail = strings.TrimSpace(a.SourceDetail)
	a.InterviewAt = strings.TrimSpace(a.InterviewAt)
	a.ContactName = strings.TrimSpace(a.ContactName)
	a.ContactEmail = strings.TrimSpace(a.ContactEmail)
	if a.Source == "" {
		a.Source = SourceSpontaneous
	}
	if a.Status == "" {
		a.Status = StatusApplied
	}
	if a.AppliedAt == "" {
		a.AppliedAt = time.Now().UTC().Format(DateLayout)
	}
}

// Validate checks an application as it will be stored, and returns every error it finds, joined.
func (a *Application) Validate() error {
	checks := []struct {
		bad bool
		err error
	}{
		{a.Company == "", ErrNoCompany},
		{a.Location == "", ErrNoLocation},
		{runes(a.Company) > MaxShort || runes(a.Location) > MaxShort || runes(a.Street) > MaxShort || runes(a.PostalCode) > MaxShort || runes(a.Country) > MaxShort || runes(a.Website) > MaxShort || runes(a.SourceDetail) > MaxShort || runes(a.ContactName) > MaxShort || runes(a.ContactEmail) > MaxShort || runes(a.Comment) > MaxComment, ErrLong},
		{a.Website != "" && !isURL(a.Website) && !isEmail(a.Website), ErrInvalidWebsite},
		{runes(string(a.Source)) > MaxSource, ErrInvalidSource},
		{!validDetail(a.Source, a.SourceDetail), ErrInvalidDetail},
		{!slices.Contains(Statuses, a.Status), ErrInvalidStatus},
		{a.Rating < 0 || a.Rating > 5, ErrInvalidRating},
		{!IsDate(a.AppliedAt), ErrInvalidDate},
		{a.InterviewAt != "" && !isDateTime(a.InterviewAt), ErrInvalidInterview},
		{a.ContactEmail != "" && !isEmail(a.ContactEmail), ErrInvalidContact},
	}
	var errs []error
	for _, check := range checks {
		if check.bad {
			errs = append(errs, check.err)
		}
	}
	return errors.Join(errs...)
}

// KnownSources are the source names in use, to write a source the way it is already written.
type KnownSources struct {
	exact  map[Source]bool
	folded map[string]Source
}

// NewKnownSources knows the built-in sources and sources.
func NewKnownSources(sources []Source) KnownSources {
	known := KnownSources{exact: map[Source]bool{}, folded: map[string]Source{}}
	for _, source := range append(slices.Clone(Sources), sources...) {
		known.Match(source)
	}
	return known
}

// Match returns the known name that source equals but for case, else source, then knows it: "linkedin" reuses
// "LinkedIn", and a batch agrees with itself. A name already in use as written is kept, so a case can be fixed.
func (k KnownSources) Match(source Source) Source {
	key := strings.ToLower(string(source))
	if name, ok := k.folded[key]; ok && !k.exact[source] {
		return name
	}
	k.exact[source] = true
	if _, ok := k.folded[key]; !ok {
		k.folded[key] = source
	}
	return source
}

// IsDate tells a date of DateLayout.
func IsDate(value string) bool {
	_, err := time.Parse(DateLayout, value)
	return err == nil
}

func isDateTime(value string) bool {
	_, err := time.Parse(DateTimeLayout, value)
	return err == nil
}

func validDetail(source Source, detail string) bool {
	switch source {
	case SourceJobPost:
		return isURL(detail)
	case SourceRecruiter:
		return isEmail(detail)
	case SourceSpontaneous:
		return detail == ""
	default:
		return true
	}
}

// isEmail tells a bare address, not "Name <address>".
func isEmail(value string) bool {
	address, err := mail.ParseAddress(value)
	return err == nil && address.Address == value
}

func isURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func runes(value string) int {
	return utf8.RuneCountInString(value)
}

// NewApplication is an application to create: what is left out gets its default (Normalize).
type NewApplication struct {
	Company      string `json:"company"`
	Location     string `json:"location"`
	Street       string `json:"street,omitempty" binding:"optional"`
	PostalCode   string `json:"postal_code,omitempty" binding:"optional"`
	Country      string `json:"country,omitempty" binding:"optional"`
	Website      string `json:"website,omitempty" binding:"optional"`
	Source       Source `json:"source,omitempty" binding:"optional" example:"job_post" swaggertype:"string"`
	SourceDetail string `json:"source_detail,omitempty" binding:"optional"`
	Status       Status `json:"status,omitempty" binding:"optional" enums:"applied,responded,interview,offer,rejected"`
	Rating       int    `json:"rating,omitempty" binding:"optional"`
	Comment      string `json:"comment,omitempty" binding:"optional"`
	AppliedAt    string `json:"applied_at,omitempty" binding:"optional" example:"2026-01-31"`
	InterviewAt  string `json:"interview_at,omitempty" binding:"optional" example:"2026-01-31T14:30"`
	ContactName  string `json:"contact_name,omitempty" binding:"optional"`
	ContactEmail string `json:"contact_email,omitempty" binding:"optional"`
}

// Application is the new application, not yet stored: no id nor timestamps.
func (n NewApplication) Application() Application {
	return Application{Company: n.Company, Location: n.Location, Street: n.Street, PostalCode: n.PostalCode, Country: n.Country, Website: n.Website, Source: n.Source, SourceDetail: n.SourceDetail, Status: n.Status, Rating: n.Rating, Comment: n.Comment, AppliedAt: n.AppliedAt, InterviewAt: n.InterviewAt, ContactName: n.ContactName, ContactEmail: n.ContactEmail}
}

// Changes are the fields an update sets; a field left out keeps its value.
type Changes struct {
	Company      *string `json:"company,omitempty" binding:"optional"`
	Location     *string `json:"location,omitempty" binding:"optional"`
	Street       *string `json:"street,omitempty" binding:"optional"`
	PostalCode   *string `json:"postal_code,omitempty" binding:"optional"`
	Country      *string `json:"country,omitempty" binding:"optional"`
	Website      *string `json:"website,omitempty" binding:"optional"`
	Source       *Source `json:"source,omitempty" binding:"optional" example:"job_post" swaggertype:"string"`
	SourceDetail *string `json:"source_detail,omitempty" binding:"optional"`
	Status       *Status `json:"status,omitempty" binding:"optional" enums:"applied,responded,interview,offer,rejected"`
	Rating       *int    `json:"rating,omitempty" binding:"optional"`
	Comment      *string `json:"comment,omitempty" binding:"optional"`
	AppliedAt    *string `json:"applied_at,omitempty" binding:"optional" example:"2026-01-31"`
	// InterviewAt, ContactName and ContactEmail are cleared by "".
	InterviewAt  *string `json:"interview_at,omitempty" binding:"optional" example:"2026-01-31T14:30"`
	ContactName  *string `json:"contact_name,omitempty" binding:"optional"`
	ContactEmail *string `json:"contact_email,omitempty" binding:"optional"`
	// FollowedUp records a follow-up now: followed_up_at becomes now, and the status counts as news again.
	FollowedUp bool `json:"followed_up,omitempty" binding:"optional"`
	// UndoStatus takes the status back to the one before its last change and forgets that change: a status set by
	// mistake leaves no step in the history.
	UndoStatus bool `json:"undo_status,omitempty" binding:"optional"`
	// UndoFollowUp forgets the last follow-up: followed_up_at goes back to the one before, if any.
	UndoFollowUp bool `json:"undo_follow_up,omitempty" binding:"optional"`
}

var (
	ErrNoChanges      = Invalid("set changes nothing: give at least one field")
	ErrUndoConflict   = Invalid("undo_status goes without status, undo_follow_up without followed_up")
	ErrNoStatusChange = Invalid("no status change to undo")
	ErrNoFollowUp     = Invalid("no follow-up to undo")
)

// Empty tells changes that set nothing.
func (c Changes) Empty() bool {
	return c == Changes{}
}

// Apply writes the changes into a.
func (c Changes) Apply(a *Application) {
	set(&a.Company, c.Company)
	set(&a.Location, c.Location)
	set(&a.Street, c.Street)
	set(&a.PostalCode, c.PostalCode)
	set(&a.Country, c.Country)
	set(&a.Website, c.Website)
	set(&a.Source, c.Source)
	set(&a.SourceDetail, c.SourceDetail)
	set(&a.Status, c.Status)
	set(&a.Rating, c.Rating)
	set(&a.Comment, c.Comment)
	set(&a.AppliedAt, c.AppliedAt)
	set(&a.InterviewAt, c.InterviewAt)
	set(&a.ContactName, c.ContactName)
	set(&a.ContactEmail, c.ContactEmail)
}

func set[T any](field *T, value *T) {
	if value != nil {
		*field = *value
	}
}
