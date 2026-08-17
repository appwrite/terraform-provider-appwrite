package user

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &userDataSource{}
	_ datasource.DataSourceWithConfigure = &userDataSource{}
)

type userDataSource struct {
	clients *common.AppwriteClients
}

type userDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Email             types.String `tfsdk:"email"`
	Phone             types.String `tfsdk:"phone"`
	Status            types.Bool   `tfsdk:"status"`
	Labels            types.List   `tfsdk:"labels"`
	EmailVerification types.Bool   `tfsdk:"email_verification"`
	PhoneVerification types.Bool   `tfsdk:"phone_verification"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
	ProjectID         types.String `tfsdk:"project_id"`
}

func NewUserDataSource() datasource.DataSource {
	return &userDataSource{}
}

func (d *userDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_auth_user"
}

func (d *userDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches an Appwrite user by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The user ID.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "The user name.",
				Computed:    true,
			},
			"email": schema.StringAttribute{
				Description: "The user email address.",
				Computed:    true,
			},
			"phone": schema.StringAttribute{
				Description: "The user phone number.",
				Computed:    true,
			},
			"status": schema.BoolAttribute{
				Description: "Whether the user account is enabled.",
				Computed:    true,
			},
			"labels": schema.ListAttribute{
				Description: "User labels.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"email_verification": schema.BoolAttribute{
				Description: "Whether the user email is verified.",
				Computed:    true,
			},
			"phone_verification": schema.BoolAttribute{
				Description: "Whether the user phone is verified.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "The user creation timestamp.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "The user last update timestamp.",
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "The Appwrite project ID. Defaults to the provider-level project_id.",
				Optional:    true,
				Computed:    true,
			},
		},
	}
}

func (d *userDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	clients, ok := req.ProviderData.(*common.AppwriteClients)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *common.AppwriteClients, got: %T", req.ProviderData),
		)
		return
	}
	d.clients = clients
}

func (d *userDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config userDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(d.clients, config.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}
	usersClient := appwrite.NewUsers(d.clients.ClientForProject(projectID))

	user, err := usersClient.Get(config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading user", common.FormatError(err))
		return
	}

	config.ProjectID = types.StringValue(projectID)
	config.ID = types.StringValue(user.Id)
	config.Status = types.BoolValue(user.Status)
	config.EmailVerification = types.BoolValue(user.EmailVerification)
	config.PhoneVerification = types.BoolValue(user.PhoneVerification)
	config.CreatedAt = types.StringValue(user.CreatedAt)
	config.UpdatedAt = types.StringValue(user.UpdatedAt)

	if user.Name != "" {
		config.Name = types.StringValue(user.Name)
	}
	if user.Email != "" {
		config.Email = types.StringValue(user.Email)
	}
	if user.Phone != "" {
		config.Phone = types.StringValue(user.Phone)
	}

	labelsList, diags := types.ListValueFrom(ctx, types.StringType, user.Labels)
	resp.Diagnostics.Append(diags...)
	config.Labels = labelsList

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
