package common

import (
	"strings"

	"github.com/appwrite/sdk-for-go/v7/client"
)

// Values the provider reports in the SDK identity headers. Appwrite reads
// x-sdk-name to attribute API key usage and x-sdk-language to classify
// deployments, so Terraform traffic must not report itself as the Go SDK.
const (
	SDKName     = "Terraform"
	SDKPlatform = "server"
	SDKLanguage = "terraform"
)

// WithIdentity returns a ClientOption that identifies requests as coming from
// this provider rather than the Go SDK it is built on.
//
// The User-Agent follows the HashiCorp convention of Terraform core, then the
// provider, then the underlying client, so the SDK version stays traceable. The
// Terraform token is omitted when the version is unknown. TF_APPEND_USER_AGENT
// is appended when set.
//
// It must be applied through appwrite.NewClient, which sets the SDK's own
// User-Agent before running options.
func WithIdentity(providerVersion string, terraformVersion string) client.ClientOption {
	return func(clt *client.Client) error {
		tokens := make([]string, 0, 4)
		if terraformVersion != "" {
			tokens = append(tokens, "Terraform/"+terraformVersion)
		}
		tokens = append(tokens, "terraform-provider-appwrite/"+providerVersion)
		if sdk := clt.Headers["user-agent"]; sdk != "" {
			tokens = append(tokens, sdk)
		}
		if appended := AppendedUserAgent(); appended != "" {
			tokens = append(tokens, appended)
		}

		clt.Headers["user-agent"] = strings.Join(tokens, " ")
		clt.Headers["x-sdk-name"] = SDKName
		clt.Headers["x-sdk-platform"] = SDKPlatform
		clt.Headers["x-sdk-language"] = SDKLanguage
		clt.Headers["x-sdk-version"] = providerVersion
		return nil
	}
}
