// Package apitest serves the real API to the tests of the CLI commands and the MCP, which are clients of it.
package apitest

import (
	"io"
	"os"
	"testing"

	"github.com/cyril-fl/callback/config"
	"github.com/cyril-fl/callback/core/services/testserver"
)

// Serve starts the API on a base of its own and points CALLBACK_URL and CALLBACK_SESSION at it, logged in.
func Serve(t *testing.T) *testserver.Server {
	t.Helper()
	server := testserver.Serve(t)
	// The session file of the user, under HOME, stays untouched.
	t.Setenv("HOME", t.TempDir())
	t.Setenv(config.Env("URL"), server.URL)
	t.Setenv(config.Env("SESSION"), server.Session)
	return server
}

// Stdout returns what print writes to stdout.
func Stdout(t *testing.T, print func() error) (string, error) {
	t.Helper()
	out, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	before := os.Stdout
	os.Stdout = writer
	defer func() { os.Stdout = before }()

	printErr := print()
	writer.Close()
	printed, _ := io.ReadAll(out)
	return string(printed), printErr
}
