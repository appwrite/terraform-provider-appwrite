package project

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/models"
	sdkproject "github.com/appwrite/sdk-for-go/v7/project"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
)

const mockPhoneResourceName = "appwrite_project_mock_phone"

var mockPhoneScopes = []string{"mocks.read", "mocks.write"}

var (
	// The E.164 form the create route accepts. Appwrite also checks that the
	// number starts with a calling code it knows.
	mockPhoneNumberPattern = regexp.MustCompile(`^\+[1-9][0-9]{6,14}$`)
	mockPhoneOTPPattern    = regexp.MustCompile(`^[0-9]{6}$`)
)

var (
	_ resource.Resource                = &mockPhoneResource{}
	_ resource.ResourceWithConfigure   = &mockPhoneResource{}
	_ resource.ResourceWithImportState = &mockPhoneResource{}
)

type mockPhoneResource struct {
	clients *common.AppwriteClients
}

type mockPhoneResourceModel struct {
	ID        types.String `tfsdk:"id"`
	ProjectID types.String `tfsdk:"project_id"`
	Number    types.String `tfsdk:"number"`
	OTP       types.String `tfsdk:"otp"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func NewMockPhoneResource() resource.Resource {
	return &mockPhoneResource{}
}

func (r *mockPhoneResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_mock_phone"
}

func (r *mockPhoneResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Registers a mock phone number in a project. Signing in with a phone token for the number " +
			"sends no SMS and accepts the fixed code set here, which lets test and app-review accounts sign in " +
			"without a real phone.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The phone number, the same as `number`.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": common.ProjectIDAttribute(),
			"number": schema.StringAttribute{
				Description: "The phone number in E.164 format, such as `+15555550100`. " + common.ForcesReplacementNote,
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(mockPhoneNumberPattern, "must be an E.164 phone number: a + followed by 7 to 15 digits, not starting with 0"),
				},
			},
			"otp": schema.StringAttribute{
				Description: "The six-digit code that signs in with this number.",
				Required:    true,
				Sensitive:   true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(mockPhoneOTPPattern, "must be exactly six digits"),
				},
			},
			"created_at": schema.StringAttribute{
				Description:   "When the mock number was added.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Description:   "When the mock number's code was last changed.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{common.UseStateForUnknownUnlessUpdating()},
			},
		},
	}
}

func (r *mockPhoneResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *mockPhoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mockPhoneResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectID, client, ok := r.client(plan.ProjectID, &resp.Diagnostics)
	if !ok {
		return
	}

	unlock := lockAuthSettings(projectID)
	created, err := client.CreateMockPhone(plan.Number.ValueString(), plan.OTP.ValueString())
	unlock()
	if err != nil {
		if common.ErrorType(err) == "mock_number_already_exists" {
			resp.Diagnostics.AddError("Mock phone number already exists",
				fmt.Sprintf("%s is already a mock number in project %s. Import it to manage it here: "+
					"terraform import <address> %s/%s", plan.Number.ValueString(), projectID, projectID, plan.Number.ValueString()))
			return
		}
		resp.Diagnostics.AddError("Error creating mock phone number", common.FormatErrorWithAuthGuidance(err, mockPhoneGuidance()))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	mapMockPhoneToState(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *mockPhoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state mockPhoneResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectID, client, ok := r.client(state.ProjectID, &resp.Diagnostics)
	if !ok {
		return
	}

	found, err := client.GetMockPhone(state.Number.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading mock phone number", common.FormatErrorWithAuthGuidance(err, mockPhoneGuidance()))
		return
	}

	state.ProjectID = types.StringValue(projectID)
	mapMockPhoneToState(found, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *mockPhoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan mockPhoneResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectID, client, ok := r.client(plan.ProjectID, &resp.Diagnostics)
	if !ok {
		return
	}

	unlock := lockAuthSettings(projectID)
	updated, err := client.UpdateMockPhone(plan.Number.ValueString(), plan.OTP.ValueString())
	unlock()
	if err != nil {
		resp.Diagnostics.AddError("Error updating mock phone number", common.FormatErrorWithAuthGuidance(err, mockPhoneGuidance()))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	mapMockPhoneToState(updated, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *mockPhoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mockPhoneResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectID, client, ok := r.client(state.ProjectID, &resp.Diagnostics)
	if !ok {
		return
	}

	unlock := lockAuthSettings(projectID)
	_, err := client.DeleteMockPhone(state.Number.ValueString())
	unlock()
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting mock phone number", common.FormatErrorWithAuthGuidance(err, mockPhoneGuidance()))
	}
}

func (r *mockPhoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	const expected = "Expected number or project_id/number"
	parts := strings.Split(req.ID, "/")
	var number string
	switch {
	case len(parts) == 1 && parts[0] != "":
		number = parts[0]
	case len(parts) == 2 && parts[0] != "" && parts[1] != "":
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
		number = parts[1]
	default:
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("%s, got: %s", expected, req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("number"), number)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), number)...)
}

func (r *mockPhoneResource) client(configured types.String, diags *diag.Diagnostics) (string, *sdkproject.Project, bool) {
	projectID, err := common.ResolveProjectID(r.clients, configured)
	if err != nil {
		diags.AddError("Missing project_id", err.Error())
		return "", nil, false
	}
	if err := common.ValidateProjectCredential(r.clients, mockPhoneResourceName, mockPhoneScopes...); err != nil {
		diags.AddError("Unsupported Appwrite credential", err.Error())
		return "", nil, false
	}
	return projectID, appwrite.NewProject(r.clients.ClientForProject(projectID)), true
}

func mockPhoneGuidance() string {
	return common.ProjectCredentialGuidance(mockPhoneResourceName, mockPhoneScopes...)
}

func mapMockPhoneToState(mock *models.MockNumber, model *mockPhoneResourceModel) {
	model.ID = types.StringValue(mock.Number)
	model.Number = types.StringValue(mock.Number)
	model.OTP = types.StringValue(mock.Otp)
	model.CreatedAt = types.StringValue(mock.CreatedAt)
	model.UpdatedAt = types.StringValue(mock.UpdatedAt)
}
