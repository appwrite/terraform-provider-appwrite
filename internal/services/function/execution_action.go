package function

import (
	"context"
	"fmt"
	"time"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/functions"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/action/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ action.Action              = &executionAction{}
	_ action.ActionWithConfigure = &executionAction{}
)

// executionPollInterval is how often a synchronous invocation checks whether
// the function has finished.
const executionPollInterval = 2 * time.Second

type executionAction struct {
	clients *common.AppwriteClients
}

type executionActionModel struct {
	FunctionID     types.String `tfsdk:"function_id"`
	Path           types.String `tfsdk:"path"`
	Method         types.String `tfsdk:"method"`
	Body           types.String `tfsdk:"body"`
	Async          types.Bool   `tfsdk:"async"`
	TimeoutSeconds types.Int64  `tfsdk:"timeout_seconds"`
	ProjectID      types.String `tfsdk:"project_id"`
}

func NewExecutionAction() action.Action {
	return &executionAction{}
}

func (a *executionAction) Metadata(_ context.Context, req action.MetadataRequest, resp *action.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_function_execution"
}

func (a *executionAction) Schema(_ context.Context, _ action.SchemaRequest, resp *action.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Executes an Appwrite function. Running a function is a thing done at a point in time, " +
			"not a thing that exists, so it is an action rather than a resource: nothing is recorded in state " +
			"and nothing is destroyed later.",
		Attributes: map[string]schema.Attribute{
			"function_id": schema.StringAttribute{
				Description: "The ID of the function to execute.",
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"path": schema.StringAttribute{
				Description: "The request path passed to the function. Defaults to /.",
				Optional:    true,
			},
			"method": schema.StringAttribute{
				Description: "The HTTP method passed to the function. Defaults to POST.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"),
				},
			},
			"body": schema.StringAttribute{
				Description: "The request body passed to the function.",
				Optional:    true,
			},
			"async": schema.BoolAttribute{
				Description: "Queue the execution and return immediately, rather than waiting for it to finish. " +
					"An asynchronous execution that fails will not fail the action, because nothing waits to see it.",
				Optional: true,
			},
			"timeout_seconds": schema.Int64Attribute{
				Description: "How long to wait for a synchronous execution to finish, in seconds. Defaults to 300.",
				Optional:    true,
			},
			"project_id": schema.StringAttribute{
				Description: "The Appwrite project ID. Defaults to the provider-level project_id.",
				Optional:    true,
			},
		},
	}
}

func (a *executionAction) Configure(_ context.Context, req action.ConfigureRequest, resp *action.ConfigureResponse) {
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

func (a *executionAction) Invoke(ctx context.Context, req action.InvokeRequest, resp *action.InvokeResponse) {
	var config executionActionModel
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

	async := !config.Async.IsNull() && config.Async.ValueBool()

	var opts []functions.CreateExecutionOption
	if v := config.Path; !v.IsNull() && !v.IsUnknown() {
		opts = append(opts, functionsClient.WithCreateExecutionPath(v.ValueString()))
	}
	if v := config.Method; !v.IsNull() && !v.IsUnknown() {
		opts = append(opts, functionsClient.WithCreateExecutionMethod(v.ValueString()))
	}
	if v := config.Body; !v.IsNull() && !v.IsUnknown() {
		opts = append(opts, functionsClient.WithCreateExecutionBody(v.ValueString()))
	}
	opts = append(opts, functionsClient.WithCreateExecutionAsync(async))

	resp.SendProgress(action.InvokeProgressEvent{
		Message: fmt.Sprintf("Executing function %s", config.FunctionID.ValueString()),
	})

	execution, err := functionsClient.CreateExecution(config.FunctionID.ValueString(), opts...)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error executing function",
			common.FormatErrorWithAuthGuidance(err, common.ProjectCredentialGuidance("appwrite_function_execution", "executions.write")),
		)
		return
	}

	if async {
		resp.SendProgress(action.InvokeProgressEvent{
			Message: fmt.Sprintf("Queued execution %s", execution.Id),
		})
		return
	}

	timeout := 300 * time.Second
	if v := config.TimeoutSeconds; !v.IsNull() && !v.IsUnknown() {
		timeout = time.Duration(v.ValueInt64()) * time.Second
	}

	// A synchronous execution usually returns complete, but the server may
	// still report it as processing; poll until it settles so the action
	// reports the outcome the caller actually cares about.
	deadline := time.Now().Add(timeout)
	for execution.Status == "waiting" || execution.Status == "processing" {
		if time.Now().After(deadline) {
			resp.Diagnostics.AddError(
				"Timed out waiting for function execution",
				fmt.Sprintf("Execution %s was still %q after %s. Raise timeout_seconds, or set async to queue it without waiting.", execution.Id, execution.Status, timeout),
			)
			return
		}
		select {
		case <-ctx.Done():
			resp.Diagnostics.AddError("Cancelled waiting for function execution", ctx.Err().Error())
			return
		case <-time.After(executionPollInterval):
		}

		execution, err = functionsClient.GetExecution(config.FunctionID.ValueString(), execution.Id)
		if err != nil {
			resp.Diagnostics.AddError("Error reading function execution", common.FormatError(err))
			return
		}
	}

	if execution.Status == "failed" {
		resp.Diagnostics.AddError(
			"Function execution failed",
			fmt.Sprintf("Execution %s finished with status %q and response code %d.\n\nLogs:\n%s\n\nErrors:\n%s",
				execution.Id, execution.Status, execution.ResponseStatusCode, execution.Logs, execution.Errors),
		)
		return
	}

	resp.SendProgress(action.InvokeProgressEvent{
		Message: fmt.Sprintf("Execution %s completed with response code %d", execution.Id, execution.ResponseStatusCode),
	})
}
