package escalation

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/id"
	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/appwrite/sdk-for-go/v7/organization"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	resourceName = "appwrite_escalation"

	maxIDLength   = 36
	maxNameLength = 128
	maxDelay      = 2592000
	maxRepeat     = 9
	maxSteps      = 100
)

var (
	scopes = []string{"escalations.read", "escalations.write"}

	escalationIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

	stepType = types.ObjectType{AttrTypes: map[string]attr.Type{
		"delay":    types.Int64Type,
		"roles":    types.ListType{ElemType: types.StringType},
		"user_ids": types.ListType{ElemType: types.StringType},
	}}
)

var (
	_ resource.Resource                = &escalationResource{}
	_ resource.ResourceWithConfigure   = &escalationResource{}
	_ resource.ResourceWithImportState = &escalationResource{}
)

type escalationResource struct {
	clients *common.AppwriteClients
}

type escalationResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Steps          types.List   `tfsdk:"steps"`
	Repeat         types.Int64  `tfsdk:"repeat"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
	OrganizationID types.String `tfsdk:"organization_id"`
}

type stepModel struct {
	Delay   types.Int64 `tfsdk:"delay"`
	Roles   types.List  `tfsdk:"roles"`
	UserIDs types.List  `tfsdk:"user_ids"`
}

func NewEscalationResource() resource.Resource {
	return &escalationResource{}
}

func (r *escalationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_escalation"
}

func (r *escalationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	emptyList := listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{}))
	resp.Schema = schema.Schema{
		Description: "Manages an escalation of an Appwrite organization: who a monitor's firing alert notifies, " +
			"step by step, until a member acknowledges it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The escalation ID. Generated when not set. " + common.ForcesReplacementNote,
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, maxIDLength),
					stringvalidator.RegexMatches(escalationIDPattern,
						"must contain only letters, digits, periods, hyphens and underscores, and must not start with a special character"),
				},
			},
			"name": schema.StringAttribute{
				Description: fmt.Sprintf("The escalation name, up to %d characters.", maxNameLength),
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthBetween(1, maxNameLength)},
			},
			"steps": schema.ListNestedAttribute{
				Description: fmt.Sprintf("The steps notified in order while nobody acknowledges the alert, up to "+
					"%d. Each needs at least one of roles or user_ids.", maxSteps),
				Required:   true,
				Validators: []validator.List{listvalidator.SizeBetween(1, maxSteps)},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"delay": schema.Int64Attribute{
							Description: fmt.Sprintf("Seconds to wait before notifying this step, from 0 to %d: "+
								"after the alert fires for the first step, after the previous step for the "+
								"others. Defaults to 0.", maxDelay),
							Optional:   true,
							Computed:   true,
							Default:    int64default.StaticInt64(0),
							Validators: []validator.Int64{int64validator.Between(0, maxDelay)},
						},
						"roles": schema.ListAttribute{
							Description: "Organization roles, such as owner or developer, whose confirmed " +
								"members this step notifies. Defaults to none.",
							Optional:    true,
							Computed:    true,
							ElementType: types.StringType,
							Default:     emptyList,
							Validators:  []validator.List{listvalidator.UniqueValues()},
						},
						"user_ids": schema.ListAttribute{
							Description: "IDs of organization members who accepted their invitation, notified " +
								"by this step whatever their role. Defaults to none.",
							Optional:    true,
							Computed:    true,
							ElementType: types.StringType,
							Default:     emptyList,
							Validators:  []validator.List{listvalidator.UniqueValues()},
						},
					},
				},
			},
			"repeat": schema.Int64Attribute{
				Description: fmt.Sprintf("How many more times the steps run from the first while nobody "+
					"acknowledges the alert, from 0 to %d. Each round starts a monitor's cooldown after the last "+
					"step. Defaults to 0.", maxRepeat),
				Optional:   true,
				Computed:   true,
				Default:    int64default.StaticInt64(0),
				Validators: []validator.Int64{int64validator.Between(0, maxRepeat)},
			},
			"created_at": schema.StringAttribute{
				Description:   "The escalation creation timestamp in ISO 8601 format.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Description:   "The escalation last update timestamp in ISO 8601 format.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{common.UseStateForUnknownUnlessUpdating()},
			},
			"organization_id": common.OrganizationIDAttribute(),
		},
	}
}

func (r *escalationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *escalationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan escalationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	organizationID, client, ok := r.client(plan.OrganizationID, &resp.Diagnostics)
	if !ok {
		return
	}

	escalationID := plan.ID.ValueString()
	if plan.ID.IsNull() || plan.ID.IsUnknown() {
		escalationID = id.Unique()
	}
	steps := stepsFromModel(ctx, plan.Steps, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	escalation, err := client.CreateEscalation(escalationID, plan.Name.ValueString(), steps,
		client.WithCreateEscalationRepeat(int(plan.Repeat.ValueInt64())))
	if err != nil {
		resp.Diagnostics.AddError("Error creating escalation", common.FormatErrorWithAuthGuidance(err, common.OrganizationCredentialGuidance(resourceName, scopes...)))
		return
	}

	plan.OrganizationID = types.StringValue(organizationID)
	resp.Diagnostics.Append(mapToState(ctx, escalation, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *escalationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state escalationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	organizationID, client, ok := r.client(state.OrganizationID, &resp.Diagnostics)
	if !ok {
		return
	}

	escalation, err := client.GetEscalation(state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading escalation", common.FormatErrorWithAuthGuidance(err, common.OrganizationCredentialGuidance(resourceName, scopes...)))
		return
	}

	state.OrganizationID = types.StringValue(organizationID)
	resp.Diagnostics.Append(mapToState(ctx, escalation, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *escalationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state escalationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	organizationID, client, ok := r.client(plan.OrganizationID, &resp.Diagnostics)
	if !ok {
		return
	}

	// The route changes only what it is sent.
	var opts []organization.UpdateEscalationOption
	if !plan.Name.Equal(state.Name) {
		opts = append(opts, client.WithUpdateEscalationName(plan.Name.ValueString()))
	}
	if !plan.Steps.Equal(state.Steps) {
		steps := stepsFromModel(ctx, plan.Steps, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		opts = append(opts, client.WithUpdateEscalationSteps(steps))
	}
	if !plan.Repeat.Equal(state.Repeat) {
		opts = append(opts, client.WithUpdateEscalationRepeat(int(plan.Repeat.ValueInt64())))
	}

	escalation, err := client.UpdateEscalation(plan.ID.ValueString(), opts...)
	if err != nil {
		resp.Diagnostics.AddError("Error updating escalation", common.FormatErrorWithAuthGuidance(err, common.OrganizationCredentialGuidance(resourceName, scopes...)))
		return
	}

	plan.OrganizationID = types.StringValue(organizationID)
	resp.Diagnostics.Append(mapToState(ctx, escalation, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *escalationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state escalationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, client, ok := r.client(state.OrganizationID, &resp.Diagnostics)
	if !ok {
		return
	}

	_, err := client.DeleteEscalation(state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting escalation", common.FormatErrorWithAuthGuidance(err, common.OrganizationCredentialGuidance(resourceName, scopes...)))
	}
}

func (r *escalationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	const expected = "escalation_id or organization_id/escalation_id"

	parts := strings.Split(req.ID, "/")
	for _, part := range parts {
		if part == "" {
			resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected %s, got: %s", expected, req.ID))
			return
		}
	}
	switch len(parts) {
	case 1:
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[0])...)
	case 2:
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
	default:
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected %s, got: %s", expected, req.ID))
	}
}

func (r *escalationResource) client(organizationID types.String, diags *diag.Diagnostics) (string, *organization.Organization, bool) {
	resolved, err := common.ResolveOrganizationID(r.clients, organizationID)
	if err != nil {
		diags.AddError("Error resolving organization ID", err.Error())
		return "", nil, false
	}
	if err := common.ValidateOrganizationCredential(r.clients, resourceName, scopes...); err != nil {
		diags.AddError("Invalid credential", err.Error())
		return "", nil, false
	}
	return resolved, appwrite.NewOrganization(r.clients.ClientForOrganization(resolved)), true
}

// stepsFromModel builds the steps argument, one object per step in the shape
// the API documents.
func stepsFromModel(ctx context.Context, list types.List, diags *diag.Diagnostics) []interface{} {
	var steps []stepModel
	diags.Append(list.ElementsAs(ctx, &steps, false)...)
	result := make([]interface{}, 0, len(steps))
	for _, step := range steps {
		roles := []string{}
		userIDs := []string{}
		diags.Append(step.Roles.ElementsAs(ctx, &roles, false)...)
		diags.Append(step.UserIDs.ElementsAs(ctx, &userIDs, false)...)
		result = append(result, map[string]interface{}{
			"delay":   step.Delay.ValueInt64(),
			"roles":   roles,
			"userIds": userIDs,
		})
	}
	return result
}

func mapToState(ctx context.Context, escalation *models.Escalation, model *escalationResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	model.ID = types.StringValue(escalation.Id)
	model.Name = types.StringValue(escalation.Name)
	model.Repeat = types.Int64Value(int64(escalation.Repeat))
	model.CreatedAt = types.StringValue(escalation.CreatedAt)
	model.UpdatedAt = types.StringValue(escalation.UpdatedAt)

	steps := make([]stepModel, 0, len(escalation.Steps))
	for _, step := range escalation.Steps {
		roles, d := types.ListValueFrom(ctx, types.StringType, nonNil(step.Roles))
		diags.Append(d...)
		userIDs, d := types.ListValueFrom(ctx, types.StringType, nonNil(step.UserIds))
		diags.Append(d...)
		steps = append(steps, stepModel{Delay: types.Int64Value(int64(step.Delay)), Roles: roles, UserIDs: userIDs})
	}
	list, d := types.ListValueFrom(ctx, stepType, steps)
	diags.Append(d...)
	model.Steps = list
	return diags
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
