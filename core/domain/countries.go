package domain

import (
	_ "embed"
	"strings"

	"gopkg.in/yaml.v3"
)

// ErrInvalidCountry is a country that is neither a code nor an English name of countries.yaml.
var ErrInvalidCountry = Invalid("country is an ISO 3166-1 alpha-2 code (IT) or its English name (Italy): GET /api/countries")

// Country is a country an application can name: Code is stored, Key is the web app's i18n key of its name.
type Country struct {
	Code string `yaml:"code" json:"code" example:"IT"`
	Name string `yaml:"name" json:"name" example:"Italy"`
	Key  string `yaml:"key" json:"key" example:"countries.IT"`
}

//go:embed countries.yaml
var countriesYAML []byte

// Countries are the countries of countries.yaml, by code.
var Countries = loadCountries()

// countryCodes are the codes by their code or English name, lowercase.
var countryCodes = map[string]string{}

func loadCountries() []Country {
	var countries []Country
	if err := yaml.Unmarshal(countriesYAML, &countries); err != nil {
		panic("countries.yaml: " + err.Error())
	}
	for _, c := range countries {
		countryCodes[strings.ToLower(c.Code)] = c.Code
		countryCodes[strings.ToLower(c.Name)] = c.Code
	}
	return countries
}

// CountryCode is the code of a country given by its code or English name, whatever the case; "" stays "".
// A value it does not know is returned as is, for Validate to refuse.
func CountryCode(value string) string {
	value = strings.TrimSpace(value)
	if code, ok := countryCodes[strings.ToLower(value)]; ok {
		return code
	}
	return value
}

// IsCountry tells a stored country: empty or a code.
func IsCountry(value string) bool {
	return value == "" || countryCodes[strings.ToLower(value)] == value
}

// KeepCountry stores a's country as its code, or moves a country it does not know (free text from a base or a backup
// older than Countries) to the start of the comment rather than lose it. It tells whether it changed a.
func (a *Application) KeepCountry() bool {
	code := CountryCode(a.Country)
	if IsCountry(code) {
		changed := code != a.Country
		a.Country = code
		return changed
	}
	a.Comment = strings.TrimSpace("Country: " + a.Country + "\n" + a.Comment)
	a.Country = ""
	return true
}
