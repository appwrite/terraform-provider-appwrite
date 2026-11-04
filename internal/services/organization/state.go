package organization

import (
	"context"

	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// mapMembershipToState copies a membership into the model.
//
// email is the address the member was added with, and is kept as it is in the
// model once set. Appwrite matches it to an account without regard to case and
// stores the lowercase form, and the account's address can change later
// without the membership changing. Refreshing it from the account would turn
// either into a planned replacement, which would remove the member and add
// them again under an address they may no longer use. user_email reports the
// account's current address instead. With no email in the model, as after an
// import or when the member was added by user_id, the account's address is
// stored.
func mapMembershipToState(ctx context.Context, membership *models.Membership, model *membershipResourceModel) diag.Diagnostics {
	model.ID = types.StringValue(membership.Id)
	model.UserID = types.StringValue(membership.UserId)
	model.UserName = types.StringValue(membership.UserName)
	model.UserEmail = types.StringValue(membership.UserEmail)
	model.Confirm = types.BoolValue(membership.Confirm)
	model.JoinedAt = types.StringValue(membership.Joined)

	if model.Email.IsNull() || model.Email.IsUnknown() || model.Email.ValueString() == "" {
		model.Email = types.StringValue(membership.UserEmail)
	}

	roles := membership.Roles
	if roles == nil {
		roles = []string{}
	}
	set, diags := types.SetValueFrom(ctx, types.StringType, roles)
	model.Roles = set
	return diags
}
