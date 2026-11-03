package domain

import (
	"context"
	"fmt"
	"strings"

	"github.com/appwrite/sdk-for-go/v7/domains"
	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
)

const recordResourceName = "appwrite_domain_record"

var (
	_ resource.Resource                   = &recordResource{}
	_ resource.ResourceWithConfigure      = &recordResource{}
	_ resource.ResourceWithImportState    = &recordResource{}
	_ resource.ResourceWithValidateConfig = &recordResource{}
)

type recordResource struct {
	clients *common.AppwriteClients
}

type recordResourceModel struct {
	ID             types.String `tfsdk:"id"`
	DomainID       types.String `tfsdk:"domain_id"`
	Domain         types.String `tfsdk:"domain"`
	Type           types.String `tfsdk:"type"`
	Name           types.String `tfsdk:"name"`
	Value          types.String `tfsdk:"value"`
	TTL            types.Int64  `tfsdk:"ttl"`
	Priority       types.Int64  `tfsdk:"priority"`
	Weight         types.Int64  `tfsdk:"weight"`
	Port           types.Int64  `tfsdk:"port"`
	Comment        types.String `tfsdk:"comment"`
	Locked         types.Bool   `tfsdk:"locked"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
	OrganizationID types.String `tfsdk:"organization_id"`
}

func NewRecordResource() resource.Resource {
	return &recordResource{}
}

func (r *recordResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_record"
}

func (r *recordResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	uint16Range := []validator.Int64{int64validator.Between(0, maxUint16)}
	resp.Schema = schema.Schema{
		Description: "Manages a DNS record in the zone Appwrite hosts for an organization domain.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:   "The DNS record ID.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"domain_id": schema.StringAttribute{
				Description:   "The ID of the domain whose zone holds the record. " + common.ForcesReplacementNote,
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"domain": schema.StringAttribute{
				Description:   "The name of the domain whose zone holds the record.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"type": schema.StringAttribute{
				Description: "The record type: " + strings.Join(recordTypes, ", ") + ". " +
					common.ForcesReplacementNote,
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.OneOf(recordTypes...)},
			},
			"name": schema.StringAttribute{
				Description: "The record name relative to the domain, such as www, or @ for the domain itself. " +
					"A full name ending in the domain is also accepted. For SRV records, the service and " +
					"protocol, such as _sip._tcp.",
				Required:   true,
				Validators: []validator.String{stringvalidator.LengthBetween(1, 255)},
			},
			"value": schema.StringAttribute{
				Description: "The record value: an IPv4 address for A, an IPv6 address for AAAA, a host name for " +
					"ALIAS, CNAME, MX, NS and SRV, a CAA value such as 0 issue \"letsencrypt.org\", or the " +
					"text of a TXT record. TXT values may be up to 16384 characters, others up to 255.",
				Required:   true,
				Validators: []validator.String{stringvalidator.LengthBetween(1, maxTXTLength)},
			},
			"ttl": schema.Int64Attribute{
				Description: fmt.Sprintf("Time to live, in seconds. Defaults to %d.", defaultTTL),
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(defaultTTL),
				Validators:  []validator.Int64{int64validator.Between(minTTL, maxTTL)},
			},
			"priority": schema.Int64Attribute{
				Description: "The record priority, from 0 to 65535. Lower values are tried first. " +
					"Required for MX and SRV records and not accepted for other types.",
				Optional:   true,
				Validators: uint16Range,
			},
			"weight": schema.Int64Attribute{
				Description: "The weight used to share load between SRV targets of equal priority, from 0 to " +
					"65535. Required for SRV records and not accepted for other types.",
				Optional:   true,
				Validators: uint16Range,
			},
			"port": schema.Int64Attribute{
				Description: "The port the SRV target serves on, from 0 to 65535. Required for SRV records " +
					"and not accepted for other types.",
				Optional:   true,
				Validators: uint16Range,
			},
			"comment": schema.StringAttribute{
				Description: "A comment explaining what the record is for.",
				Optional:    true,
				Validators:  []validator.String{stringvalidator.LengthAtMost(maxCommentLength)},
			},
			"locked": schema.BoolAttribute{
				Description: "Whether Appwrite manages the record. Locked records cannot be changed or " +
					"deleted, so they cannot be managed by Terraform; a record this resource creates is never locked.",
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				Description:   "The record creation timestamp in ISO 8601 format.",
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Description: "The record last update timestamp in ISO 8601 format.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					common.UseStateForUnknownUnlessUpdating(),
				},
			},
			"organization_id": common.OrganizationIDAttribute(),
		},
	}
}

// ValidateConfig catches at plan time what the API would otherwise reject
// mid-apply: a numeric field on a type that has no use for it, a missing one on
// a type that requires it, and a value that cannot be the type's.
func (r *recordResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config recordResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || config.Type.IsUnknown() || config.Type.IsNull() {
		return
	}
	resp.Diagnostics.Append(validateRecordConfig(config)...)
}

func validateRecordConfig(config recordResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	recordType := config.Type.ValueString()

	fields := []struct {
		name  string
		value types.Int64
		used  bool
	}{
		{"priority", config.Priority, usesPriority(recordType)},
		{"weight", config.Weight, usesWeightAndPort(recordType)},
		{"port", config.Port, usesWeightAndPort(recordType)},
	}
	for _, field := range fields {
		// A value from another resource may not be known until apply, and
		// may turn out null. Only a value known now can be judged.
		if field.value.IsUnknown() {
			continue
		}
		switch {
		case field.used && field.value.IsNull():
			diags.AddAttributeError(path.Root(field.name), "Missing required argument",
				fmt.Sprintf("%s records require %s.", recordType, field.name))
		case !field.used && !field.value.IsNull():
			diags.AddAttributeError(path.Root(field.name), "Unsupported argument",
				fmt.Sprintf("%s is not used by %s records. Remove it from the configuration.", field.name, recordType))
		}
	}

	if config.Value.IsUnknown() || config.Value.IsNull() {
		return diags
	}
	value := config.Value.ValueString()
	switch {
	case recordType == recordTypeA && !isValidIPv4(value):
		diags.AddAttributeError(path.Root("value"), "Invalid record value",
			fmt.Sprintf("A records take an IPv4 address, got %q.", value))
	case recordType == recordTypeAAAA && !isValidIPv6(value):
		diags.AddAttributeError(path.Root("value"), "Invalid record value",
			fmt.Sprintf("AAAA records take an IPv6 address, got %q.", value))
	case recordType != recordTypeTXT && len(value) > maxValueLength:
		diags.AddAttributeError(path.Root("value"), "Invalid record value",
			fmt.Sprintf("%s record values may be at most %d characters; only TXT values may be longer.", recordType, maxValueLength))
	}
	return diags
}

func (r *recordResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *recordResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan recordResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationID, client, ok := r.client(plan.OrganizationID, &resp.Diagnostics)
	if !ok {
		return
	}

	// The zone name is what turns a full name in configuration into the
	// relative one the API stores, so it is needed before the first read.
	domain, err := client.Get(plan.DomainID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading domain", common.FormatErrorWithAuthGuidance(err, credentialGuidance(recordResourceName)))
		return
	}

	record, err := createRecord(client, plan)
	if err != nil {
		resp.Diagnostics.AddError("Error creating DNS record", common.FormatErrorWithAuthGuidance(err, credentialGuidance(recordResourceName)))
		return
	}

	plan.OrganizationID = types.StringValue(organizationID)
	plan.Domain = types.StringValue(domain.Domain)
	mapRecordToState(record, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *recordResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state recordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationID, client, ok := r.client(state.OrganizationID, &resp.Diagnostics)
	if !ok {
		return
	}

	// Both a deleted record and a deleted domain come back as 404.
	record, err := client.GetRecord(state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil {
		if common.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading DNS record", common.FormatErrorWithAuthGuidance(err, credentialGuidance(recordResourceName)))
		return
	}

	// The zone name is set on create and kept in state, so it is fetched here
	// only after an import.
	if state.Domain.IsNull() || state.Domain.IsUnknown() || state.Domain.ValueString() == "" {
		domain, err := client.Get(state.DomainID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading domain", common.FormatErrorWithAuthGuidance(err, credentialGuidance(recordResourceName)))
			return
		}
		state.Domain = types.StringValue(domain.Domain)
	}

	state.OrganizationID = types.StringValue(organizationID)
	mapRecordToState(record, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *recordResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state recordResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	organizationID, client, ok := r.client(plan.OrganizationID, &resp.Diagnostics)
	if !ok {
		return
	}

	plan.ID = state.ID
	plan.Domain = state.Domain
	record, err := updateRecord(client, plan)
	if err != nil {
		resp.Diagnostics.AddError("Error updating DNS record", common.FormatErrorWithAuthGuidance(err, credentialGuidance(recordResourceName)))
		return
	}

	plan.OrganizationID = types.StringValue(organizationID)
	mapRecordToState(record, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *recordResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state recordResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, client, ok := r.client(state.OrganizationID, &resp.Diagnostics)
	if !ok {
		return
	}

	_, err := client.DeleteRecord(state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil && !common.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting DNS record", common.FormatErrorWithAuthGuidance(err, credentialGuidance(recordResourceName)))
	}
}

func (r *recordResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	const expected = "domain_id/record_id or organization_id/domain_id/record_id"

	parts := strings.Split(req.ID, "/")
	for _, part := range parts {
		if part == "" {
			resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected %s, got: %s", expected, req.ID))
			return
		}
	}

	organizationID := types.StringNull()
	var domainID, recordID string
	switch len(parts) {
	case 2:
		domainID, recordID = parts[0], parts[1]
	case 3:
		organizationID = types.StringValue(parts[0])
		domainID, recordID = parts[1], parts[2]
	default:
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected %s, got: %s", expected, req.ID))
		return
	}

	// A locked record would import cleanly and then fail on the first change
	// or on destroy, which is a worse time to find out. Refuse it here.
	resolvedOrganizationID, client, ok := r.client(organizationID, &resp.Diagnostics)
	if !ok {
		return
	}
	record, err := client.GetRecord(domainID, recordID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading DNS record", common.FormatErrorWithAuthGuidance(err, credentialGuidance(recordResourceName)))
		return
	}
	if record.Lock {
		resp.Diagnostics.AddError("Cannot import a locked DNS record",
			fmt.Sprintf("DNS record %q is locked: Appwrite created it for its own routing and does not allow it to be "+
				"changed or deleted, so Terraform cannot manage it.", recordID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), resolvedOrganizationID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), domainID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), recordID)...)
}

func (r *recordResource) client(organizationID types.String, diags *diag.Diagnostics) (string, *domains.Domains, bool) {
	resolved, err := common.ResolveOrganizationID(r.clients, organizationID)
	if err != nil {
		diags.AddError("Error resolving organization ID", err.Error())
		return "", nil, false
	}
	client, err := domainsClient(r.clients, recordResourceName, resolved)
	if err != nil {
		diags.AddError("Unsupported Appwrite credential", err.Error())
		return "", nil, false
	}
	return resolved, client, true
}

// createRecord sends the create for the planned record's type. Each type has
// its own route, and MX and SRV take extra positional arguments.
func createRecord(client *domains.Domains, plan recordResourceModel) (*models.DnsRecord, error) {
	domainID := plan.DomainID.ValueString()
	name := plan.Name.ValueString()
	value := plan.Value.ValueString()
	ttl := int(plan.TTL.ValueInt64())
	comment, hasComment := plan.Comment.ValueString(), !plan.Comment.IsNull()

	switch plan.Type.ValueString() {
	case recordTypeA:
		var opts []domains.CreateRecordAOption
		if hasComment {
			opts = append(opts, client.WithCreateRecordAComment(comment))
		}
		return client.CreateRecordA(domainID, name, value, ttl, opts...)
	case recordTypeAAAA:
		var opts []domains.CreateRecordAAAAOption
		if hasComment {
			opts = append(opts, client.WithCreateRecordAAAAComment(comment))
		}
		return client.CreateRecordAAAA(domainID, name, value, ttl, opts...)
	case recordTypeALIAS:
		var opts []domains.CreateRecordAliasOption
		if hasComment {
			opts = append(opts, client.WithCreateRecordAliasComment(comment))
		}
		return client.CreateRecordAlias(domainID, name, value, ttl, opts...)
	case recordTypeCAA:
		var opts []domains.CreateRecordCAAOption
		if hasComment {
			opts = append(opts, client.WithCreateRecordCAAComment(comment))
		}
		return client.CreateRecordCAA(domainID, name, value, ttl, opts...)
	case recordTypeCNAME:
		var opts []domains.CreateRecordCNAMEOption
		if hasComment {
			opts = append(opts, client.WithCreateRecordCNAMEComment(comment))
		}
		return client.CreateRecordCNAME(domainID, name, value, ttl, opts...)
	case recordTypeHTTPS:
		var opts []domains.CreateRecordHTTPSOption
		if hasComment {
			opts = append(opts, client.WithCreateRecordHTTPSComment(comment))
		}
		return client.CreateRecordHTTPS(domainID, name, value, ttl, opts...)
	case recordTypeMX:
		var opts []domains.CreateRecordMXOption
		if hasComment {
			opts = append(opts, client.WithCreateRecordMXComment(comment))
		}
		return client.CreateRecordMX(domainID, name, value, ttl, int(plan.Priority.ValueInt64()), opts...)
	case recordTypeNS:
		var opts []domains.CreateRecordNSOption
		if hasComment {
			opts = append(opts, client.WithCreateRecordNSComment(comment))
		}
		return client.CreateRecordNS(domainID, name, value, ttl, opts...)
	case recordTypeSRV:
		var opts []domains.CreateRecordSRVOption
		if hasComment {
			opts = append(opts, client.WithCreateRecordSRVComment(comment))
		}
		return client.CreateRecordSRV(domainID, name, value, ttl,
			int(plan.Priority.ValueInt64()), int(plan.Weight.ValueInt64()), int(plan.Port.ValueInt64()), opts...)
	case recordTypeTXT:
		// The TXT route alone treats the value as optional, so it is an
		// option here rather than a positional argument.
		opts := []domains.CreateRecordTXTOption{client.WithCreateRecordTXTValue(value)}
		if hasComment {
			opts = append(opts, client.WithCreateRecordTXTComment(comment))
		}
		return client.CreateRecordTXT(domainID, name, ttl, opts...)
	default:
		return nil, fmt.Errorf("unsupported DNS record type %q", plan.Type.ValueString())
	}
}

// updateRecord sends the update for the planned record's type.
//
// The comment is always sent. The update routes replace the whole record and
// treat a missing comment as an empty one, so leaving it out when unchanged
// would be harmless, but leaving it out when it was removed from configuration
// is exactly how it gets cleared -- sending it every time makes both cases the
// same request.
func updateRecord(client *domains.Domains, plan recordResourceModel) (*models.DnsRecord, error) {
	domainID := plan.DomainID.ValueString()
	recordID := plan.ID.ValueString()
	name := plan.Name.ValueString()
	value := plan.Value.ValueString()
	ttl := int(plan.TTL.ValueInt64())
	comment := plan.Comment.ValueString()

	switch plan.Type.ValueString() {
	case recordTypeA:
		return client.UpdateRecordA(domainID, recordID, name, value, ttl, client.WithUpdateRecordAComment(comment))
	case recordTypeAAAA:
		return client.UpdateRecordAAAA(domainID, recordID, name, value, ttl, client.WithUpdateRecordAAAAComment(comment))
	case recordTypeALIAS:
		return client.UpdateRecordAlias(domainID, recordID, name, value, ttl, client.WithUpdateRecordAliasComment(comment))
	case recordTypeCAA:
		return client.UpdateRecordCAA(domainID, recordID, name, value, ttl, client.WithUpdateRecordCAAComment(comment))
	case recordTypeCNAME:
		return client.UpdateRecordCNAME(domainID, recordID, name, value, ttl, client.WithUpdateRecordCNAMEComment(comment))
	case recordTypeHTTPS:
		return client.UpdateRecordHTTPS(domainID, recordID, name, value, ttl, client.WithUpdateRecordHTTPSComment(comment))
	case recordTypeMX:
		return client.UpdateRecordMX(domainID, recordID, name, value, ttl, int(plan.Priority.ValueInt64()),
			client.WithUpdateRecordMXComment(comment))
	case recordTypeNS:
		return client.UpdateRecordNS(domainID, recordID, name, value, ttl, client.WithUpdateRecordNSComment(comment))
	case recordTypeSRV:
		return client.UpdateRecordSRV(domainID, recordID, name, value, ttl,
			int(plan.Priority.ValueInt64()), int(plan.Weight.ValueInt64()), int(plan.Port.ValueInt64()),
			client.WithUpdateRecordSRVComment(comment))
	case recordTypeTXT:
		return client.UpdateRecordTXT(domainID, recordID, name, value, ttl, client.WithUpdateRecordTXTComment(comment))
	default:
		return nil, fmt.Errorf("unsupported DNS record type %q", plan.Type.ValueString())
	}
}

// mapRecordToState copies a record into the model. model.Domain must already
// hold the zone name, which is what recognizes a full name in configuration as
// the relative one the API returns.
func mapRecordToState(record *models.DnsRecord, model *recordResourceModel) {
	zone := model.Domain.ValueString()

	model.ID = types.StringValue(record.Id)
	model.Type = types.StringValue(record.Type)
	model.Name = preferConfigured(model.Name, record.Name, func(name string) string {
		return normalizeRecordName(name, zone)
	})
	model.Value = preferConfigured(model.Value, record.Value, func(value string) string {
		return normalizeRecordValue(record.Type, value)
	})
	model.TTL = types.Int64Value(int64(record.Ttl))
	model.Locked = types.BoolValue(record.Lock)
	model.CreatedAt = types.StringValue(record.CreatedAt)
	model.UpdatedAt = types.StringValue(record.UpdatedAt)

	model.Priority = types.Int64Null()
	if usesPriority(record.Type) {
		model.Priority = types.Int64Value(int64(record.Priority))
	}
	model.Weight, model.Port = types.Int64Null(), types.Int64Null()
	if usesWeightAndPort(record.Type) {
		model.Weight = types.Int64Value(int64(record.Weight))
		model.Port = types.Int64Value(int64(record.Port))
	}

	// The API returns an empty comment for a record that has none. An unset
	// argument stays null rather than becoming "" and showing as a diff.
	if record.Comment != "" || !model.Comment.IsNull() {
		model.Comment = types.StringValue(record.Comment)
	}
}
