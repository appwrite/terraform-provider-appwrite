package monitor

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/id"
	"github.com/appwrite/sdk-for-go/v7/monitors"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	typeHTTP  = "http"
	typeDNS   = "dns"
	typeTCP   = "tcp"
	typeSMTP  = "smtp"
	typeUsage = "usage"

	resourceName = "appwrite_monitor"

	maxIDLength   = 36
	maxNameLength = 128
	maxBodyLength = 16384
	maxSeconds    = 2592000
	maxTimeout    = 60
)

var (
	monitorTypes = []string{typeHTTP, typeDNS, typeTCP, typeSMTP, typeUsage}
	probeTypes   = []string{typeHTTP, typeDNS, typeTCP, typeSMTP}
	severities   = []string{"info", "warning", "error", "critical"}
	methods      = []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	records      = []string{"A", "AAAA", "CNAME", "MX", "TXT", "NS"}
	tlsModes     = []string{"auto", "none", "starttls", "implicit"}
	scopes       = []string{"monitors.read", "monitors.write"}

	monitorIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
)

var (
	_ resource.Resource                   = &monitorResource{}
	_ resource.ResourceWithConfigure      = &monitorResource{}
	_ resource.ResourceWithImportState    = &monitorResource{}
	_ resource.ResourceWithValidateConfig = &monitorResource{}
)

type monitorResource struct {
	clients *common.AppwriteClients
}

type monitorResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Type         types.String `tfsdk:"type"`
	Name         types.String `tfsdk:"name"`
	Rules        types.List   `tfsdk:"rules"`
	Interval     types.Int64  `tfsdk:"interval"`
	Timeout      types.Int64  `tfsdk:"timeout"`
	Cooldown     types.Int64  `tfsdk:"cooldown"`
	Enabled      types.Bool   `tfsdk:"enabled"`
	Notify       types.List   `tfsdk:"notify"`
	Severity     types.String `tfsdk:"severity"`
	EscalationID types.String `tfsdk:"escalation_id"`
	URL          types.String `tfsdk:"url"`
	Method       types.String `tfsdk:"method"`
	Headers      types.Map    `tfsdk:"headers"`
	Body         types.String `tfsdk:"body"`
	Domain       types.String `tfsdk:"domain"`
	Record       types.String `tfsdk:"record"`
	Resolver     types.String `tfsdk:"resolver"`
	Host         types.String `tfsdk:"host"`
	Port         types.Int64  `tfsdk:"port"`
	TLSMode      types.String `tfsdk:"tls_mode"`
	Window       types.Int64  `tfsdk:"window"`
	Status       types.String `tfsdk:"status"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
	ProjectID    types.String `tfsdk:"project_id"`
}

func NewMonitorResource() resource.Resource {
	return &monitorResource{}
}

func (r *monitorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitor"
}

func (r *monitorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a monitor on an Appwrite project: an HTTP, DNS, TCP or SMTP probe of a target, or " +
			"a check of the project's usage, that opens an alert when its rules match.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The monitor ID. Generated when not set. " + common.ForcesReplacementNote,
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, maxIDLength),
					stringvalidator.RegexMatches(monitorIDPattern,
						"must contain only letters, digits, periods, hyphens and underscores, and must not start with a special character"),
				},
			},
			"type": schema.StringAttribute{
				Description: "What the monitor checks: `http`, `dns`, `tcp` and `smtp` probe a target on their own " +
					"interval, and `usage` compares a usage metric with a threshold. " + common.ForcesReplacementNote,
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.OneOf(monitorTypes...)},
			},
			"name": schema.StringAttribute{
				Description: fmt.Sprintf("The monitor name, up to %d characters.", maxNameLength),
				Required:    true,
				Validators:  []validator.String{stringvalidator.LengthBetween(1, maxNameLength)},
			},
			"rules": schema.ListAttribute{
				Description: "Rules that must all hold for the monitor to fire, each a query string usually written " +
					"with jsonencode, such as jsonencode({ method = \"greaterThan\", attribute = \"latency\", " +
					"values = [500] }). A usage monitor needs one comparing a usage metric with a threshold, and " +
					"may add equal rules on resourceType and resourceId to watch one database, bucket, function or " +
					"site. Probe monitors compare variables of the check: statusCode, latency, certificateDays and " +
					"certificateLifetime for http; answer, answers and latency for dns; latency for tcp; code, " +
					"latency, certificateDays and certificateLifetime for smtp. Required for usage monitors. Left " +
					"out, a probe monitor uses its type's default rules, which are not recorded in state.",
				Optional:    true,
				ElementType: types.StringType,
				Validators:  []validator.List{listvalidator.SizeAtLeast(1)},
			},
			"interval": schema.Int64Attribute{
				Description: fmt.Sprintf("Seconds between two checks of a probe monitor, from 1 to %d. Defaults to "+
					"60, or to the shortest interval the organization's plan allows when that is longer; left out, "+
					"an existing monitor keeps its interval. Not accepted for usage monitors.", maxSeconds),
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{intervalDefault{}},
				Validators:    []validator.Int64{int64validator.Between(1, maxSeconds)},
			},
			"timeout": schema.Int64Attribute{
				Description: fmt.Sprintf("Seconds a probe monitor's check may take before it fails, from 1 to %d, "+
					"and no longer than interval. Defaults to 10 for probe monitors; not accepted for usage "+
					"monitors.", maxTimeout),
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{defaultInt64For(10, probeTypes...)},
				Validators:    []validator.Int64{int64validator.Between(1, maxTimeout)},
			},
			"cooldown": schema.Int64Attribute{
				Description: fmt.Sprintf("Minimum seconds between two team notifications of the monitor firing, "+
					"from 300 to %d. Alert events fire on every change regardless. Defaults to 3600.", maxSeconds),
				Optional:   true,
				Computed:   true,
				Default:    int64default.StaticInt64(3600),
				Validators: []validator.Int64{int64validator.Between(300, maxSeconds)},
			},
			"enabled": schema.BoolAttribute{
				Description: "Whether the monitor is evaluated. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"notify": schema.ListAttribute{
				Description: "Organization roles, such as owner or developer, whose confirmed members get an email " +
					"and a console notification when the monitor fires and resolves. An empty list leaves the " +
					"alert events as the only delivery. Not used while escalation_id is set. Defaults to " +
					"`[\"owner\"]`.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Validators:  []validator.List{listvalidator.UniqueValues()},
				Default: listdefault.StaticValue(
					types.ListValueMust(types.StringType, []attr.Value{types.StringValue("owner")}),
				),
			},
			"severity": schema.StringAttribute{
				Description: "How urgent the monitor's alerts are: " + strings.Join(severities, ", ") +
					". Defaults to warning.",
				Optional:   true,
				Computed:   true,
				Default:    stringdefault.StaticString("warning"),
				Validators: []validator.String{stringvalidator.OneOf(severities...)},
			},
			"escalation_id": schema.StringAttribute{
				Description: "The ID of an escalation of the organization, such as an appwrite_escalation, that " +
					"notifies the team step by step until a member acknowledges the alert. When set, notify is " +
					"not used. Defaults to empty, which notifies the notify roles once.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(""),
			},

			"url": schema.StringAttribute{
				Description: "The URL an http monitor requests, with an http or https scheme. Required for http " +
					"monitors; not accepted for other types.",
				Optional:   true,
				Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"method": schema.StringAttribute{
				Description: "The HTTP method an http monitor requests the URL with: " + strings.Join(methods, ", ") +
					". Defaults to GET for http monitors; not accepted for other types.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{defaultStringFor("GET", typeHTTP)},
				Validators:    []validator.String{stringvalidator.OneOf(methods...)},
			},
			"headers": schema.MapAttribute{
				Description: "HTTP headers an http monitor sends, as a map of header names and values. " +
					"Removing every header replaces the monitor, because Appwrite cannot clear a monitor's " +
					"headers in place. Not accepted for other types.",
				Optional:      true,
				Sensitive:     true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.Map{replaceWhenHeadersCleared()},
				Validators:    []validator.Map{mapvalidator.SizeAtLeast(1)},
			},
			"body": schema.StringAttribute{
				Description: fmt.Sprintf("The request body an http monitor sends, up to %d characters. Not "+
					"accepted for other types.", maxBodyLength),
				Optional:   true,
				Validators: []validator.String{stringvalidator.LengthBetween(1, maxBodyLength)},
			},
			"domain": schema.StringAttribute{
				Description: "The domain a dns monitor queries, such as example.com. Required for dns monitors; " +
					"not accepted for other types.",
				Optional:   true,
				Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"record": schema.StringAttribute{
				Description: "The record type a dns monitor queries: " + strings.Join(records, ", ") +
					". Defaults to A for dns monitors; not accepted for other types.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{defaultStringFor("A", typeDNS)},
				Validators:    []validator.String{stringvalidator.OneOf(records...)},
			},
			"resolver": schema.StringAttribute{
				Description: "The resolver a dns monitor queries, as an IP address with an optional port, such as " +
					"1.1.1.1 or [2606:4700:4700::1111]:53. Left out, the server's resolver is used. Not accepted " +
					"for other types.",
				Optional:   true,
				Validators: []validator.String{stringvalidator.LengthBetween(1, 128)},
			},
			"host": schema.StringAttribute{
				Description: "The hostname or IP address a tcp or smtp monitor connects to. Required for tcp and " +
					"smtp monitors; not accepted for other types.",
				Optional:   true,
				Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"port": schema.Int64Attribute{
				Description: "The port a tcp or smtp monitor connects to, from 1 to 65535. Required for tcp and " +
					"smtp monitors; not accepted for other types.",
				Optional:   true,
				Validators: []validator.Int64{int64validator.Between(1, 65535)},
			},
			"tls_mode": schema.StringAttribute{
				Description: "How an smtp monitor secures the connection: auto (implicit on port 465, starttls on " +
					"587 and none otherwise), none, starttls or implicit. Defaults to auto for smtp monitors; not " +
					"accepted for other types.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{defaultStringFor("auto", typeSMTP)},
				Validators:    []validator.String{stringvalidator.OneOf(tlsModes...)},
			},
			"window": schema.Int64Attribute{
				Description: fmt.Sprintf("Seconds a usage monitor sums a counter metric over, from 60 to %d. "+
					"Gauges compare their latest value. Defaults to 3600 for usage monitors; not accepted for "+
					"other types.", maxSeconds),
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{defaultInt64For(3600, typeUsage)},
				Validators:    []validator.Int64{int64validator.Between(60, maxSeconds)},
			},

			"status": schema.StringAttribute{
				Description: "Whether the monitor is firing, as of its last evaluation.",
				Computed:    true,
			},
			"created_at": schema.StringAttribute{
				Description:   "The monitor creation timestamp in ISO 8601 format.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Description:   "The monitor last update timestamp in ISO 8601 format.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{common.UseStateForUnknownUnlessUpdating()},
			},
			"project_id": common.ProjectIDAttribute(),
		},
	}
}

// typedField is an argument only some monitor types take.
type typedField struct {
	name     string
	types    []string
	value    attr.Value
	required bool
}

// ValidateConfig catches at plan time what the API would otherwise reject
// mid-apply: an argument on a monitor type that has no use for it, or a
// missing one that the type requires.
func (r *monitorResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config monitorResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || config.Type.IsUnknown() || config.Type.IsNull() {
		return
	}
	monitorType := config.Type.ValueString()

	if monitorType == typeUsage && config.Rules.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("rules"), "Missing required argument",
			"usage monitors require rules: one comparing a usage metric with a threshold.")
	}

	// A check that outlasts its interval would still be running when the next
	// one starts, so the API refuses it. Left out, timeout is 10.
	if slices.Contains(probeTypes, monitorType) && !config.Interval.IsNull() && !config.Interval.IsUnknown() && !config.Timeout.IsUnknown() {
		timeout := int64(10)
		if !config.Timeout.IsNull() {
			timeout = config.Timeout.ValueInt64()
		}
		if timeout > config.Interval.ValueInt64() {
			resp.Diagnostics.AddAttributeError(path.Root("timeout"), "Invalid argument combination",
				fmt.Sprintf("timeout (%d) must not be longer than interval (%d).", timeout, config.Interval.ValueInt64()))
		}
	}

	fields := []typedField{
		{"interval", probeTypes, config.Interval, false},
		{"timeout", probeTypes, config.Timeout, false},
		{"url", []string{typeHTTP}, config.URL, true},
		{"method", []string{typeHTTP}, config.Method, false},
		{"headers", []string{typeHTTP}, config.Headers, false},
		{"body", []string{typeHTTP}, config.Body, false},
		{"domain", []string{typeDNS}, config.Domain, true},
		{"record", []string{typeDNS}, config.Record, false},
		{"resolver", []string{typeDNS}, config.Resolver, false},
		{"host", []string{typeTCP, typeSMTP}, config.Host, true},
		{"port", []string{typeTCP, typeSMTP}, config.Port, true},
		{"tls_mode", []string{typeSMTP}, config.TLSMode, false},
		{"window", []string{typeUsage}, config.Window, false},
	}
	for _, field := range fields {
		// A value from another resource may not be known until apply, and
		// may turn out null. Only a value known now can be judged.
		if field.value.IsUnknown() {
			continue
		}
		used := slices.Contains(field.types, monitorType)
		switch {
		case used && field.required && field.value.IsNull():
			resp.Diagnostics.AddAttributeError(path.Root(field.name), "Missing required argument",
				fmt.Sprintf("%s monitors require %s.", monitorType, field.name))
		case !used && !field.value.IsNull():
			resp.Diagnostics.AddAttributeError(path.Root(field.name), "Unsupported argument",
				fmt.Sprintf("%s is used only by %s monitors, not %s monitors. Remove it from the configuration.",
					field.name, strings.Join(field.types, " and "), monitorType))
		}
	}
}

func (r *monitorResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *monitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan monitorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectID, client, ok := r.client(plan.ProjectID, &resp.Diagnostics)
	if !ok {
		return
	}

	monitorID := plan.ID.ValueString()
	if plan.ID.IsNull() || plan.ID.IsUnknown() {
		monitorID = id.Unique()
	}

	f := changedFields(ctx, plan, nil, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	monitor, err := createMonitor(client, plan.Type.ValueString(), monitorID, f)
	if err != nil {
		resp.Diagnostics.AddError("Error creating monitor", common.FormatErrorWithAuthGuidance(err, common.ProjectCredentialGuidance(resourceName, scopes...)))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	resp.Diagnostics.Append(mapToState(ctx, monitor, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *monitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state monitorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectID, client, ok := r.client(state.ProjectID, &resp.Diagnostics)
	if !ok {
		return
	}

	monitor, err := client.Get(state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading monitor", common.FormatErrorWithAuthGuidance(err, common.ProjectCredentialGuidance(resourceName, scopes...)))
		return
	}

	state.ProjectID = types.StringValue(projectID)
	resp.Diagnostics.Append(mapToState(ctx, monitor, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *monitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state monitorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	projectID, client, ok := r.client(plan.ProjectID, &resp.Diagnostics)
	if !ok {
		return
	}

	f := changedFields(ctx, plan, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	monitor, err := updateMonitor(client, plan.Type.ValueString(), plan.ID.ValueString(), f)
	if err != nil {
		resp.Diagnostics.AddError("Error updating monitor", common.FormatErrorWithAuthGuidance(err, common.ProjectCredentialGuidance(resourceName, scopes...)))
		return
	}

	plan.ProjectID = types.StringValue(projectID)
	resp.Diagnostics.Append(mapToState(ctx, monitor, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *monitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state monitorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	_, client, ok := r.client(state.ProjectID, &resp.Diagnostics)
	if !ok {
		return
	}

	_, err := client.Delete(state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting monitor", common.FormatErrorWithAuthGuidance(err, common.ProjectCredentialGuidance(resourceName, scopes...)))
	}
}

func (r *monitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	const expected = "monitor_id or project_id/monitor_id"

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

func (r *monitorResource) client(projectID types.String, diags *diag.Diagnostics) (string, *monitors.Monitors, bool) {
	resolved, err := common.ResolveProjectID(r.clients, projectID)
	if err != nil {
		diags.AddError("Error resolving project ID", err.Error())
		return "", nil, false
	}
	if err := common.ValidateProjectCredential(r.clients, resourceName, scopes...); err != nil {
		diags.AddError("Invalid credential", err.Error())
		return "", nil, false
	}
	return resolved, appwrite.NewMonitors(r.clients.ClientForProject(resolved)), true
}
