// Package cmdapplications is the commands on the applications: callback add, ls, show, edit, rm and stats. Each one is a
// single request to the API, so a batch is one transaction.
package cmdapplications

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/cyril-fl/callback/cli/services/apiclient"
	"github.com/cyril-fl/callback/cli/services/output"
	"github.com/spf13/cobra"
)

// defaultColumns are the columns of ls when --fields names none; id always comes first.
var defaultColumns = []string{"company", "status", "applied_at"}

// showColumns is the order of show, every field.
var showColumns = []string{"company", "location", "street", "postal_code", "country", "website", "source", "source_detail", "status", "rating", "applied_at", "comment", "truncated", "created_at", "updated_at"}

// Add creates the application of the flags, or the batch of --file, in one request.
func Add(cobra *cobra.Command, _ []string) error {
	client, err := apiclient.Read().Client()
	if err != nil {
		return err
	}
	var items []apiclient.DomainNewApplication
	if file, _ := cobra.Flags().GetString("file"); file != "" {
		if items, err = readItems(file); err != nil {
			return err
		}
	} else {
		item := apiclient.DomainNewApplication{}
		item.Company, _ = cobra.Flags().GetString("company")
		item.Location, _ = cobra.Flags().GetString("location")
		set := changes(cobra)
		item.Street, item.PostalCode, item.Country = set.Street, set.PostalCode, set.Country
		item.Website, item.Source, item.SourceDetail, item.Status, item.Rating, item.Comment, item.AppliedAt =
			set.Website, set.Source, set.SourceDetail, set.Status, set.Rating, set.Comment, set.AppliedAt
		items = append(items, item)
	}
	return batch(cobra, "added")(apiclient.Decode[apiclient.DomainBatchResult](client.PostApiApplications(cobra.Context(), items)))
}

// readItems reads the JSON array of file, stdin for -.
func readItems(file string) ([]apiclient.DomainNewApplication, error) {
	var data []byte
	var err error
	if file == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(file)
	}
	if err != nil {
		return nil, err
	}
	var items []apiclient.DomainNewApplication
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("%s: a JSON array of applications: %w", file, err)
	}
	return items, nil
}

// List prints the applications of the filters, a page of them, or their count.
func List(cobra *cobra.Command, _ []string) error {
	client, err := apiclient.Read().Client()
	if err != nil {
		return err
	}
	flags := cobra.Flags()
	f := filters(cobra, "status", "source")
	fields, _ := flags.GetStringSlice("fields")
	if len(fields) == 0 {
		fields = defaultColumns
	}
	maxChars, _ := flags.GetInt("max-chars")
	limit, _ := flags.GetInt("limit")
	offset, _ := flags.GetInt("offset")
	count, _ := flags.GetBool("count")
	asJSON, _ := flags.GetBool("json")
	params := &apiclient.GetApiApplicationsParams{
		Source: (*string)(f.Source), Q: f.Q, AppliedBefore: f.AppliedBefore, AppliedAfter: f.AppliedAfter,
		UntouchedSince: f.UntouchedSince, Sort: opt(flags.GetString("sort")), Limit: &limit, Offset: &offset, CountOnly: &count,
	}
	if f.Status != nil {
		status := make([]apiclient.GetApiApplicationsParamsStatus, 0, len(*f.Status))
		for _, s := range *f.Status {
			status = append(status, apiclient.GetApiApplicationsParamsStatus(s))
		}
		params.Status = &status
	}
	if !count {
		params.Fields = &fields
	}
	if maxChars > 0 {
		params.MaxChars = &maxChars
	}
	applications, total, err := apiclient.Paged[[]apiclient.DomainApplication](client.GetApiApplications(cobra.Context(), params))
	switch {
	case err != nil:
		return err
	case count && asJSON:
		return output.JSON(map[string]int{"total": total})
	case count:
		fmt.Println(total)
		return nil
	case asJSON:
		return output.JSON(applications)
	}
	columns := append([]string{"id"}, fields...)
	rows := make([][]string, 0, len(applications))
	for _, application := range applications {
		values := fieldsOf(application)
		row := make([]string, 0, len(columns))
		for _, column := range columns {
			// A newline or a tab would break the table's rows and columns.
			row = append(row, strings.NewReplacer("\n", " ", "\t", " ").Replace(values[column]))
		}
		rows = append(rows, row)
	}
	if err := output.Table(columns, rows); err != nil {
		return err
	}
	if len(applications) < total {
		fmt.Fprintf(os.Stderr, "%d of %d: --limit and --offset for more\n", len(applications), total)
	}
	return nil
}

// Show prints the applications of the ids whole, in the order given; an id not found is an error.
func Show(cobra *cobra.Command, args []string) error {
	client, err := apiclient.Read().Client()
	if err != nil {
		return err
	}
	all := 0
	params := &apiclient.GetApiApplicationsParams{Ids: &args, Limit: &all}
	if maxChars, _ := cobra.Flags().GetInt("max-chars"); maxChars > 0 {
		params.MaxChars = &maxChars
	}
	found, err := apiclient.Decode[[]apiclient.DomainApplication](client.GetApiApplications(cobra.Context(), params))
	if err != nil {
		return err
	}
	var shown []apiclient.DomainApplication
	var missing []string
	for _, id := range args {
		if i := slices.IndexFunc(found, func(a apiclient.DomainApplication) bool { return a.Id == id }); i >= 0 {
			shown = append(shown, found[i])
		} else {
			missing = append(missing, id)
		}
	}
	if asJSON, _ := cobra.Flags().GetBool("json"); asJSON {
		err = output.JSON(shown)
	} else {
		for i, application := range shown {
			if i > 0 {
				fmt.Println()
			}
			values := fieldsOf(application)
			fmt.Println("id:", application.Id)
			for _, column := range showColumns {
				if values[column] != "" {
					fmt.Printf("%s: %s\n", column, values[column])
				}
			}
		}
	}
	if err == nil && len(missing) > 0 {
		err = fmt.Errorf("not found: %s", strings.Join(missing, ", "))
	}
	return err
}

// Edit sets the fields of the flags on the applications of the ids or the --where filters, in one op.
func Edit(cobra *cobra.Command, args []string) error {
	client, err := apiclient.Read().Client()
	if err != nil {
		return err
	}
	f := filters(cobra, "where-status", "where-source")
	op := apiclient.UpdateapplicationsOp{
		Ids: ids(args), Status: f.Status, Source: f.Source, Q: f.Q, AppliedBefore: f.AppliedBefore, AppliedAfter: f.AppliedAfter,
		UntouchedSince: f.UntouchedSince, Set: changes(cobra),
	}
	if company, _ := cobra.Flags().GetString("company"); cobra.Flags().Changed("company") {
		op.Set.Company = &company
	}
	if location, _ := cobra.Flags().GetString("location"); cobra.Flags().Changed("location") {
		op.Set.Location = &location
	}
	dryRun, _ := cobra.Flags().GetBool("dry-run")
	in := apiclient.UpdateapplicationsInput{Ops: []apiclient.UpdateapplicationsOp{op}, DryRun: &dryRun}
	return batch(cobra, "updated")(apiclient.Decode[apiclient.DomainBatchResult](client.PatchApiApplications(cobra.Context(), in)))
}

// Remove deletes the applications of the ids or the filters.
func Remove(cobra *cobra.Command, args []string) error {
	client, err := apiclient.Read().Client()
	if err != nil {
		return err
	}
	f := filters(cobra, "status", "source")
	dryRun, _ := cobra.Flags().GetBool("dry-run")
	in := apiclient.DeleteapplicationsInput{
		Ids: ids(args), Status: f.Status, Source: f.Source, Q: f.Q, AppliedBefore: f.AppliedBefore, AppliedAfter: f.AppliedAfter,
		UntouchedSince: f.UntouchedSince, DryRun: &dryRun,
	}
	return batch(cobra, "deleted")(apiclient.Decode[apiclient.DomainBatchResult](client.DeleteApiApplications(cobra.Context(), in)))
}

// Stats prints the count per status and the response rate.
func Stats(cobra *cobra.Command, _ []string) error {
	client, err := apiclient.Read().Client()
	if err != nil {
		return err
	}
	stats, err := apiclient.Decode[apiclient.ReadstatsStats](client.GetApiStats(cobra.Context()))
	if err != nil {
		return err
	}
	if asJSON, _ := cobra.Flags().GetBool("json"); asJSON {
		return output.JSON(stats)
	}
	rows := [][]string{}
	for _, status := range []string{"applied", "responded", "interview", "offer", "rejected"} {
		rows = append(rows, []string{status, strconv.Itoa(stats.ByStatus[status])})
	}
	rows = append(rows, []string{"total", strconv.Itoa(stats.Total)}, []string{"response rate", fmt.Sprintf("%.0f%%", stats.ResponseRate*100)})
	return output.Table([]string{"status", "count"}, rows)
}

// errFailed makes the exit code not 0 when an item of a batch failed: the others were written.
var errFailed = errors.New("some items failed")

// batch prints what a batch wrote, its failures on stderr.
func batch(cobra *cobra.Command, done string) func(apiclient.DomainBatchResult, error) error {
	return func(result apiclient.DomainBatchResult, err error) error {
		if err != nil {
			return err
		}
		if asJSON, _ := cobra.Flags().GetBool("json"); asJSON {
			err = output.JSON(result)
		} else {
			if result.DryRun != nil && *result.DryRun {
				done = "would be " + done + " (dry run)"
			}
			fmt.Printf("%d %s\n", result.Count, done)
			if result.Ids != nil {
				for _, id := range *result.Ids {
					fmt.Println(id)
				}
			}
		}
		if result.Failed == nil || len(*result.Failed) == 0 {
			return err
		}
		for _, failure := range *result.Failed {
			at := ""
			switch {
			case failure.Index != nil:
				at = fmt.Sprintf("item %d: ", *failure.Index)
			case failure.Id != nil:
				at = *failure.Id + ": "
			case failure.Op != nil:
				at = fmt.Sprintf("op %d: ", *failure.Op)
			}
			fmt.Fprintf(os.Stderr, "failed: %s%s\n", at, failure.Error)
		}
		return fmt.Errorf("%w: %d", errFailed, len(*result.Failed))
	}
}

// filter is the filter flags, nil when not given: the API reads a missing filter as matching all.
type filter struct {
	Status                                         *[]apiclient.DomainStatus
	Source                                         *apiclient.DomainSource
	Q, AppliedBefore, AppliedAfter, UntouchedSince *string
}

// filters reads the filter flags; edit names its status and source filters apart from the fields it sets.
func filters(cobra *cobra.Command, statusFlag, sourceFlag string) filter {
	flags := cobra.Flags()
	f := filter{
		Source: (*apiclient.DomainSource)(opt(flags.GetString(sourceFlag))),
		Q:      opt(flags.GetString("query")), AppliedBefore: opt(flags.GetString("applied-before")),
		AppliedAfter: opt(flags.GetString("applied-after")), UntouchedSince: opt(flags.GetString("untouched-since")),
	}
	if statuses, _ := flags.GetStringSlice(statusFlag); len(statuses) > 0 {
		status := make([]apiclient.DomainStatus, 0, len(statuses))
		for _, s := range statuses {
			status = append(status, apiclient.DomainStatus(s))
		}
		f.Status = &status
	}
	return f
}

// changes is the fields the flags set, the ones not given left nil; company and location are the caller's, add needs
// them as plain strings.
func changes(cobra *cobra.Command) apiclient.DomainChanges {
	flags := cobra.Flags()
	text := func(name string) *string {
		if !flags.Changed(name) {
			return nil
		}
		value, _ := flags.GetString(name)
		return &value
	}
	set := apiclient.DomainChanges{
		Website: text("website"), Street: text("street"), PostalCode: text("postal-code"), Country: text("country"), Source: (*apiclient.DomainSource)(text("source")), SourceDetail: text("source-detail"),
		Status: (*apiclient.DomainStatus)(text("status")), Comment: text("comment"), AppliedAt: text("applied-at"),
	}
	if flags.Changed("rating") {
		rating, _ := flags.GetInt("rating")
		set.Rating = &rating
	}
	return set
}

func ids(args []string) *[]string {
	if len(args) == 0 {
		return nil
	}
	return &args
}

// opt is nil for an empty flag.
func opt(value string, _ error) *string {
	if value == "" {
		return nil
	}
	return &value
}

// fieldsOf is the application's fields as text, by their JSON name, for the table and show.
func fieldsOf(application apiclient.DomainApplication) map[string]string {
	data, _ := json.Marshal(application)
	var values map[string]any
	_ = json.Unmarshal(data, &values)
	out := map[string]string{}
	for key, value := range values {
		out[key] = fmt.Sprint(value)
	}
	return out
}
