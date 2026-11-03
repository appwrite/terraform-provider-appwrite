package domain

import (
	"net/netip"
	"regexp"
	"sort"
	"strings"

	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/domains"
	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/appwrite/terraform-provider-appwrite/internal/common"
)

// Domains are organization resources: they belong to a team, live in the
// console project, and are authorized with an organization key's domains.*
// scopes. Every resource in this package shares the one credential check.
var domainScopes = []string{"domains.read", "domains.write"}

func domainsClient(clients *common.AppwriteClients, resourceName, organizationID string) (*domains.Domains, error) {
	return domainsClientWithScopes(clients, resourceName, organizationID, domainScopes...)
}

func domainsClientWithScopes(clients *common.AppwriteClients, resourceName, organizationID string, scopes ...string) (*domains.Domains, error) {
	if err := common.ValidateOrganizationCredential(clients, resourceName, scopes...); err != nil {
		return nil, err
	}
	return appwrite.NewDomains(clients.ClientForOrganization(organizationID)), nil
}

func credentialGuidance(resourceName string) string {
	return common.OrganizationCredentialGuidance(resourceName, domainScopes...)
}

// Record types the Domains API can create, each with its own route.
const (
	recordTypeA     = "A"
	recordTypeAAAA  = "AAAA"
	recordTypeALIAS = "ALIAS"
	recordTypeCAA   = "CAA"
	recordTypeCNAME = "CNAME"
	recordTypeHTTPS = "HTTPS"
	recordTypeMX    = "MX"
	recordTypeNS    = "NS"
	recordTypeSRV   = "SRV"
	recordTypeTXT   = "TXT"
)

var recordTypes = []string{
	recordTypeA, recordTypeAAAA, recordTypeALIAS, recordTypeCAA, recordTypeCNAME,
	recordTypeHTTPS, recordTypeMX, recordTypeNS, recordTypeSRV, recordTypeTXT,
}

// The server rejects a priority, weight or port outside a 16-bit unsigned range,
// and a TTL that is not a positive 32-bit integer.
const (
	minTTL       = 1
	maxTTL       = 2147483647
	defaultTTL   = 3600
	maxUint16    = 65535
	maxTXTLength = 16384
	// Every value other than a TXT record's is capped at 255 characters.
	maxValueLength   = 255
	maxCommentLength = 16384
)

// domainNamePattern accepts a lowercase domain name with no scheme, path, port
// or trailing dot.
//
// Appwrite stores the name as written but derives the domain ID from its
// lowercase form, so "Example.com" and "example.com" are the same domain under
// two spellings. Accepting only the lowercase form keeps one configuration from
// planning a replacement against a domain that has not changed.
var domainNamePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$`)

func domainNameValidators() []validator.String {
	return []validator.String{
		stringvalidator.LengthBetween(1, 253),
		stringvalidator.RegexMatches(
			domainNamePattern,
			"must be a lowercase domain name such as example.com, without a scheme, path or trailing dot",
		),
	}
}

// nameserverPattern accepts a hostname, optionally written absolute with a
// trailing dot. The server checks each entry against its own hostname rules;
// this only catches what is plainly not a hostname before the apply starts.
var nameserverPattern = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9-]*[A-Za-z0-9])?(\.[A-Za-z0-9]([A-Za-z0-9-]*[A-Za-z0-9])?)+\.?$`)

// normalizeRecordName returns a record name in the form Appwrite stores it,
// which is what the API returns on every read.
//
// It mirrors Appwrite\Cloud\DNS\Records::normalize: lowercase, no trailing dot,
// "@" for the zone apex, and relative to the zone when written in full. The
// provider needs the same rule so that a name written "www.example.com." in
// configuration is recognized as the "www" the server hands back, rather than
// shown as a change on every plan.
func normalizeRecordName(name, zone string) string {
	zone = strings.ToLower(zone)
	name = strings.ToLower(strings.TrimRight(strings.TrimSpace(name), "."))
	if zone == "" {
		if name == "" {
			return "@"
		}
		return name
	}

	suffix := "." + zone
	switch {
	case name == "" || name == zone:
		return "@"
	// A name that would still end in the zone once stripped stays absolute;
	// the server keeps it as written so its zone export round-trips.
	case name == zone+suffix, strings.HasSuffix(name, suffix+suffix):
		return name
	case strings.HasSuffix(name, suffix):
		return strings.TrimSuffix(name, suffix)
	default:
		return name
	}
}

// normalizeRecordValue returns a record value in the form Appwrite stores it.
// Only values that are host names are rewritten: they are served lowercase and
// absolute, so the server drops the case and the trailing dot. The root ".",
// a null MX or an SRV for no service, is kept as it is.
func normalizeRecordValue(recordType, value string) string {
	switch recordType {
	case recordTypeCNAME, recordTypeNS, recordTypeMX, recordTypeSRV, recordTypeALIAS:
		target := strings.ToLower(strings.TrimSpace(value))
		if target == "." {
			return target
		}
		return strings.TrimRight(target, ".")
	default:
		return value
	}
}

// preferConfigured returns the prior value when it is a different spelling of
// what the API returned, and the API's value otherwise.
//
// Writing the API's form back would make every refresh disagree with a
// configuration that wrote the name another valid way, which Terraform reports
// either as a perpetual diff or, straight after an apply, as a provider bug.
// Anything that normalizes differently is a real change and is reported.
//
// Both sides are normalized because records written before Appwrite began
// normalizing are still stored as they were written.
func preferConfigured(prior types.String, actual string, normalize func(string) string) types.String {
	if !prior.IsNull() && !prior.IsUnknown() && normalize(prior.ValueString()) == normalize(actual) {
		return prior
	}
	return types.StringValue(actual)
}

// usesPriority, usesWeight and usesPort report which of the numeric fields a
// record type takes. The API returns 0 for the fields a type does not use, which
// the provider stores as null so that leaving them out of configuration is not
// a diff.
func usesPriority(recordType string) bool {
	return recordType == recordTypeMX || recordType == recordTypeSRV
}

func usesWeightAndPort(recordType string) bool {
	return recordType == recordTypeSRV
}

func isValidIPv4(value string) bool {
	addr, err := netip.ParseAddr(value)
	return err == nil && addr.Is4()
}

func isValidIPv6(value string) bool {
	addr, err := netip.ParseAddr(value)
	// A zone index (fe80::1%eth0) has no meaning in a DNS record.
	return err == nil && addr.Is6() && addr.Zone() == ""
}

// appwriteNameservers returns the nameservers Appwrite serves a domain's zone
// from.
//
// The API has no field for them. Appwrite writes one locked NS record at the
// apex for each of its nameservers when the domain is added, so those records
// are the authoritative list for this installation, and reading them avoids
// hard-coding hostnames that differ between Cloud regions and self-hosted
// deployments.
//
// Records written before Appwrite began normalizing can hold the apex as the
// domain name rather than "@", so the name is normalized before it is compared.
func appwriteNameservers(domain *models.Domain) []string {
	nameservers := []string{}
	for _, record := range domain.DnsRecords {
		if record.Type == recordTypeNS && record.Lock && normalizeRecordName(record.Name, domain.Domain) == "@" {
			nameservers = append(nameservers, normalizeRecordValue(recordTypeNS, record.Value))
		}
	}
	sort.Strings(nameservers)
	return nameservers
}

// nameserversVerified reports whether Appwrite last found the domain delegated
// to its own nameservers. The API stores that as the string "Appwrite", and as
// an empty string otherwise.
func nameserversVerified(domain *models.Domain) bool {
	return domain.Nameservers == "Appwrite"
}

// Descriptions shared across the package's schemas.
const (
	domainIDDescription  = "The domain ID."
	registrarDescription = "Who the domain is registered through: appwrite for a domain bought or transferred " +
		"through Appwrite. Any other value, including an empty one, means it is registered elsewhere."
)
