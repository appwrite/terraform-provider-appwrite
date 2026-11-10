package analytics

import (
	"context"
	"fmt"
	"strings"

	"github.com/appwrite/sdk-for-go/v7/analytics"
	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/id"
	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	typeWeb    = "web"
	typeMobile = "mobile"
	typeAgent  = "agent"
)

var (
	_ resource.Resource                = &propertyResource{}
	_ resource.ResourceWithConfigure   = &propertyResource{}
	_ resource.ResourceWithImportState = &propertyResource{}
)

type propertyResource struct {
	clients *common.AppwriteClients
}

type propertyResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Type           types.String `tfsdk:"type"`
	Domain         types.String `tfsdk:"domain"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	AllowedOrigins types.List   `tfsdk:"allowed_origins"`
	IdentifyUsers  types.Bool   `tfsdk:"identify_users"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
	ProjectID      types.String `tfsdk:"project_id"`
}

func NewPropertyResource() resource.Resource {
	return &propertyResource{}
}

func (r *propertyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_analytics_property"
}

func (r *propertyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an analytics property on an Appwrite Cloud project: one website, app or agent " +
			"whose events the project collects.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The property ID, which clients name when they send an event. Changing this " +
					"forces a new resource to be created.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "The property name.",
				Required:    true,
			},
			"type": schema.StringAttribute{
				Description: "The kind of product the property tracks: `web`, `mobile` or `agent`. It only " +
					"changes which panels the console shows; every type accepts the same events. Defaults to `web`.",
				Optional:   true,
				Computed:   true,
				Default:    stringdefault.StaticString(typeWeb),
				Validators: []validator.String{stringvalidator.OneOf(typeWeb, typeMobile, typeAgent)},
			},
			"domain": schema.StringAttribute{
				Description: "The primary domain to track, such as `example.com`. Mobile and agent " +
					"properties can leave it empty. Defaults to empty.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the property accepts events. Events sent to a disabled property are " +
					"rejected as not found. Defaults to true.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"allowed_origins": schema.ListAttribute{
				Description: "The origins allowed to send events. Defaults to `[\"*\"]`, which allows all.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default: listdefault.StaticValue(
					types.ListValueMust(types.StringType, []attr.Value{types.StringValue("*")}),
				),
			},
			"identify_users": schema.BoolAttribute{
				Description: "Whether events sent from a client session record the signed-in user's ID, so " +
					"traffic can be counted by user. Defaults to false.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"created_at": schema.StringAttribute{
				Description: "The property creation timestamp in ISO 8601 format.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description: "The property last update timestamp in ISO 8601 format.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					common.UseStateForUnknownUnlessUpdating(),
				},
			},
			"project_id": common.ProjectIDAttribute(),
		},
	}
}

func (r *propertyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *propertyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan propertyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	client := appwrite.NewAnalytics(r.clients.ClientForProject(projectID))

	propertyID := plan.ID.ValueString()
	if plan.ID.IsNull() || plan.ID.IsUnknown() {
		propertyID = id.Unique()
	}

	var origins []string
	resp.Diagnostics.Append(plan.AllowedOrigins.ElementsAs(ctx, &origins, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	property, err := client.CreateProperty(
		propertyID,
		plan.Name.ValueString(),
		client.WithCreatePropertyType(plan.Type.ValueString()),
		client.WithCreatePropertyDomain(plan.Domain.ValueString()),
		client.WithCreatePropertyEnabled(plan.Enabled.ValueBool()),
		client.WithCreatePropertyAllowedOrigins(origins),
		client.WithCreatePropertyIdentifyUsers(plan.IdentifyUsers.ValueBool()),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error creating analytics property", common.FormatError(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	mapToState(ctx, property, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *propertyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state propertyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	client := appwrite.NewAnalytics(r.clients.ClientForProject(projectID))

	property, err := client.GetProperty(state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading analytics property", common.FormatError(err))
		return
	}

	state.ProjectID = types.StringValue(projectID)
	mapToState(ctx, property, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *propertyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan propertyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	client := appwrite.NewAnalytics(r.clients.ClientForProject(projectID))

	var origins []string
	resp.Diagnostics.Append(plan.AllowedOrigins.ElementsAs(ctx, &origins, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The endpoint changes only what it is sent, so every attribute goes each
	// time: an empty domain then clears one that was set before.
	opts := []analytics.UpdatePropertyOption{
		client.WithUpdatePropertyName(plan.Name.ValueString()),
		client.WithUpdatePropertyType(plan.Type.ValueString()),
		client.WithUpdatePropertyDomain(plan.Domain.ValueString()),
		client.WithUpdatePropertyEnabled(plan.Enabled.ValueBool()),
		client.WithUpdatePropertyAllowedOrigins(origins),
		client.WithUpdatePropertyIdentifyUsers(plan.IdentifyUsers.ValueBool()),
	}

	property, err := client.UpdateProperty(plan.ID.ValueString(), opts...)
	if err != nil {
		resp.Diagnostics.AddError("Error updating analytics property", common.FormatError(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	mapToState(ctx, property, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *propertyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state propertyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	client := appwrite.NewAnalytics(r.clients.ClientForProject(projectID))

	_, err = client.DeleteProperty(state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting analytics property", common.FormatError(err))
	}
}

func (r *propertyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	const expected = "property_id or project_id/property_id"

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
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
	default:
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected %s, got: %s", expected, req.ID))
	}
}

func mapToState(ctx context.Context, property *models.AnalyticsProperty, model *propertyResourceModel, diagnostics *diag.Diagnostics) {
	model.ID = types.StringValue(property.Id)
	model.Name = types.StringValue(property.Name)
	model.Type = types.StringValue(property.Type)
	model.Domain = types.StringValue(property.Domain)
	model.Enabled = types.BoolValue(property.Enabled)
	model.IdentifyUsers = types.BoolValue(property.IdentifyUsers)
	model.CreatedAt = types.StringValue(property.CreatedAt)
	model.UpdatedAt = types.StringValue(property.UpdatedAt)

	origins, diags := types.ListValueFrom(ctx, types.StringType, property.AllowedOrigins)
	diagnostics.Append(diags...)
	model.AllowedOrigins = origins
}
