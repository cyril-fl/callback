package createschema_test

import (
	"context"
	"testing"

	"github.com/cyril-fl/jat/core/services/process/createschema"
	"github.com/cyril-fl/jat/core/services/testdb"
)

// The schema runs at every start: a second run on a base that has it changes nothing and fails on nothing.
func TestRunTwice(t *testing.T) {
	db := testdb.New(t)
	again := createschema.Process(context.Background(), db)
	again.Run()
	if again.HasErr() {
		t.Fatalf("second run: %v", again.GetErr())
	}
}

// A column a model gained is added to the table of an older base.
func TestAddColumn(t *testing.T) {
	db := testdb.New(t)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `ALTER TABLE applications DROP COLUMN comment`); err != nil {
		t.Fatal(err)
	}
	again := createschema.Process(ctx, db)
	again.Run()
	if again.HasErr() {
		t.Fatalf("run: %v", again.GetErr())
	}
	var count int
	if err := db.NewRaw(`SELECT count(*) FROM pragma_table_info('applications') WHERE name = 'comment'`).Scan(ctx, &count); err != nil || count != 1 {
		t.Fatalf("comment column: %d, %v", count, err)
	}
}
