package monitor

import (
	"fmt"

	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/appwrite/sdk-for-go/v7/monitors"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
)

// fields holds the arguments sent to a create or update route. A nil field is
// not sent, which on update leaves the stored value as it is.
type fields struct {
	Name         *string
	Rules        *[]string
	Interval     *int
	Timeout      *int
	Cooldown     *int
	Enabled      *bool
	Notify       *[]string
	Severity     *string
	EscalationID *string

	URL     *string
	Method  *string
	Headers *map[string]string
	Body    *string

	Domain   *string
	Record   *string
	Resolver *string

	Host    *string
	Port    *int
	TLSMode *string

	Window *int
}

// createMonitor sends f to the create route for monitorType. Each type has its
// own route and its own SDK call.
func createMonitor(client *monitors.Monitors, monitorType, monitorID string, f fields) (*models.Monitor, error) {
	name := common.Deref(f.Name)
	switch monitorType {
	case typeHTTP:
		var opts []monitors.CreateHttpMonitorOption
		add(&opts, f.Method, client.WithCreateHttpMonitorMethod)
		if f.Headers != nil {
			opts = append(opts, client.WithCreateHttpMonitorHeaders(*f.Headers))
		}
		add(&opts, f.Body, client.WithCreateHttpMonitorBody)
		add(&opts, f.Rules, client.WithCreateHttpMonitorRules)
		add(&opts, f.Interval, client.WithCreateHttpMonitorInterval)
		add(&opts, f.Timeout, client.WithCreateHttpMonitorTimeout)
		add(&opts, f.Cooldown, client.WithCreateHttpMonitorCooldown)
		add(&opts, f.Enabled, client.WithCreateHttpMonitorEnabled)
		add(&opts, f.Notify, client.WithCreateHttpMonitorNotify)
		add(&opts, f.Severity, client.WithCreateHttpMonitorSeverity)
		add(&opts, f.EscalationID, client.WithCreateHttpMonitorEscalationId)
		return client.CreateHttpMonitor(monitorID, name, common.Deref(f.URL), opts...)

	case typeDNS:
		var opts []monitors.CreateDnsMonitorOption
		add(&opts, f.Record, client.WithCreateDnsMonitorRecord)
		add(&opts, f.Resolver, client.WithCreateDnsMonitorResolver)
		add(&opts, f.Rules, client.WithCreateDnsMonitorRules)
		add(&opts, f.Interval, client.WithCreateDnsMonitorInterval)
		add(&opts, f.Timeout, client.WithCreateDnsMonitorTimeout)
		add(&opts, f.Cooldown, client.WithCreateDnsMonitorCooldown)
		add(&opts, f.Enabled, client.WithCreateDnsMonitorEnabled)
		add(&opts, f.Notify, client.WithCreateDnsMonitorNotify)
		add(&opts, f.Severity, client.WithCreateDnsMonitorSeverity)
		add(&opts, f.EscalationID, client.WithCreateDnsMonitorEscalationId)
		return client.CreateDnsMonitor(monitorID, name, common.Deref(f.Domain), opts...)

	case typeTCP:
		var opts []monitors.CreateTcpMonitorOption
		add(&opts, f.Rules, client.WithCreateTcpMonitorRules)
		add(&opts, f.Interval, client.WithCreateTcpMonitorInterval)
		add(&opts, f.Timeout, client.WithCreateTcpMonitorTimeout)
		add(&opts, f.Cooldown, client.WithCreateTcpMonitorCooldown)
		add(&opts, f.Enabled, client.WithCreateTcpMonitorEnabled)
		add(&opts, f.Notify, client.WithCreateTcpMonitorNotify)
		add(&opts, f.Severity, client.WithCreateTcpMonitorSeverity)
		add(&opts, f.EscalationID, client.WithCreateTcpMonitorEscalationId)
		return client.CreateTcpMonitor(monitorID, name, common.Deref(f.Host), common.Deref(f.Port), opts...)

	case typeSMTP:
		var opts []monitors.CreateSmtpMonitorOption
		add(&opts, f.TLSMode, client.WithCreateSmtpMonitorTlsMode)
		add(&opts, f.Rules, client.WithCreateSmtpMonitorRules)
		add(&opts, f.Interval, client.WithCreateSmtpMonitorInterval)
		add(&opts, f.Timeout, client.WithCreateSmtpMonitorTimeout)
		add(&opts, f.Cooldown, client.WithCreateSmtpMonitorCooldown)
		add(&opts, f.Enabled, client.WithCreateSmtpMonitorEnabled)
		add(&opts, f.Notify, client.WithCreateSmtpMonitorNotify)
		add(&opts, f.Severity, client.WithCreateSmtpMonitorSeverity)
		add(&opts, f.EscalationID, client.WithCreateSmtpMonitorEscalationId)
		return client.CreateSmtpMonitor(monitorID, name, common.Deref(f.Host), common.Deref(f.Port), opts...)

	case typeUsage:
		var opts []monitors.CreateUsageMonitorOption
		add(&opts, f.Window, client.WithCreateUsageMonitorWindow)
		add(&opts, f.Cooldown, client.WithCreateUsageMonitorCooldown)
		add(&opts, f.Enabled, client.WithCreateUsageMonitorEnabled)
		add(&opts, f.Notify, client.WithCreateUsageMonitorNotify)
		add(&opts, f.Severity, client.WithCreateUsageMonitorSeverity)
		add(&opts, f.EscalationID, client.WithCreateUsageMonitorEscalationId)
		return client.CreateUsageMonitor(monitorID, name, common.Deref(f.Rules), opts...)
	}
	return nil, fmt.Errorf("unsupported monitor type %q", monitorType)
}

// updateMonitor sends f to the update route for monitorType. Each route
// changes only the arguments it is sent.
func updateMonitor(client *monitors.Monitors, monitorType, monitorID string, f fields) (*models.Monitor, error) {
	switch monitorType {
	case typeHTTP:
		var opts []monitors.UpdateHttpMonitorOption
		add(&opts, f.Name, client.WithUpdateHttpMonitorName)
		add(&opts, f.URL, client.WithUpdateHttpMonitorUrl)
		add(&opts, f.Method, client.WithUpdateHttpMonitorMethod)
		if f.Headers != nil {
			opts = append(opts, client.WithUpdateHttpMonitorHeaders(*f.Headers))
		}
		add(&opts, f.Body, client.WithUpdateHttpMonitorBody)
		add(&opts, f.Rules, client.WithUpdateHttpMonitorRules)
		add(&opts, f.Interval, client.WithUpdateHttpMonitorInterval)
		add(&opts, f.Timeout, client.WithUpdateHttpMonitorTimeout)
		add(&opts, f.Cooldown, client.WithUpdateHttpMonitorCooldown)
		add(&opts, f.Enabled, client.WithUpdateHttpMonitorEnabled)
		add(&opts, f.Notify, client.WithUpdateHttpMonitorNotify)
		add(&opts, f.Severity, client.WithUpdateHttpMonitorSeverity)
		add(&opts, f.EscalationID, client.WithUpdateHttpMonitorEscalationId)
		return client.UpdateHttpMonitor(monitorID, opts...)

	case typeDNS:
		var opts []monitors.UpdateDnsMonitorOption
		add(&opts, f.Name, client.WithUpdateDnsMonitorName)
		add(&opts, f.Domain, client.WithUpdateDnsMonitorDomain)
		add(&opts, f.Record, client.WithUpdateDnsMonitorRecord)
		add(&opts, f.Resolver, client.WithUpdateDnsMonitorResolver)
		add(&opts, f.Rules, client.WithUpdateDnsMonitorRules)
		add(&opts, f.Interval, client.WithUpdateDnsMonitorInterval)
		add(&opts, f.Timeout, client.WithUpdateDnsMonitorTimeout)
		add(&opts, f.Cooldown, client.WithUpdateDnsMonitorCooldown)
		add(&opts, f.Enabled, client.WithUpdateDnsMonitorEnabled)
		add(&opts, f.Notify, client.WithUpdateDnsMonitorNotify)
		add(&opts, f.Severity, client.WithUpdateDnsMonitorSeverity)
		add(&opts, f.EscalationID, client.WithUpdateDnsMonitorEscalationId)
		return client.UpdateDnsMonitor(monitorID, opts...)

	case typeTCP:
		var opts []monitors.UpdateTcpMonitorOption
		add(&opts, f.Name, client.WithUpdateTcpMonitorName)
		add(&opts, f.Host, client.WithUpdateTcpMonitorHost)
		add(&opts, f.Port, client.WithUpdateTcpMonitorPort)
		add(&opts, f.Rules, client.WithUpdateTcpMonitorRules)
		add(&opts, f.Interval, client.WithUpdateTcpMonitorInterval)
		add(&opts, f.Timeout, client.WithUpdateTcpMonitorTimeout)
		add(&opts, f.Cooldown, client.WithUpdateTcpMonitorCooldown)
		add(&opts, f.Enabled, client.WithUpdateTcpMonitorEnabled)
		add(&opts, f.Notify, client.WithUpdateTcpMonitorNotify)
		add(&opts, f.Severity, client.WithUpdateTcpMonitorSeverity)
		add(&opts, f.EscalationID, client.WithUpdateTcpMonitorEscalationId)
		return client.UpdateTcpMonitor(monitorID, opts...)

	case typeSMTP:
		var opts []monitors.UpdateSmtpMonitorOption
		add(&opts, f.Name, client.WithUpdateSmtpMonitorName)
		add(&opts, f.Host, client.WithUpdateSmtpMonitorHost)
		add(&opts, f.Port, client.WithUpdateSmtpMonitorPort)
		add(&opts, f.TLSMode, client.WithUpdateSmtpMonitorTlsMode)
		add(&opts, f.Rules, client.WithUpdateSmtpMonitorRules)
		add(&opts, f.Interval, client.WithUpdateSmtpMonitorInterval)
		add(&opts, f.Timeout, client.WithUpdateSmtpMonitorTimeout)
		add(&opts, f.Cooldown, client.WithUpdateSmtpMonitorCooldown)
		add(&opts, f.Enabled, client.WithUpdateSmtpMonitorEnabled)
		add(&opts, f.Notify, client.WithUpdateSmtpMonitorNotify)
		add(&opts, f.Severity, client.WithUpdateSmtpMonitorSeverity)
		add(&opts, f.EscalationID, client.WithUpdateSmtpMonitorEscalationId)
		return client.UpdateSmtpMonitor(monitorID, opts...)

	case typeUsage:
		var opts []monitors.UpdateUsageMonitorOption
		add(&opts, f.Name, client.WithUpdateUsageMonitorName)
		add(&opts, f.Rules, client.WithUpdateUsageMonitorRules)
		add(&opts, f.Window, client.WithUpdateUsageMonitorWindow)
		add(&opts, f.Cooldown, client.WithUpdateUsageMonitorCooldown)
		add(&opts, f.Enabled, client.WithUpdateUsageMonitorEnabled)
		add(&opts, f.Notify, client.WithUpdateUsageMonitorNotify)
		add(&opts, f.Severity, client.WithUpdateUsageMonitorSeverity)
		add(&opts, f.EscalationID, client.WithUpdateUsageMonitorEscalationId)
		return client.UpdateUsageMonitor(monitorID, opts...)
	}
	return nil, fmt.Errorf("unsupported monitor type %q", monitorType)
}

// add appends the option for value when it is set.
func add[T any, O any](opts *[]O, value *T, option func(T) O) {
	if value != nil {
		*opts = append(*opts, option(*value))
	}
}
