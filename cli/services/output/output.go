// Package output prints the answers of the commands: JSON for scripts, aligned tables for people.
package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

// JSON prints v on one line, for jq and scripts.
func JSON(v any) error {
	return json.NewEncoder(os.Stdout).Encode(v)
}

// Table prints rows under header, the columns aligned.
func Table(header []string, rows [][]string) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, strings.ToUpper(strings.Join(header, "\t")))
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	return w.Flush()
}
