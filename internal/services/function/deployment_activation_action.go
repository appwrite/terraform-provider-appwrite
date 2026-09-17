package function

import (
	"context"
	"fmt"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ action.Action              = &deploymentActivationAction{}
	_ action.ActionWithConfigure = &deploymentActivationAction{}
)

type deploymentActivationAction struct {
	clients *common.AppwriteClients
}

type deploymentActivationModel struct {
	FunctionID   types.String `tfsdk:"function_id"`
	DeploymentID types.String `tfsdk:"deployment_id"`
	ProjectID    types.String `tfsdk:"project_id"`
}

func NewDeploymentActivationAction() action.Action {
	return &deploymentActivationAction{}
}

func (a *deploymentActivationAction) Metadata(_ context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_function_deployment_activation"
}

func (a *deploymentActivationAction) Schema(_ context.Context, _ action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Makes an existing deployment the active one for a function. Useful for rolling back to a " +
			"known-good deployment without rebuilding it, which is a decision taken at a moment rather than a " +
			"piece of desired state.",
		Attributes: map[string]schema.Attribute{
			"function_id": schema.StringAttribute{
				Description: "The ID of the function.",
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"deployment_id": schema.StringAttribute{
				Description: "The ID of the deployment to activate.",
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"project_id": schema.StringAttribute{
				Description: "The Appwrite project ID. Defaults to the provider-level project_id.",
				Optional:    true,
			},
		},
	}
}

func (a *deploymentActivationAction) Configure(_ context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	clients, ok := req.ProviderData.(*common.AppwriteClients)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Action Configure Type", fmt.Sprintf("Expected *common.AppwriteClients, got: %T", req.ProviderData))
		return
	}
	a.clients = clients
}

func (a *deploymentActivationAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var config deploymentActivationModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectID, err := common.ResolveProjectID(a.clients, config.ProjectID)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving project ID", err.Error())
		return
	}

	functionsClient := appwrite.NewFunctions(a.clients.ClientForProject(projectID))
	if _, err := functionsClient.UpdateFunctionDeployment(config.FunctionID.ValueString(), config.DeploymentID.ValueString()); err != nil {
		resp.Diagnostics.AddError(
			"Error activating deployment",
			common.FormatErrorWithAuthGuidance(err, common.ProjectCredentialGuidance("appwrite_function_deployment_activation", "functions.write")),
		)
		return
	}

	resp.SendProgress(action.InvokeProgressEvent{
		Message: fmt.Sprintf("Deployment %s is now active for function %s",
			config.DeploymentID.ValueString(), config.FunctionID.ValueString()),
	})
}
