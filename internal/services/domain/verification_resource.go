package domain

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/appwrite/sdk-for-go/v7/client"
	"github.com/appwrite/sdk-for-go/v7/domains"
	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
)

const verificationResourceName = "appwrite_domain_nameservers_verification"

// defaultVerificationTimeout bounds the wait when the configuration sets no
// timeouts block. A registrar usually publishes a nameserver change within
// minutes, but resolvers can hold the old delegation for the parent zone's TTL,
// commonly up to two days, so a longer wait is left to the configuration.
const defaultVerificationTimeout = 30 * time.Minute

// verificationInterval is how long to wait between checks. Each check makes
// Appwrite query DNS, so polling faster would not see a change any sooner.
var verificationInterval = 30 * time.Second

var (
	_ resource.Resource              = &verificationResource{}
	_ resource.ResourceWithConfigure = &verificationResource{}
)

type verificationResource struct {
	clients *common.AppwriteClients
}

type verificationResourceModel struct {
	ID             types.String   `tfsdk:"id"`
	DomainID       types.String   `tfsdk:"domain_id"`
	OrganizationID types.String   `tfsdk:"organization_id"`
	Timeouts       timeouts.Value `tfsdk:"timeouts"`
}

func NewVerificationResource() resource.Resource {
	return &verificationResource{}
}

func (r *verificationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_nameservers_verification"
}

func (r *verificationResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Waits for Appwrite to confirm that a domain is delegated to its nameservers. Appwrite only " +
			"checks delegation when asked, so this asks until the check passes or the create timeout expires. " +
			"It changes nothing at the registrar: point the domain at appwrite_domain.appwrite_nameservers first.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   domainIDDescription,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"domain_id": schema.StringAttribute{
				Description:   "The ID of the domain to verify. " + common.ForcesReplacementNote,
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"organization_id": common.OrganizationIDAttribute(),
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true}),
		},
	}
}

func (r *verificationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	clients, ok := req.ProviderData.(*common.AppwriteClients)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *common.AppwriteClients, got: %T", req.ProviderData))
		return
	}
	r.clients = clients
}

func (r *verificationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan verificationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel, diags := common.CreateContext(ctx, plan.Timeouts, defaultVerificationTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	defer cancel()

	organizationID, err := common.ResolveOrganizationID(r.clients, plan.OrganizationID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving organization ID", err.Error())
		return
	}
	client, err := domainsClient(r.clients, verificationResourceName, organizationID)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	domain, err := waitForDelegation(ctx, client, plan.DomainID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error verifying domain nameservers", err.Error())
		return
	}

	plan.ID = types.StringValue(domain.Id)
	plan.OrganizationID = types.StringValue(organizationID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// waitForDelegation asks Appwrite to check the domain's delegation until it
// passes or ctx expires.
//
// A wait can last hours, so a rate limit, a server error or a dropped
// connection on one check is waited out rather than failing the apply. Any
// other error -- a credential, a missing domain -- will not pass by asking
// again and is returned at once.
func waitForDelegation(ctx context.Context, client *domains.Domains, domainID string) (*models.Domain, error) {
	var last *models.Domain
	var lastErr error
	for {
		domain, err := client.VerifyNameservers(domainID)
		switch {
		case err == nil && nameserversVerified(domain):
			return domain, nil
		case err == nil:
			last, lastErr = domain, nil
		case isTransient(err):
			lastErr = err
		default:
			return nil, errors.New(common.FormatErrorWithAuthGuidance(err, credentialGuidance(verificationResourceName)))
		}

		select {
		case <-ctx.Done():
			if last == nil {
				return nil, fmt.Errorf("timed out waiting to verify the nameservers of domain %q: %w; last error: %s",
					domainID, ctx.Err(), common.FormatError(lastErr))
			}
			return nil, fmt.Errorf(
				"domain %q is not delegated to Appwrite's nameservers (%s); timed out waiting: %w. "+
					"Set them at the domain's registrar, then apply again. A registrar can take several "+
					"hours to publish the change; raise the create timeout to wait longer",
				last.Domain, strings.Join(appwriteNameservers(last), ", "), ctx.Err())
		case <-time.After(verificationInterval):
		}
	}
}

// isTransient reports whether a failed check is worth repeating: a rate limit,
// a server error, or no response at all.
func isTransient(err error) bool {
	var appErr *client.AppwriteError
	if !errors.As(err, &appErr) {
		return true
	}
	status := appErr.GetStatusCode()
	return status == http.StatusTooManyRequests || status >= http.StatusInternalServerError
}

// Read keeps the resource while the domain exists. It does not recheck
// delegation: the check is a write that Appwrite records, and a refresh should
// not change anything.
func (r *verificationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state verificationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationID, err := common.ResolveOrganizationID(r.clients, state.OrganizationID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving organization ID", err.Error())
		return
	}
	client, err := domainsClient(r.clients, verificationResourceName, organizationID)
	if err != nil {
		resp.Diagnostics.AddError("Unsupported Appwrite credential", err.Error())
		return
	}

	if _, err := client.Get(state.DomainID.ValueString()); err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading domain", common.FormatErrorWithAuthGuidance(err, credentialGuidance(verificationResourceName)))
		return
	}

	state.OrganizationID = types.StringValue(organizationID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is reachable only through the timeouts block, which has no effect
// after create, so the planned value is stored as it is.
func (r *verificationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan verificationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete only removes the resource from state. Verification is a record of a
// check Appwrite made, and there is nothing to undo.
func (r *verificationResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}
