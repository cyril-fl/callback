// Package listapplications reads applications: a filtered, sorted page, some columns only, comments cut.
package listapplications

import (
	"context"
	"math"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/cyril-fl/callback/core/domain"
	"github.com/cyril-fl/callback/core/services/appfilter"
	"github.com/cyril-fl/callback/core/services/applists"
	plumprimitive "github.com/cyril-fl/plum/primitive"
	"github.com/uptrace/bun"
)

// DefaultSort is the newest applications first.
const DefaultSort = "-applied_at"

// Sorts are the columns a list sorts on, - first for the descending order.
var Sorts = []string{"applied_at", "company", "location", "source", "status", "rating", "followed_up_at", "created_at", "updated_at", "position"}

// Lists are the fields read from their own tables: asked by fields, or there when no fields are.
var Lists = []string{"interviews", "contacts"}

var (
	ErrInvalidField  = domain.Invalid("fields are among id, company, title, location, website, source, source_detail, status, rating, comment, applied_at, interview_at, contact_name, contact_email, interviews, contacts, created_at, updated_at, followed_up_at, position")
	ErrInvalidSort   = domain.Invalid("sort is one of applied_at, company, location, source, status, rating, followed_up_at, created_at, updated_at, position, - first to reverse it")
	ErrInvalidPaging = domain.Invalid("limit, offset and max_chars are 0 or more")
)

type Input struct {
	domain.Filter
	// Fields are the columns returned, all when empty: id always comes.
	Fields []string
	// MaxChars cuts each comment to this many characters, 0 keeps them whole.
	MaxChars int
	Sort     string
	// Limit is the most applications returned, 0 for all.
	Limit, Offset int
	// CountOnly gives the total and no application.
	CountOnly bool
}

// Result is a page of applications and the count of the whole list.
type Result struct {
	Applications []domain.Application
	Total        int
}

type run struct {
	plumprimitive.Process
	cache

	ctx   context.Context
	db    bun.IDB
	input Input
}

type cache struct {
	result Result
}

func Process(ctx context.Context, db bun.IDB, input Input) *run {
	return &run{
		ctx:   ctx,
		db:    db,
		input: input,
	}
}

func (p *run) Run() {
	p.ensureInput()
	p.countApplications()
	p.loadApplications()
	p.cutComments()
}

// Getters & Setters
func (p *run) GetResult() Result {
	return p.cache.result
}

// Methods
func (p *run) ensureInput() {
	if err := p.input.Filter.Validate(); err != nil {
		p.SetErr(err)
		return
	}
	for _, field := range p.input.Fields {
		if !slices.Contains(domain.Columns, field) && !slices.Contains(Lists, field) {
			p.SetErr(ErrInvalidField)
			return
		}
	}
	if p.input.Sort == "" {
		p.input.Sort = DefaultSort
	}
	if !slices.Contains(Sorts, strings.TrimPrefix(p.input.Sort, "-")) {
		p.SetErr(ErrInvalidSort)
		return
	}
	if p.input.Limit < 0 || p.input.Offset < 0 || p.input.MaxChars < 0 {
		p.SetErr(ErrInvalidPaging)
	}
}

func (p *run) countApplications() {
	if p.HasErr() {
		return
	}
	total, err := p.db.NewSelect().Model((*domain.Application)(nil)).ApplyQueryBuilder(appfilter.Where(p.ctx, p.input.Filter)).Count(p.ctx)
	p.cache.result.Total = int(total)
	p.SetErr(err)
}

func (p *run) loadApplications() {
	if p.HasErr() || p.input.CountOnly {
		return
	}
	query := p.db.NewSelect().Model(&p.cache.result.Applications).ApplyQueryBuilder(appfilter.Where(p.ctx, p.input.Filter))
	if len(p.input.Fields) > 0 {
		query = query.Column(columns(p.input.Fields)...)
	}
	column, descending := strings.CutPrefix(p.input.Sort, "-")
	order := " ASC"
	if descending {
		order = " DESC"
	}
	if column == "status" {
		// The pipeline's order, not the alphabet's.
		query = query.OrderExpr("CASE a.status WHEN 'applied' THEN 0 WHEN 'responded' THEN 1 WHEN 'interview' THEN 2 WHEN 'offer' THEN 3 ELSE 4 END" + order)
	} else if column == "position" {
		// NULL stands for 0, and the two bases sort NULL apart.
		query = query.OrderExpr("COALESCE(a.position, 0)" + order)
	} else {
		query = query.OrderExpr("a.?"+order, bun.Ident(column))
	}
	query = query.OrderExpr("a.created_at DESC, a.id")
	if p.input.Limit > 0 {
		query = query.Limit(int64(p.input.Limit))
	}
	if p.input.Offset > 0 {
		// SQLite needs a LIMIT before an OFFSET.
		if p.input.Limit == 0 {
			query = query.Limit(math.MaxInt64)
		}
		query = query.Offset(int64(p.input.Offset))
	}
	if err := query.Scan(p.ctx); err != nil {
		p.SetErr(err)
		return
	}
	if len(p.input.Fields) == 0 || slices.ContainsFunc(p.input.Fields, func(field string) bool { return slices.Contains(Lists, field) }) {
		p.SetErr(applists.Load(p.ctx, p.db, p.cache.result.Applications))
	}
}

func (p *run) cutComments() {
	if p.HasErr() || p.input.MaxChars == 0 {
		return
	}
	for i := range p.cache.result.Applications {
		application := &p.cache.result.Applications[i]
		if utf8.RuneCountInString(application.Comment) > p.input.MaxChars {
			application.Comment = string([]rune(application.Comment)[:p.input.MaxChars])
			application.Truncated = true
		}
	}
}

// Helpers

// columns are the fields asked, id first and once.
func columns(fields []string) []string {
	result := []string{"id"}
	for _, field := range fields {
		if !slices.Contains(result, field) && !slices.Contains(Lists, field) {
			result = append(result, field)
		}
	}
	return result
}
