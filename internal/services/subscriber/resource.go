package subscriber

import (
	"context"
	"fmt"
	"strings"

	"github.com/appwrite/sdk-for-go/v2/appwrite"
	"github.com/appwrite/sdk-for-go/v2/id"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &subscriberResource{}
	_ resource.ResourceWithConfigure   = &subscriberResource{}
	_ resource.ResourceWithImportState = &subscriberResource{}
)

type subscriberResource struct {
	clients *common.AppwriteClients
}

type subscriberResourceModel struct {
	ID        types.String `tfsdk:"id"`
	ProjectID types.String `tfsdk:"project_id"`
	TopicID   types.String `tfsdk:"topic_id"`
	TargetID  types.String `tfsdk:"target_id"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func NewSubscriberResource() resource.Resource {
	return &subscriberResource{}
}

func (r *subscriberResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_messaging_subscriber"
}

func (r *subscriberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a subscriber to an Appwrite messaging topic.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The subscriber ID.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": common.ProjectIDAttribute(),
			"topic_id": schema.StringAttribute{
				Description:   "The topic ID to subscribe to.",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"target_id": schema.StringAttribute{
				Description:   "The target ID (e.g. a user's email or push target).",
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"created_at": schema.StringAttribute{
				Description: "The subscriber creation timestamp in ISO 8601 format.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The subscriber last update timestamp in ISO 8601 format.",
				Computed:    true,
			},
		},
	}
}

func (r *subscriberResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *subscriberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan subscriberResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, plan.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}
	messagingClient := appwrite.NewMessaging(r.clients.ClientForProject(projectID))

	subscriberID := plan.ID.ValueString()
	if plan.ID.IsNull() || plan.ID.IsUnknown() {
		subscriberID = id.Unique()
	}

	subscriber, err := messagingClient.CreateSubscriber(
		plan.TopicID.ValueString(),
		subscriberID,
		plan.TargetID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error creating subscriber", common.FormatError(err))
		return
	}

	plan.ID = types.StringValue(subscriber.Id)
	plan.CreatedAt = types.StringValue(subscriber.CreatedAt)
	plan.UpdatedAt = types.StringValue(subscriber.UpdatedAt)
	plan.TargetID = types.StringValue(subscriber.TargetId)

	plan.ProjectID = types.StringValue(projectID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *subscriberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state subscriberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}
	messagingClient := appwrite.NewMessaging(r.clients.ClientForProject(projectID))

	subscriber, err := messagingClient.GetSubscriber(state.TopicID.ValueString(), state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading subscriber", common.FormatError(err))
		return
	}

	state.ID = types.StringValue(subscriber.Id)
	state.TargetID = types.StringValue(subscriber.TargetId)
	state.CreatedAt = types.StringValue(subscriber.CreatedAt)
	state.UpdatedAt = types.StringValue(subscriber.UpdatedAt)

	state.ProjectID = types.StringValue(projectID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *subscriberResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Subscribers are immutable. Delete and recreate to change.")
}

func (r *subscriberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state subscriberResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(r.clients, state.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Missing project_id", err.Error())
		return
	}
	messagingClient := appwrite.NewMessaging(r.clients.ClientForProject(projectID))

	_, err = messagingClient.DeleteSubscriber(state.TopicID.ValueString(), state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting subscriber", common.FormatError(err))
	}
}

func (r *subscriberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: topic_id/subscriber_id
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected format: topic_id/subscriber_id")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("topic_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
