package site

import (
	"context"
	"fmt"
	"strings"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/id"
	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/appwrite/sdk-for-go/v7/sites"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &variableResource{}
	_ resource.ResourceWithConfigure   = &variableResource{}
	_ resource.ResourceWithImportState = &variableResource{}
)

type variableResource struct {
	clients *common.AppwriteClients
}

type variableResourceModel struct {
	ID             types.String `tfsdk:"id"`
	SiteID         types.String `tfsdk:"site_id"`
	Key            types.String `tfsdk:"key"`
	Value          types.String `tfsdk:"value"`
	ValueWO        types.String `tfsdk:"value_wo"`
	ValueWOVersion types.Int64  `tfsdk:"value_wo_version"`
	Secret         types.Bool   `tfsdk:"secret"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
	ProjectID      types.String `tfsdk:"project_id"`
}

func NewVariableResource() resource.Resource {
	return &variableResource{}
}

func (r *variableResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site_variable"
}

func (r *variableResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Appwrite site environment variable.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The variable ID.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"site_id": schema.StringAttribute{
				Description:   "The site ID this variable belongs to.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"key": schema.StringAttribute{
				Description: "The variable key (name).",
				Required:    true,
				Validators:  common.VariableKeyValidators(),
			},
			"value": schema.StringAttribute{
				Description: "The variable value. Stored in Terraform state; prefer value_wo, which is not. " +
					"Exactly one of value or value_wo must be set.",
				Optional:  true,
				Sensitive: true,
			},
			"value_wo": schema.StringAttribute{
				Description: "The variable value, as a write-only argument. Read from the configuration during " +
					"apply and never persisted. Change value_wo_version to apply a new value, since Terraform " +
					"cannot detect a change in a value it does not store. Requires Terraform 1.11 or later.",
				Optional:  true,
				Sensitive: true,
				WriteOnly: true,
				// The version is what makes a changed secret visible to
				// Terraform. Without it, switching an existing resource from
				// the stored attribute to this one leaves both versions null,
				// nothing compares unequal, and the apply reports success while
				// the old secret stays in place.
				Validators: []validator.String{
					stringvalidator.AlsoRequires(path.MatchRoot("value_wo_version")),
				},
			},
			"value_wo_version": schema.Int64Attribute{
				Description: "Increment to apply a changed value_wo.",
				Optional:    true,
			},
			"secret": schema.BoolAttribute{
				Description: "Whether the variable is secret. Secret variables can only be updated or deleted, never read.",
				Optional:    true,
			},
			"created_at": schema.StringAttribute{
				Description:   "The variable creation timestamp in ISO 8601 format.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Description: "The variable last update timestamp in ISO 8601 format.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					common.UseStateForUnknownUnlessUpdating(),
				},
			},
			"project_id": common.ProjectIDAttribute(),
		},
	}
}

// ConfigValidators enforces that a value arrives exactly one way. Making
// `value` optional to admit `value_wo` would otherwise allow a variable with no
// value at all, which the API rejects with a less obvious message.
func (r *variableResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.ExactlyOneOf(
			path.MatchRoot("value"),
			path.MatchRoot("value_wo"),
		),
	}
}

func (r *variableResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *variableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan variableResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	sitesClient := appwrite.NewSites(r.clients.ClientForProject(projectID))

	var createOpts []sites.CreateVariableOption
	if !plan.Secret.IsNull() {
		createOpts = append(createOpts, sitesClient.WithCreateVariableSecret(plan.Secret.ValueBool()))
	}

	valueWO := common.WriteOnlyValue(ctx, req.Config, "value_wo", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	value := common.ResolveSecret(valueWO, plan.Value)

	variable, err := sitesClient.CreateVariable(
		plan.SiteID.ValueString(),
		id.Unique(),
		plan.Key.ValueString(),
		value,
		createOpts...,
	)
	if err != nil {
		resp.Diagnostics.AddError("Error creating site variable", common.FormatError(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	r.mapToState(variable, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *variableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state variableResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	sitesClient := appwrite.NewSites(r.clients.ClientForProject(projectID))

	variable, err := sitesClient.GetVariable(state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading site variable", common.FormatError(err))
		return
	}

	state.ProjectID = types.StringValue(projectID)
	r.mapToState(variable, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *variableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan variableResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	sitesClient := appwrite.NewSites(r.clients.ClientForProject(projectID))

	valueWO := common.WriteOnlyValue(ctx, req.Config, "value_wo", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	updateValue := common.ResolveSecret(valueWO, plan.Value)

	updateOpts := []sites.UpdateVariableOption{
		sitesClient.WithUpdateVariableKey(plan.Key.ValueString()),
		sitesClient.WithUpdateVariableValue(updateValue),
	}
	if !plan.Secret.IsNull() {
		updateOpts = append(updateOpts, sitesClient.WithUpdateVariableSecret(plan.Secret.ValueBool()))
	}

	variable, err := sitesClient.UpdateVariable(
		plan.SiteID.ValueString(),
		plan.ID.ValueString(),
		updateOpts...,
	)
	if err != nil {
		resp.Diagnostics.AddError("Error updating site variable", common.FormatError(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	r.mapToState(variable, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *variableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state variableResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	sitesClient := appwrite.NewSites(r.clients.ClientForProject(projectID))

	_, err = sitesClient.DeleteVariable(state.SiteID.ValueString(), state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting site variable", common.FormatError(err))
	}
}

func (r *variableResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: site_id/variable_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("site_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func (r *variableResource) mapToState(variable *models.Variable, model *variableResourceModel) {
	model.ID = types.StringValue(variable.Id)
	model.Key = types.StringValue(variable.Key)
	model.CreatedAt = types.StringValue(variable.CreatedAt)
	model.UpdatedAt = types.StringValue(variable.UpdatedAt)
	// Only refresh `value` when the configuration owns it. A non-secret
	// variable has its value returned by the API on both create and read, so
	// copying it unconditionally would write a value_wo secret into state and
	// break the guarantee the write-only argument exists to make. A secret
	// variable comes back empty, so that path was never the risk.
	if variable.Value != "" && !model.Value.IsNull() {
		model.Value = types.StringValue(variable.Value)
	}
}
