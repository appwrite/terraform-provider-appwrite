package common

import (
	"testing"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
)

// Requests used to report x-sdk-name "Go", so Appwrite attributed provider
// traffic to the Go SDK and only the User-Agent told them apart.
func TestWithIdentityReplacesSDKHeaders(t *testing.T) {
	t.Setenv("TF_APPEND_USER_AGENT", "")

	clt := appwrite.NewClient(WithIdentity("2.1.0", "1.9.5"))

	want := map[string]string{
		"x-sdk-name":     "Terraform",
		"x-sdk-platform": "server",
		"x-sdk-language": "terraform",
		"x-sdk-version":  "2.1.0",
	}
	for name, value := range want {
		if got := clt.Headers[name]; got != value {
			t.Errorf("%s = %q, want %q", name, got, value)
		}
	}
}

func TestWithIdentityUserAgent(t *testing.T) {
	sdk := appwrite.NewClient().Headers["user-agent"]

	tests := []struct {
		name      string
		terraform string
		appended  string
		want      string
	}{
		{
			name:      "full",
			terraform: "1.9.5",
			appended:  "terragrunt/0.55.0",
			want:      "Terraform/1.9.5 terraform-provider-appwrite/2.1.0 " + sdk + " terragrunt/0.55.0",
		},
		{
			name: "unknown terraform version",
			want: "terraform-provider-appwrite/2.1.0 " + sdk,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("TF_APPEND_USER_AGENT", test.appended)

			clt := appwrite.NewClient(WithIdentity("2.1.0", test.terraform))
			if got := clt.Headers["user-agent"]; got != test.want {
				t.Errorf("user-agent = %q, want %q", got, test.want)
			}
		})
	}
}
