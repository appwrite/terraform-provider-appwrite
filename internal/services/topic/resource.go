package topic

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v2/appwrite"
	"github.com/appwrite/sdk-for-go/v2/id"
	"github.com/appwrite/sdk-for-go/v2/messaging"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &topicResource{}
	_ resource.ResourceWithConfigure   = &topicResource{}
	_ resource.ResourceWithImportState = &topicResource{}
)

type topicResource struct {
	clients *common.AppwriteClients
}

type topicResourceModel struct {
	ProjectID types.String `tfsdk:"project_id"`
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Subscribe types.List   `tfsdk:"subscribe"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func NewTopicResource() resource.Resource {
	return &topicResource{}
}

func (r *topicResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_messaging_topic"
}

func (r *topicResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an Appwrite messaging topic.",
		Attributes: map[string]schema.Attribute{
			"project_id": common.ProjectIDAttribute(),
			"id": schema.StringAttribute{
				Description:   "The topic ID.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Description: "The topic name.",
				Required:    true,
			},
			"subscribe": schema.ListAttribute{
				Description: "Subscribe permissions.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
			},
			"created_at": schema.StringAttribute{
				Description: "The topic creation timestamp in ISO 8601 format.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The topic last update timestamp in ISO 8601 format.",
				Computed:    true,
			},
		},
	}
}

func (r *topicResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *topicResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan topicResourceModel
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

	var opts []messaging.CreateTopicOption
	if !plan.Subscribe.IsNull() && !plan.Subscribe.IsUnknown() {
		var subscribe []string
		resp.Diagnostics.Append(plan.Subscribe.ElementsAs(ctx, &subscribe, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		opts = append(opts, messagingClient.WithCreateTopicSubscribe(subscribe))
	}

	topicID := plan.ID.ValueString()
	if plan.ID.IsNull() || plan.ID.IsUnknown() {
		topicID = id.Unique()
	}

	topic, err := messagingClient.CreateTopic(topicID, plan.Name.ValueString(), opts...)
	if err != nil {
		resp.Diagnostics.AddError("Error creating topic", common.FormatError(err))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	plan.ID = types.StringValue(topic.Id)
	plan.Name = types.StringValue(topic.Name)
	plan.CreatedAt = types.StringValue(topic.CreatedAt)
	plan.UpdatedAt = types.StringValue(topic.UpdatedAt)
	subList, diags := types.ListValueFrom(ctx, types.StringType, topic.Subscribe)
	resp.Diagnostics.Append(diags...)
	plan.Subscribe = subList

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *topicResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state topicResourceModel
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

	topic, err := messagingClient.GetTopic(state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading topic", common.FormatError(err))
		return
	}

	state.ProjectID = types.StringValue(projectID)
	state.ID = types.StringValue(topic.Id)
	state.Name = types.StringValue(topic.Name)
	state.CreatedAt = types.StringValue(topic.CreatedAt)
	state.UpdatedAt = types.StringValue(topic.UpdatedAt)
	subList, diags := types.ListValueFrom(ctx, types.StringType, topic.Subscribe)
	resp.Diagnostics.Append(diags...)
	state.Subscribe = subList

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *topicResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan topicResourceModel
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

	var opts []messaging.UpdateTopicOption
	opts = append(opts, messagingClient.WithUpdateTopicName(plan.Name.ValueString()))
	if !plan.Subscribe.IsNull() && !plan.Subscribe.IsUnknown() {
		var subscribe []string
		resp.Diagnostics.Append(plan.Subscribe.ElementsAs(ctx, &subscribe, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		opts = append(opts, messagingClient.WithUpdateTopicSubscribe(subscribe))
	}

	topic, err := messagingClient.UpdateTopic(plan.ID.ValueString(), opts...)
	if err != nil {
		resp.Diagnostics.AddError("Error updating topic", common.FormatError(err))
		return
	}

	plan.ID = types.StringValue(topic.Id)
	plan.Name = types.StringValue(topic.Name)
	plan.CreatedAt = types.StringValue(topic.CreatedAt)
	plan.UpdatedAt = types.StringValue(topic.UpdatedAt)
	subList, diags := types.ListValueFrom(ctx, types.StringType, topic.Subscribe)
	resp.Diagnostics.Append(diags...)
	plan.Subscribe = subList

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *topicResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state topicResourceModel
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

	_, err = messagingClient.DeleteTopic(state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting topic", common.FormatError(err))
	}
}

func (r *topicResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
