package organization

import (
	"context"
	"regexp"
	"sort"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/models"
	sdkorganization "github.com/appwrite/sdk-for-go/v7/organization"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
)

const membershipResourceName = "appwrite_organization_membership"

// Organization memberships are authorized with an organization key's
// organization.memberships.* scopes. Only an organization owner can create a
// key holding organization.memberships.write.
var membershipScopes = []string{"organization.memberships.read", "organization.memberships.write"}

// The server's limits on a membership's roles: at most 100 of them, each a
// key of up to 81 characters.
const (
	maxRoles      = 100
	maxRoleLength = 81
	maxUserIDLen  = 36
	maxNameLength = 128
)

// keyPattern is the server's rule for a role and a user ID: letters, digits,
// periods, hyphens and underscores, not starting with one of the last three.
var keyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// emailPattern only catches what is plainly not an address, such as a missing
// "@" or embedded whitespace. The server validates the rest.
var emailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+$`)

func membershipClient(clients *common.AppwriteClients, organizationID string) (*sdkorganization.Organization, error) {
	if err := common.ValidateOrganizationCredential(clients, membershipResourceName, membershipScopes...); err != nil {
		return nil, err
	}
	// The organization routes carry no organization ID in their path; the
	// server resolves it from X-Appwrite-Organization, which this client sets.
	return appwrite.NewOrganization(clients.ClientForOrganization(organizationID)), nil
}

// membershipError formats an API error for a diagnostic, adding what the
// server's message leaves out for the errors this resource is likely to meet.
func membershipError(err error) string {
	formatted := common.FormatErrorWithAuthGuidance(err, common.OrganizationCredentialGuidance(membershipResourceName, membershipScopes...))
	switch common.ErrorType(err) {
	case "membership_downgrade_prohibited":
		return formatted + "\n\nAppwrite requires an organization to keep at least one confirmed owner, so it refuses " +
			"to remove the owner role from the last one. Give another member the owner role first."
	case "membership_deletion_prohibited":
		return formatted + "\n\nAppwrite refuses to remove an organization's last confirmed owner, and the member " +
			"whose payment method the organization is billed to. Give another member the owner role, or change the " +
			"organization's payment method, first."
	case "membership_already_confirmed":
		return formatted + "\n\nThe account is already a member of this organization. To manage that membership, " +
			"import it with `terraform import " + membershipResourceName + ".<name> <organization_id>/<membership_id>`."
	default:
		return formatted
	}
}

// createMembership adds a member and returns the membership as the server
// holds it afterwards.
//
// The server reuses a membership it already has for the account. A pending
// invitation sent from the Console is confirmed in place and keeps the roles
// it was sent with, so the requested roles are applied with an update when
// they differ.
func createMembership(client *sdkorganization.Organization, plan membershipResourceModel, roles []string) (*models.Membership, error) {
	var opts []sdkorganization.CreateMembershipOption
	if v := plan.Email; !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
		opts = append(opts, client.WithCreateMembershipEmail(v.ValueString()))
	}
	if v := plan.UserID; !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
		opts = append(opts, client.WithCreateMembershipUserId(v.ValueString()))
	}
	if v := plan.Name; !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
		opts = append(opts, client.WithCreateMembershipName(v.ValueString()))
	}

	created, err := client.CreateMembership(roles, opts...)
	if err != nil {
		return nil, err
	}
	if sameRoles(created.Roles, roles) {
		return created, nil
	}
	return client.UpdateMembership(created.Id, roles)
}

// updateMembership sends the roles when they changed. Otherwise there is
// nothing to send, since a change to email letter case or to name is not kept
// on the membership, and the membership is read back instead.
func updateMembership(client *sdkorganization.Organization, membershipID string, roles []string, rolesChanged bool) (*models.Membership, error) {
	if rolesChanged {
		return client.UpdateMembership(membershipID, roles)
	}
	return client.GetMembership(membershipID)
}

// planRoles reads the planned roles in a stable order, so requests are
// reproducible.
func planRoles(ctx context.Context, set types.Set) ([]string, diag.Diagnostics) {
	var roles []string
	diags := set.ElementsAs(ctx, &roles, false)
	sort.Strings(roles)
	return roles, diags
}

func sameRoles(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	seen := make(map[string]int, len(a))
	for _, role := range a {
		seen[role]++
	}
	for _, role := range b {
		if seen[role] == 0 {
			return false
		}
		seen[role]--
	}
	return true
}
