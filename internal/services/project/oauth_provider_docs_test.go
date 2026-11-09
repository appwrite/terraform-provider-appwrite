package project

import (
	"os"
	"strings"
	"testing"
)

// oauthProviderDocsTable renders the provider table the resource
// documentation carries.
func oauthProviderDocsTable() string {
	var b strings.Builder
	b.WriteString("| `provider_id` | `client_id` in the API | `client_secret` in the API | Other arguments |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, p := range oauthProviders {
		extras := make([]string, 0, len(p.extras))
		for _, e := range p.extras {
			extras = append(extras, "`"+e+"`")
		}
		row := "| `" + p.id + "` | `" + p.clientIDParam() + "` | `" + p.secretParam() + "` |"
		if len(extras) > 0 {
			row += " " + strings.Join(extras, ", ") + " |"
		} else {
			row += " |"
		}
		b.WriteString(row + "\n")
	}
	return b.String()
}

// TestOAuthProviderDocsTable keeps the documentation's provider table in step
// with oauthProviders, so a provider or argument added to one cannot be
// missing from the other.
func TestOAuthProviderDocsTable(t *testing.T) {
	t.Parallel()

	const template = "../../../templates/resources/project_oauth_provider.md.tmpl"
	raw, err := os.ReadFile(template)
	if err != nil {
		t.Fatal(err)
	}
	if want := oauthProviderDocsTable(); !strings.Contains(string(raw), want) {
		t.Errorf("%s does not contain the provider table. Replace it with:\n\n%s", template, want)
	}
}
