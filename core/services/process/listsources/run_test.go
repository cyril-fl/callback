package listsources_test

import (
	"testing"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/process/listsources"
	"github.com/cyril-fl/callback/core/services/seed"
	"github.com/cyril-fl/callback/core/services/testdb"
)

func TestRun(t *testing.T) {
	db := testdb.New(t)
	ctx := testdb.Account(t, db, "test")
	seed.Applications(t, ctx, db,
		domain.NewApplication{Company: "A", Location: "Paris", Source: "Facebook"},
		domain.NewApplication{Company: "B", Location: "Paris", Source: "Facebook"},
		domain.NewApplication{Company: "C", Location: "Paris"},
	)
	process := listsources.Process(ctx, db)
	process.Run()
	got := process.GetResult()
	if process.HasErr() || len(got) != 2 || got[0] != (listsources.Source{Source: "Facebook", Count: 2}) || got[1].Source != domain.SourceSpontaneous {
		t.Errorf("sources = %+v, %v; want Facebook first", got, process.GetErr())
	}
}
