// Package config is the CLI's commands, their help and flags (config.yaml, built in), and the app's name.
package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime/debug"
	"strings"

	plumcliconfig "github.com/cyril-fl/plum/cli/config"
	plummillload "github.com/cyril-fl/plum/services/factory/millload"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// version is set by the images (-ldflags -X), whose build context has no .git to read the commit from.
var version string

// Version comes from the build: the tag of an image, the git commit, or the module version.
func Version() string {
	if version != "" {
		return version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			return setting.Value
		}
	}
	return info.Main.Version
}

// embedded is the config built into the binary, so it runs from any folder; CONFIG_FILE overrides it.
//
//go:embed config.yaml
var embedded []byte

var Current config

type config struct {
	Name     string   `yaml:"name"`
	Commands commands `yaml:"commands"`
}

type commands struct {
	ServerRoot    plumcliconfig.Cmdparams `yaml:"server-root"`
	Serve         plumcliconfig.Cmdparams `yaml:"serve"`
	ResetPassword plumcliconfig.Cmdparams `yaml:"reset-password"`
	Backup        plumcliconfig.Cmdparams `yaml:"backup"`

	Root     plumcliconfig.Cmdparams `yaml:"root"`
	Login    plumcliconfig.Cmdparams `yaml:"login"`
	Logout   plumcliconfig.Cmdparams `yaml:"logout"`
	Whoami   plumcliconfig.Cmdparams `yaml:"whoami"`
	Add      plumcliconfig.Cmdparams `yaml:"add"`
	Ls       plumcliconfig.Cmdparams `yaml:"ls"`
	Show     plumcliconfig.Cmdparams `yaml:"show"`
	Edit     plumcliconfig.Cmdparams `yaml:"edit"`
	Rm       plumcliconfig.Cmdparams `yaml:"rm"`
	Stats    plumcliconfig.Cmdparams `yaml:"stats"`
	Token    plumcliconfig.Cmdparams `yaml:"token"`
	TokenAdd plumcliconfig.Cmdparams `yaml:"token-add"`
	TokenLs  plumcliconfig.Cmdparams `yaml:"token-ls"`
	TokenRm  plumcliconfig.Cmdparams `yaml:"token-rm"`
	MCP      plumcliconfig.Cmdparams `yaml:"mcp"`
}

func load() (*config, error) {
	file := strings.TrimSpace(os.Getenv("CONFIG_FILE"))
	if file == "" {
		var current config
		if err := yaml.Unmarshal(embedded, &current); err != nil {
			return nil, fmt.Errorf("embedded config: %w", err)
		}
		return &current, nil
	}

	loader := plummillload.New(file)
	current := loader.LoadFromFile[config]()
	if loader.HasErr() {
		return nil, loader.GetErr()
	}
	return current, nil
}

// Env is the name of the app's environment variable key: CALLBACK_<key>.
func Env(key string) string {
	return strings.ToUpper(Current.Name) + "_" + key
}

// Dir is the user's folder of the app, ~/.config/callback: its session and its .env.
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	return filepath.Join(home, ".config", Current.Name), err
}

// expand writes the name into the texts of the commands, which say {name} for the program and {NAME} for the prefix
// of its environment variables.
func (c *config) expand() {
	replacer := strings.NewReplacer("{name}", c.Name, "{NAME}", strings.ToUpper(c.Name))
	commands := reflect.ValueOf(&c.Commands).Elem()
	for i := range commands.NumField() {
		params := commands.Field(i).Addr().Interface().(*plumcliconfig.Cmdparams)
		params.Use, params.Short, params.Long, params.Example = replacer.Replace(params.Use), replacer.Replace(params.Short), replacer.Replace(params.Long), replacer.Replace(params.Example)
		for j := range params.Flags {
			params.Flags[j].Usage = replacer.Replace(params.Flags[j].Usage)
		}
	}
}

// loadEnv reads the user's .env first: godotenv never overwrites a value, so the .env of whatever project the CLI
// runs from cannot point it elsewhere.
func loadEnv() {
	if dir, err := Dir(); err == nil {
		godotenv.Load(filepath.Join(dir, ".env"))
	}
	godotenv.Load()
}

func init() {
	if err := yaml.Unmarshal(embedded, &Current); err != nil {
		panic(fmt.Errorf("embedded config: %w", err))
	}
	loadEnv()
	loaded, err := load()
	if err != nil {
		panic(err)
	}
	Current = *loaded
	Current.expand()
}
