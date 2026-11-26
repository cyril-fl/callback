// Package applists reads and writes the interviews and contacts of applications, by their ids: the callers check
// the account.
package applists

import (
	"context"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/uptrace/bun"
)

// Tables are the lists' tables, for the deletes that go by application_id.
var Tables = []string{"interviews", "contacts"}

// Load fills the lists of applications.
func Load(ctx context.Context, db bun.IDB, applications []domain.Application) error {
	if len(applications) == 0 {
		return nil
	}
	index := map[string]*domain.Application{}
	ids := make([]string, len(applications))
	for i := range applications {
		application := &applications[i]
		application.Interviews, application.Contacts = []domain.Interview{}, []domain.Contact{}
		index[application.ID], ids[i] = application, application.ID
	}
	var interviews []domain.Interview
	if err := db.NewSelect().Model(&interviews).Where("application_id IN (?)", bun.List(ids)).Order("at", "id").Scan(ctx); err != nil {
		return err
	}
	var contacts []domain.Contact
	if err := db.NewSelect().Model(&contacts).Where("application_id IN (?)", bun.List(ids)).Order("id").Scan(ctx); err != nil {
		return err
	}
	for _, interview := range interviews {
		index[interview.ApplicationID].Interviews = append(index[interview.ApplicationID].Interviews, interview)
	}
	for _, contact := range contacts {
		index[contact.ApplicationID].Contacts = append(index[contact.ApplicationID].Contacts, contact)
	}
	return nil
}

// Save writes the lists of applications in place of the ones they had.
func Save(ctx context.Context, tx bun.IDB, applications []domain.Application) error {
	if len(applications) == 0 {
		return nil
	}
	ids := make([]string, len(applications))
	var interviews []domain.Interview
	var contacts []domain.Contact
	for i, application := range applications {
		ids[i] = application.ID
		for _, interview := range application.Interviews {
			interview.ID, interview.ApplicationID = 0, application.ID
			interviews = append(interviews, interview)
		}
		for _, contact := range application.Contacts {
			contact.ID, contact.ApplicationID = 0, application.ID
			contacts = append(contacts, contact)
		}
	}
	for _, table := range Tables {
		if _, err := tx.NewRaw("DELETE FROM ? WHERE application_id IN (?)", bun.Ident(table), bun.List(ids)).Exec(ctx); err != nil {
			return err
		}
	}
	if len(interviews) > 0 {
		if _, err := tx.NewInsert().Model(&interviews).Exec(ctx); err != nil {
			return err
		}
	}
	if len(contacts) > 0 {
		if _, err := tx.NewInsert().Model(&contacts).Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}
