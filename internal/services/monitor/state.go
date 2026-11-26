package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"

	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// changedFields returns the arguments to send. On create (state nil) that is
// every known argument; on update only those that differ from state, because
// the update routes change only what they are sent. Removing rules, headers,
// body or resolver from the configuration sends their empty value, which puts
// the server's default back.
func changedFields(ctx context.Context, plan monitorResourceModel, state *monitorResourceModel, diags *diag.Diagnostics) fields {
	changed := func(planned, prior attr.Value) bool {
		if planned.IsUnknown() {
			return false
		}
		if state == nil {
			return !planned.IsNull()
		}
		return !planned.Equal(prior)
	}
	var prior monitorResourceModel
	if state != nil {
		prior = *state
	}

	var f fields
	if changed(plan.Name, prior.Name) {
		f.Name = plan.Name.ValueStringPointer()
	}
	if changed(plan.Rules, prior.Rules) {
		rules := []string{}
		if !plan.Rules.IsNull() {
			diags.Append(plan.Rules.ElementsAs(ctx, &rules, false)...)
		}
		f.Rules = &rules
	}
	if changed(plan.Interval, prior.Interval) && !plan.Interval.IsNull() {
		f.Interval = intPointer(plan.Interval)
	}
	if changed(plan.Timeout, prior.Timeout) && !plan.Timeout.IsNull() {
		f.Timeout = intPointer(plan.Timeout)
	}
	if changed(plan.Cooldown, prior.Cooldown) {
		f.Cooldown = intPointer(plan.Cooldown)
	}
	if changed(plan.Enabled, prior.Enabled) {
		f.Enabled = plan.Enabled.ValueBoolPointer()
	}
	if changed(plan.Notify, prior.Notify) {
		notify := []string{}
		diags.Append(plan.Notify.ElementsAs(ctx, &notify, false)...)
		f.Notify = &notify
	}
	if changed(plan.Severity, prior.Severity) {
		f.Severity = plan.Severity.ValueStringPointer()
	}
	if changed(plan.EscalationID, prior.EscalationID) {
		f.EscalationID = plan.EscalationID.ValueStringPointer()
	}

	if changed(plan.URL, prior.URL) {
		f.URL = plan.URL.ValueStringPointer()
	}
	if changed(plan.Method, prior.Method) && !plan.Method.IsNull() {
		f.Method = plan.Method.ValueStringPointer()
	}
	if changed(plan.Headers, prior.Headers) {
		headers := map[string]string{}
		if !plan.Headers.IsNull() {
			diags.Append(plan.Headers.ElementsAs(ctx, &headers, false)...)
		}
		f.Headers = &headers
	}
	if changed(plan.Body, prior.Body) {
		f.Body = new(plan.Body.ValueString())
	}
	if changed(plan.Domain, prior.Domain) {
		f.Domain = plan.Domain.ValueStringPointer()
	}
	if changed(plan.Record, prior.Record) && !plan.Record.IsNull() {
		f.Record = plan.Record.ValueStringPointer()
	}
	if changed(plan.Resolver, prior.Resolver) {
		f.Resolver = new(plan.Resolver.ValueString())
	}
	if changed(plan.Host, prior.Host) {
		f.Host = plan.Host.ValueStringPointer()
	}
	if changed(plan.Port, prior.Port) {
		f.Port = intPointer(plan.Port)
	}
	if changed(plan.TLSMode, prior.TLSMode) && !plan.TLSMode.IsNull() {
		f.TLSMode = plan.TLSMode.ValueStringPointer()
	}
	if changed(plan.Window, prior.Window) && !plan.Window.IsNull() {
		f.Window = intPointer(plan.Window)
	}
	return f
}

func intPointer(value types.Int64) *int {
	v := int(value.ValueInt64())
	return &v
}

// mapToState copies monitor into model. Arguments a monitor's type does not
// take are null, and an empty optional value the configuration left out stays
// null rather than becoming an empty string or map.
func mapToState(ctx context.Context, monitor *models.Monitor, model *monitorResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	monitorType := monitor.Type
	// An import starts from the ID alone, so nothing records whether the
	// configuration sets rules; read them back then rather than leaving them out.
	importing := model.Name.IsNull()

	model.ID = types.StringValue(monitor.Id)
	model.Type = types.StringValue(monitorType)
	model.Name = types.StringValue(monitor.Name)
	model.Cooldown = types.Int64Value(int64(monitor.Cooldown))
	model.Enabled = types.BoolValue(monitor.Enabled)
	model.Severity = types.StringValue(monitor.Severity)
	model.EscalationID = types.StringValue(monitor.EscalationId)
	model.Status = types.StringValue(monitor.Status)
	model.CreatedAt = types.StringValue(monitor.CreatedAt)
	model.UpdatedAt = types.StringValue(monitor.UpdatedAt)

	notify, d := types.ListValueFrom(ctx, types.StringType, nonNil(monitor.Notify))
	diags.Append(d...)
	model.Notify = notify

	if !model.Rules.IsNull() || importing {
		rules, d := mergeRules(ctx, model.Rules, nonNil(monitor.Rules))
		diags.Append(d...)
		model.Rules = rules
	}

	probe := slices.Contains(probeTypes, monitorType)
	model.Interval = int64When(probe, monitor.Interval)
	model.Timeout = int64When(probe, monitor.Timeout)
	model.Window = int64When(monitorType == typeUsage, monitor.Window)

	http := monitorType == typeHTTP
	model.URL = stringWhen(http, monitor.Url)
	model.Method = stringWhen(http, common.Deref(monitor.Method))
	model.Body = stringWhen(http && monitor.Body != "", monitor.Body)
	headers, d := decodeHeaders(ctx, monitor.Headers)
	diags.Append(d...)
	if !http || len(headers.Elements()) == 0 {
		headers = types.MapNull(types.StringType)
	}
	model.Headers = headers

	dns := monitorType == typeDNS
	model.Domain = stringWhen(dns, monitor.Domain)
	model.Record = stringWhen(dns, common.Deref(monitor.Record))
	model.Resolver = stringWhen(dns && monitor.Resolver != "", monitor.Resolver)

	connects := monitorType == typeTCP || monitorType == typeSMTP
	model.Host = stringWhen(connects, monitor.Host)
	model.Port = int64When(connects, monitor.Port)
	model.TLSMode = stringWhen(monitorType == typeSMTP, common.Deref(monitor.TlsMode))

	return diags
}

func stringWhen(used bool, value string) types.String {
	if !used {
		return types.StringNull()
	}
	return types.StringValue(value)
}

func int64When(used bool, value int) types.Int64 {
	if !used {
		return types.Int64Null()
	}
	return types.Int64Value(int64(value))
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

// decodeHeaders reads the headers object an http monitor returns. The SDK
// leaves it untyped.
func decodeHeaders(ctx context.Context, raw any) (types.Map, diag.Diagnostics) {
	headers := map[string]string{}
	if object, ok := raw.(map[string]any); ok {
		for name, value := range object {
			if text, isString := value.(string); isString {
				headers[name] = text
			} else {
				headers[name] = fmt.Sprint(value)
			}
		}
	}
	return types.MapValueFrom(ctx, types.StringType, headers)
}

// mergeRules keeps the configured rules when the server returned the same
// rules, and the server's otherwise. The server stores each rule re-encoded,
// which can order keys or format numbers differently from the configuration,
// and a usage monitor's scope rules are stored after its metric rule. Taking
// that encoding would show as a diff on every plan.
func mergeRules(ctx context.Context, current types.List, returned []string) (types.List, diag.Diagnostics) {
	var configured []string
	if !current.IsNull() && !current.IsUnknown() {
		if diags := current.ElementsAs(ctx, &configured, false); diags.HasError() {
			return current, diags
		}
		if sameRules(configured, returned) {
			return current, nil
		}
	}
	return types.ListValueFrom(ctx, types.StringType, returned)
}

// sameRules reports whether a and b hold the same rules in any order,
// comparing each as decoded JSON.
func sameRules(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	decode := func(rule string) any {
		var value any
		if err := json.Unmarshal([]byte(rule), &value); err != nil {
			return rule
		}
		return value
	}
	remaining := make([]any, len(b))
	for i, rule := range b {
		remaining[i] = decode(rule)
	}
	for _, rule := range a {
		decoded := decode(rule)
		index := slices.IndexFunc(remaining, func(candidate any) bool { return reflect.DeepEqual(candidate, decoded) })
		if index < 0 {
			return false
		}
		remaining = slices.Delete(remaining, index, index+1)
	}
	return true
}
