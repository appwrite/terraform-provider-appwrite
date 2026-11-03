package domain

import (
	"context"
	"strings"

	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// mapDomainToState copies a domain into the model the resource and the data
// source share.
//
// Appwrite stores the name with the case it was added with, but DNS treats
// every spelling as the same domain. A spelling already in the model is kept
// when it differs only in case. With none, as after an import, the resource
// stores the lowercase form its configuration is required to use, so a domain
// added through the Console as Example.com does not plan a replacement --
// which would delete its zone -- against a configuration saying example.com.
func mapDomainToState(ctx context.Context, domain *models.Domain, model *domainResourceModel, lowercase bool) diag.Diagnostics {
	model.ID = types.StringValue(domain.Id)

	switch {
	case !model.Domain.IsNull() && !model.Domain.IsUnknown() && strings.EqualFold(model.Domain.ValueString(), domain.Domain):
	case lowercase:
		model.Domain = types.StringValue(strings.ToLower(domain.Domain))
	default:
		model.Domain = types.StringValue(domain.Domain)
	}

	model.Registrar = types.StringValue(domain.Registrar)
	model.NameserversVerified = types.BoolValue(nameserversVerified(domain))
	model.AutoRenewal = types.BoolValue(domain.AutoRenewal)
	model.ExpiresAt = types.StringValue(domain.Expire)
	model.CreatedAt = types.StringValue(domain.CreatedAt)
	model.UpdatedAt = types.StringValue(domain.UpdatedAt)

	nameservers, diags := types.ListValueFrom(ctx, types.StringType, appwriteNameservers(domain))
	model.AppwriteNameservers = nameservers
	return diags
}
