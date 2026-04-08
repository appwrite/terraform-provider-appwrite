package acceptance

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/appwrite/terraform-provider-appwrite/internal/provider"
)

var ProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"appwrite": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func PreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("APPWRITE_ENDPOINT") == "" {
		t.Fatal("APPWRITE_ENDPOINT must be set for acceptance tests")
	}
	if os.Getenv("APPWRITE_PROJECT_ID") == "" {
		t.Fatal("APPWRITE_PROJECT_ID must be set for acceptance tests")
	}
	if os.Getenv("APPWRITE_API_KEY") == "" {
		t.Fatal("APPWRITE_API_KEY must be set for acceptance tests")
	}
}
