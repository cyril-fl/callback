package domain

import (
	"cmp"
	"slices"
	"strings"

	"github.com/uptrace/bun"
)

// InterviewKind is who the interview is with.
type InterviewKind string

const (
	InterviewHR      InterviewKind = "hr"
	InterviewManager InterviewKind = "manager"
	InterviewTech    InterviewKind = "tech"
	InterviewOther   InterviewKind = "other"
)

var InterviewKinds = []InterviewKind{InterviewHR, InterviewManager, InterviewTech, InterviewOther}

// MaxList is the most interviews, or contacts, an application has.
const MaxList = 50

var (
	ErrLongList          = Invalid("an application has at most 50 interviews and 50 contacts")
	ErrInvalidKind       = Invalid("an interview's kind is hr, manager, tech or other")
	ErrInvalidContactRow = Invalid("a contact has a name or an e-mail; name, email and role have at most 200 characters, a note 10000")
)

// Interview is one interview of an application, deleted with it. The application keeps the last one in interview_at.
type Interview struct {
	bun.BaseModel `bun:"table:interviews,alias:iv" json:"-"`

	ID            int64         `bun:",pk,autoincrement" json:"-"`
	ApplicationID string        `bun:",notnull" json:"-"`
	At            string        `bun:",notnull" json:"at" example:"2026-01-31T14:30"`
	Kind          InterviewKind `bun:",notnull" json:"kind,omitempty" binding:"optional" enums:"hr,manager,tech,other"`
	Note          string        `bun:",nullzero" json:"note,omitempty" binding:"optional"`
}

// Contact is one person of an application, deleted with it. The application keeps the first one in contact_name
// and contact_email.
type Contact struct {
	bun.BaseModel `bun:"table:contacts,alias:ct" json:"-"`

	ID            int64  `bun:",pk,autoincrement" json:"-"`
	ApplicationID string `bun:",notnull" json:"-"`
	Name          string `bun:",nullzero" json:"name,omitempty" binding:"optional"`
	Email         string `bun:",nullzero" json:"email,omitempty" binding:"optional"`
	Role          string `bun:",nullzero" json:"role,omitempty" binding:"optional"`
}

// normalizeLists trims the lists and sorts the interviews by date. Lists left empty are built from interview_at
// and the contact, so a client that only knows those still writes them; then those columns follow the lists.
func (a *Application) normalizeLists() {
	if len(a.Interviews) == 0 && a.InterviewAt != "" {
		a.Interviews = []Interview{{At: a.InterviewAt}}
	}
	if len(a.Contacts) == 0 && (a.ContactName != "" || a.ContactEmail != "") {
		a.Contacts = []Contact{{Name: a.ContactName, Email: a.ContactEmail}}
	}
	for i := range a.Interviews {
		interview := &a.Interviews[i]
		interview.At, interview.Note = strings.TrimSpace(interview.At), strings.TrimSpace(interview.Note)
		if interview.Kind == "" {
			interview.Kind = InterviewOther
		}
	}
	slices.SortStableFunc(a.Interviews, func(x, y Interview) int { return cmp.Compare(x.At, y.At) })
	for i := range a.Contacts {
		contact := &a.Contacts[i]
		contact.Name, contact.Email, contact.Role = strings.TrimSpace(contact.Name), strings.TrimSpace(contact.Email), strings.TrimSpace(contact.Role)
	}
	a.InterviewAt, a.ContactName, a.ContactEmail = "", "", ""
	if n := len(a.Interviews); n > 0 {
		a.InterviewAt = a.Interviews[n-1].At
	}
	if len(a.Contacts) > 0 {
		a.ContactName, a.ContactEmail = a.Contacts[0].Name, a.Contacts[0].Email
	}
}

// listErrors are the errors of the lists.
func (a *Application) listErrors() []error {
	var errs []error
	if len(a.Interviews) > MaxList || len(a.Contacts) > MaxList {
		errs = append(errs, ErrLongList)
	}
	for _, interview := range a.Interviews {
		if !isDateTime(interview.At) || runes(interview.Note) > MaxComment {
			errs = append(errs, ErrInvalidInterview)
		}
		if !slices.Contains(InterviewKinds, interview.Kind) {
			errs = append(errs, ErrInvalidKind)
		}
	}
	for _, contact := range a.Contacts {
		if contact.Name == "" && contact.Email == "" || runes(contact.Name) > MaxShort || runes(contact.Role) > MaxShort || runes(contact.Email) > MaxShort {
			errs = append(errs, ErrInvalidContactRow)
		}
		if contact.Email != "" && !isEmail(contact.Email) {
			errs = append(errs, ErrInvalidContact)
		}
	}
	return slices.CompactFunc(errs, func(x, y error) bool { return x == y })
}

// applyLists writes the lists of the changes. Without a list, interview_at edits the last interview ("" removes it)
// and contact_name or contact_email the first contact (both "" remove it).
func (c Changes) applyLists(a *Application) {
	if !c.ChangesLists() {
		return
	}
	switch {
	case c.Interviews != nil:
		a.Interviews = *c.Interviews
	case c.InterviewAt != nil:
		n := len(a.Interviews)
		switch {
		case *c.InterviewAt == "" && n > 0:
			a.Interviews = a.Interviews[:n-1]
		case *c.InterviewAt == "":
		case n == 0:
			a.Interviews = []Interview{{At: *c.InterviewAt}}
		default:
			a.Interviews[n-1].At = *c.InterviewAt
		}
	}
	switch {
	case c.Contacts != nil:
		a.Contacts = *c.Contacts
	case c.ContactName != nil || c.ContactEmail != nil:
		first, rest := Contact{}, []Contact(nil)
		if len(a.Contacts) > 0 {
			first, rest = a.Contacts[0], a.Contacts[1:]
		}
		set(&first.Name, c.ContactName)
		set(&first.Email, c.ContactEmail)
		if first.Name == "" && first.Email == "" {
			a.Contacts = rest
		} else {
			a.Contacts = append([]Contact{first}, rest...)
		}
	}
	// The columns come back from the lists in Normalize.
	a.InterviewAt, a.ContactName, a.ContactEmail = "", "", ""
}

// ChangesLists tells changes that rewrite the interviews or the contacts.
func (c Changes) ChangesLists() bool {
	return c.Interviews != nil || c.Contacts != nil || c.InterviewAt != nil || c.ContactName != nil || c.ContactEmail != nil
}
