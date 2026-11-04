package project

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Appwrite keeps a project's auth methods, policies and mock phone numbers in
// one attribute of the project document, and every route that changes one of
// them reads that attribute, changes its own key and writes the whole
// attribute back. Two such requests in flight at once lose one of the writes:
// against Appwrite 2.2.0, ten mock numbers created concurrently left one
// behind. Terraform applies independent resources in parallel, so the
// resources here serialize their writes per project.
//
// The lock only covers this provider process. A change made in the Console,
// or by another Terraform run, at the same moment can still be lost.
var authSettingsLocks sync.Map // project ID -> *sync.Mutex

// lockAuthSettings holds the project's auth settings lock until the returned
// function is called.
func lockAuthSettings(projectID string) func() {
	value, _ := authSettingsLocks.LoadOrStore(projectID, &sync.Mutex{})
	mutex, _ := value.(*sync.Mutex) // the map holds nothing else
	mutex.Lock()
	return mutex.Unlock
}

// importProjectSettings imports a per-project singleton by project ID. The
// resource ID is the project ID, so both come from the one value.
func importProjectSettings(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID == "" || strings.Contains(req.ID, "/") {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected a project ID, got: %q", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// singletonDescription is shared by the resources that manage settings every
// project already has, so the destroy behavior is described the same way on
// each.
const singletonDescription = "Every project has these settings, so creating this resource takes them over rather " +
	"than creating anything, and destroying it resets them to Appwrite's defaults. An attribute left out of the " +
	"configuration is set to its default too. Use one per project: two would overwrite each other."
