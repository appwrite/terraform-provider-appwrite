// Package dedicated implements the Appwrite dedicated database resources.
//
// Appwrite exposes one route set per database engine (/postgresql, /mysql and
// /mongo). The Go SDK mirrors that with three separate services whose methods
// are identical in shape but whose option types are not interchangeable, so a
// single implementation cannot call them directly. The interfaces below adapt
// all three onto one shape, which lets the resources and data sources be
// written once and registered three times.
package dedicated

import (
	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/client"
	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/appwrite/sdk-for-go/v7/mongo"
	"github.com/appwrite/sdk-for-go/v7/mysql"
	"github.com/appwrite/sdk-for-go/v7/postgresql"
)

// Engine identifies which dedicated database API a resource talks to. The
// value doubles as the Terraform resource name segment, so
// appwrite_postgresql_database and siblings all come from one implementation.
type Engine string

const (
	EnginePostgresql Engine = "postgresql"
	EngineMysql      Engine = "mysql"
	EngineMongo      Engine = "mongo"
)

// Label returns the engine name as it reads in documentation and diagnostics.
func (e Engine) Label() string {
	switch e {
	case EnginePostgresql:
		return "PostgreSQL"
	case EngineMysql:
		return "MySQL"
	case EngineMongo:
		return "MongoDB"
	default:
		return string(e)
	}
}

// CreateOptions holds the optional arguments accepted when creating a
// dedicated database. A nil pointer means the attribute was not configured, so
// the option is omitted from the request and the server default applies.
type CreateOptions struct {
	Version                            *string
	Specification                      *string
	Replicas                           *int
	SyncMode                           *string
	NetworkIdleTimeoutSeconds          *int
	NetworkIPAllowlist                 []string
	IdleTimeoutMinutes                 *int
	Pitr                               *bool
	PitrRetentionDays                  *int
	StorageAutoscaling                 *bool
	StorageAutoscalingThresholdPercent *int
	StorageAutoscalingMaxGb            *int
}

// UpdateOptions holds the arguments accepted when updating a dedicated
// database. Version is absent because the engine version is changed through a
// dedicated upgrade route rather than a plain update.
type UpdateOptions struct {
	Name                               *string
	Status                             *string
	Specification                      *string
	Replicas                           *int
	SyncMode                           *string
	NetworkIdleTimeoutSeconds          *int
	NetworkIPAllowlist                 []string
	IdleTimeoutMinutes                 *int
	Pitr                               *bool
	PitrRetentionDays                  *int
	StorageAutoscaling                 *bool
	StorageAutoscalingThresholdPercent *int
	StorageAutoscalingMaxGb            *int
	SQLAPIEnabled                      *bool
	SQLAPIAllowedStatements            []string
	SQLAPIMaxRows                      *int
	SQLAPIMaxBytes                     *int
	SQLAPITimeoutSeconds               *int
}

// CreateBackupPolicyOptions holds the optional arguments for a new backup policy.
type CreateBackupPolicyOptions struct {
	Type    *string
	Enabled *bool
}

// UpdateBackupPolicyOptions holds the arguments for an existing backup policy.
// The backup type is create-only.
type UpdateBackupPolicyOptions struct {
	Name      *string
	Schedule  *string
	Retention *int
	Enabled   *bool
}

// PoolerOptions holds the arguments accepted when updating a connection pooler.
type PoolerOptions struct {
	Mode                *string
	MaxConnections      *int
	DefaultPoolSize     *int
	ReadWriteSplitting  *bool
	PoolerCPURequest    *string
	PoolerCPULimit      *string
	PoolerMemoryRequest *string
	PoolerMemoryLimit   *string
}

// databaseAPI is the engine-independent surface used by the dedicated database
// resource, data source and backup policy resource.
type databaseAPI interface {
	Create(databaseID, name string, opts CreateOptions) (*models.DedicatedDatabase, error)
	Get(databaseID string) (*models.DedicatedDatabase, error)
	Update(databaseID string, opts UpdateOptions) (*models.DedicatedDatabase, error)
	Delete(databaseID string) error
	UpdateMaintenance(databaseID, day string, hourUTC int) (*models.DedicatedDatabase, error)
	CreateUpgrade(databaseID, targetVersion string) (*models.DedicatedDatabase, error)
	ListSpecifications() (*models.DedicatedDatabaseSpecificationList, error)

	CreateBackupPolicy(databaseID, policyID, name, schedule string, retention int, opts CreateBackupPolicyOptions) (*models.BackupPolicy, error)
	GetBackupPolicy(databaseID, policyID string) (*models.BackupPolicy, error)
	UpdateBackupPolicy(databaseID, policyID string, opts UpdateBackupPolicyOptions) (*models.BackupPolicy, error)
	DeleteBackupPolicy(databaseID, policyID string) error
}

// poolerAPI is implemented by the engines that front connections with a
// pooler. MongoDB does not, so it deliberately does not satisfy this.
type poolerAPI interface {
	GetPooler(databaseID string) (*models.DedicatedDatabasePooler, error)
	UpdatePooler(databaseID string, opts PoolerOptions) (*models.DedicatedDatabasePooler, error)
}

// extensionAPI is implemented by PostgreSQL only.
type extensionAPI interface {
	ListExtensions(databaseID string) (*models.DedicatedDatabaseExtensions, error)
	CreateExtension(databaseID, name string) (*models.DedicatedDatabase, error)
	DeleteExtension(databaseID, extensionName string) (*models.DedicatedDatabase, error)
}

// newDatabaseAPI returns the adapter for an engine, bound to a project client.
func newDatabaseAPI(engine Engine, clt client.Client) databaseAPI {
	switch engine {
	case EngineMysql:
		return mysqlAPI{srv: appwrite.NewMysql(clt)}
	case EngineMongo:
		return mongoAPI{srv: appwrite.NewMongo(clt)}
	default:
		return postgresqlAPI{srv: appwrite.NewPostgresql(clt)}
	}
}

// newPoolerAPI returns the pooler adapter for an engine, or false when the
// engine has no pooler.
func newPoolerAPI(engine Engine, clt client.Client) (poolerAPI, bool) {
	switch engine {
	case EnginePostgresql:
		return postgresqlAPI{srv: appwrite.NewPostgresql(clt)}, true
	case EngineMysql:
		return mysqlAPI{srv: appwrite.NewMysql(clt)}, true
	default:
		return nil, false
	}
}

// newExtensionAPI returns the extension adapter for an engine, or false when
// the engine has no extension support.
func newExtensionAPI(engine Engine, clt client.Client) (extensionAPI, bool) {
	if engine != EnginePostgresql {
		return nil, false
	}
	return postgresqlAPI{srv: appwrite.NewPostgresql(clt)}, true
}

// ---------------------------------------------------------------------------
// PostgreSQL
// ---------------------------------------------------------------------------

type postgresqlAPI struct{ srv *postgresql.Postgresql }

func (a postgresqlAPI) Create(databaseID, name string, o CreateOptions) (*models.DedicatedDatabase, error) {
	var opts []postgresql.CreateOption
	if o.Version != nil {
		opts = append(opts, a.srv.WithCreateVersion(*o.Version))
	}
	if o.Specification != nil {
		opts = append(opts, a.srv.WithCreateSpecification(*o.Specification))
	}
	if o.Replicas != nil {
		opts = append(opts, a.srv.WithCreateReplicas(*o.Replicas))
	}
	if o.SyncMode != nil {
		opts = append(opts, a.srv.WithCreateSyncMode(*o.SyncMode))
	}
	if o.NetworkIdleTimeoutSeconds != nil {
		opts = append(opts, a.srv.WithCreateNetworkIdleTimeoutSeconds(*o.NetworkIdleTimeoutSeconds))
	}
	if o.NetworkIPAllowlist != nil {
		opts = append(opts, a.srv.WithCreateNetworkIPAllowlist(o.NetworkIPAllowlist))
	}
	if o.IdleTimeoutMinutes != nil {
		opts = append(opts, a.srv.WithCreateIdleTimeoutMinutes(*o.IdleTimeoutMinutes))
	}
	if o.Pitr != nil {
		opts = append(opts, a.srv.WithCreatePitr(*o.Pitr))
	}
	if o.PitrRetentionDays != nil {
		opts = append(opts, a.srv.WithCreatePitrRetentionDays(*o.PitrRetentionDays))
	}
	if o.StorageAutoscaling != nil {
		opts = append(opts, a.srv.WithCreateStorageAutoscaling(*o.StorageAutoscaling))
	}
	if o.StorageAutoscalingThresholdPercent != nil {
		opts = append(opts, a.srv.WithCreateStorageAutoscalingThresholdPercent(*o.StorageAutoscalingThresholdPercent))
	}
	if o.StorageAutoscalingMaxGb != nil {
		opts = append(opts, a.srv.WithCreateStorageAutoscalingMaxGb(*o.StorageAutoscalingMaxGb))
	}
	return a.srv.Create(databaseID, name, opts...)
}

func (a postgresqlAPI) Get(databaseID string) (*models.DedicatedDatabase, error) {
	return a.srv.Get(databaseID)
}

func (a postgresqlAPI) Update(databaseID string, o UpdateOptions) (*models.DedicatedDatabase, error) {
	var opts []postgresql.UpdateOption
	if o.Name != nil {
		opts = append(opts, a.srv.WithUpdateName(*o.Name))
	}
	if o.Status != nil {
		opts = append(opts, a.srv.WithUpdateStatus(*o.Status))
	}
	if o.Specification != nil {
		opts = append(opts, a.srv.WithUpdateSpecification(*o.Specification))
	}
	if o.Replicas != nil {
		opts = append(opts, a.srv.WithUpdateReplicas(*o.Replicas))
	}
	if o.SyncMode != nil {
		opts = append(opts, a.srv.WithUpdateSyncMode(*o.SyncMode))
	}
	if o.NetworkIdleTimeoutSeconds != nil {
		opts = append(opts, a.srv.WithUpdateNetworkIdleTimeoutSeconds(*o.NetworkIdleTimeoutSeconds))
	}
	if o.NetworkIPAllowlist != nil {
		opts = append(opts, a.srv.WithUpdateNetworkIPAllowlist(o.NetworkIPAllowlist))
	}
	if o.IdleTimeoutMinutes != nil {
		opts = append(opts, a.srv.WithUpdateIdleTimeoutMinutes(*o.IdleTimeoutMinutes))
	}
	if o.Pitr != nil {
		opts = append(opts, a.srv.WithUpdatePitr(*o.Pitr))
	}
	if o.PitrRetentionDays != nil {
		opts = append(opts, a.srv.WithUpdatePitrRetentionDays(*o.PitrRetentionDays))
	}
	if o.StorageAutoscaling != nil {
		opts = append(opts, a.srv.WithUpdateStorageAutoscaling(*o.StorageAutoscaling))
	}
	if o.StorageAutoscalingThresholdPercent != nil {
		opts = append(opts, a.srv.WithUpdateStorageAutoscalingThresholdPercent(*o.StorageAutoscalingThresholdPercent))
	}
	if o.StorageAutoscalingMaxGb != nil {
		opts = append(opts, a.srv.WithUpdateStorageAutoscalingMaxGb(*o.StorageAutoscalingMaxGb))
	}
	if o.SQLAPIEnabled != nil {
		opts = append(opts, a.srv.WithUpdateSqlApiEnabled(*o.SQLAPIEnabled))
	}
	if o.SQLAPIAllowedStatements != nil {
		opts = append(opts, a.srv.WithUpdateSqlApiAllowedStatements(o.SQLAPIAllowedStatements))
	}
	if o.SQLAPIMaxRows != nil {
		opts = append(opts, a.srv.WithUpdateSqlApiMaxRows(*o.SQLAPIMaxRows))
	}
	if o.SQLAPIMaxBytes != nil {
		opts = append(opts, a.srv.WithUpdateSqlApiMaxBytes(*o.SQLAPIMaxBytes))
	}
	if o.SQLAPITimeoutSeconds != nil {
		opts = append(opts, a.srv.WithUpdateSqlApiTimeoutSeconds(*o.SQLAPITimeoutSeconds))
	}
	return a.srv.Update(databaseID, opts...)
}

func (a postgresqlAPI) Delete(databaseID string) error {
	_, err := a.srv.Delete(databaseID)
	return err
}

func (a postgresqlAPI) UpdateMaintenance(databaseID, day string, hourUTC int) (*models.DedicatedDatabase, error) {
	return a.srv.UpdateMaintenance(databaseID, day, hourUTC)
}

func (a postgresqlAPI) CreateUpgrade(databaseID, targetVersion string) (*models.DedicatedDatabase, error) {
	return a.srv.CreateUpgrade(databaseID, targetVersion)
}

func (a postgresqlAPI) ListSpecifications() (*models.DedicatedDatabaseSpecificationList, error) {
	return a.srv.ListSpecifications()
}

func (a postgresqlAPI) CreateBackupPolicy(databaseID, policyID, name, schedule string, retention int, o CreateBackupPolicyOptions) (*models.BackupPolicy, error) {
	var opts []postgresql.CreateBackupPolicyOption
	if o.Type != nil {
		opts = append(opts, a.srv.WithCreateBackupPolicyType(*o.Type))
	}
	if o.Enabled != nil {
		opts = append(opts, a.srv.WithCreateBackupPolicyEnabled(*o.Enabled))
	}
	return a.srv.CreateBackupPolicy(databaseID, policyID, name, schedule, retention, opts...)
}

func (a postgresqlAPI) GetBackupPolicy(databaseID, policyID string) (*models.BackupPolicy, error) {
	return a.srv.GetBackupPolicy(databaseID, policyID)
}

func (a postgresqlAPI) UpdateBackupPolicy(databaseID, policyID string, o UpdateBackupPolicyOptions) (*models.BackupPolicy, error) {
	var opts []postgresql.UpdateBackupPolicyOption
	if o.Name != nil {
		opts = append(opts, a.srv.WithUpdateBackupPolicyName(*o.Name))
	}
	if o.Schedule != nil {
		opts = append(opts, a.srv.WithUpdateBackupPolicySchedule(*o.Schedule))
	}
	if o.Retention != nil {
		opts = append(opts, a.srv.WithUpdateBackupPolicyRetention(*o.Retention))
	}
	if o.Enabled != nil {
		opts = append(opts, a.srv.WithUpdateBackupPolicyEnabled(*o.Enabled))
	}
	return a.srv.UpdateBackupPolicy(databaseID, policyID, opts...)
}

func (a postgresqlAPI) DeleteBackupPolicy(databaseID, policyID string) error {
	_, err := a.srv.DeleteBackupPolicy(databaseID, policyID)
	return err
}

func (a postgresqlAPI) GetPooler(databaseID string) (*models.DedicatedDatabasePooler, error) {
	return a.srv.GetPooler(databaseID)
}

func (a postgresqlAPI) UpdatePooler(databaseID string, o PoolerOptions) (*models.DedicatedDatabasePooler, error) {
	var opts []postgresql.UpdatePoolerOption
	if o.Mode != nil {
		opts = append(opts, a.srv.WithUpdatePoolerMode(*o.Mode))
	}
	if o.MaxConnections != nil {
		opts = append(opts, a.srv.WithUpdatePoolerMaxConnections(*o.MaxConnections))
	}
	if o.DefaultPoolSize != nil {
		opts = append(opts, a.srv.WithUpdatePoolerDefaultPoolSize(*o.DefaultPoolSize))
	}
	if o.ReadWriteSplitting != nil {
		opts = append(opts, a.srv.WithUpdatePoolerReadWriteSplitting(*o.ReadWriteSplitting))
	}
	if o.PoolerCPURequest != nil {
		opts = append(opts, a.srv.WithUpdatePoolerPoolerCpuRequest(*o.PoolerCPURequest))
	}
	if o.PoolerCPULimit != nil {
		opts = append(opts, a.srv.WithUpdatePoolerPoolerCpuLimit(*o.PoolerCPULimit))
	}
	if o.PoolerMemoryRequest != nil {
		opts = append(opts, a.srv.WithUpdatePoolerPoolerMemoryRequest(*o.PoolerMemoryRequest))
	}
	if o.PoolerMemoryLimit != nil {
		opts = append(opts, a.srv.WithUpdatePoolerPoolerMemoryLimit(*o.PoolerMemoryLimit))
	}
	return a.srv.UpdatePooler(databaseID, opts...)
}

func (a postgresqlAPI) ListExtensions(databaseID string) (*models.DedicatedDatabaseExtensions, error) {
	return a.srv.ListExtensions(databaseID)
}

func (a postgresqlAPI) CreateExtension(databaseID, name string) (*models.DedicatedDatabase, error) {
	return a.srv.CreateExtension(databaseID, name)
}

func (a postgresqlAPI) DeleteExtension(databaseID, extensionName string) (*models.DedicatedDatabase, error) {
	return a.srv.DeleteExtension(databaseID, extensionName)
}

// ---------------------------------------------------------------------------
// MySQL
// ---------------------------------------------------------------------------

type mysqlAPI struct{ srv *mysql.Mysql }

func (a mysqlAPI) Create(databaseID, name string, o CreateOptions) (*models.DedicatedDatabase, error) {
	var opts []mysql.CreateOption
	if o.Version != nil {
		opts = append(opts, a.srv.WithCreateVersion(*o.Version))
	}
	if o.Specification != nil {
		opts = append(opts, a.srv.WithCreateSpecification(*o.Specification))
	}
	if o.Replicas != nil {
		opts = append(opts, a.srv.WithCreateReplicas(*o.Replicas))
	}
	if o.SyncMode != nil {
		opts = append(opts, a.srv.WithCreateSyncMode(*o.SyncMode))
	}
	if o.NetworkIdleTimeoutSeconds != nil {
		opts = append(opts, a.srv.WithCreateNetworkIdleTimeoutSeconds(*o.NetworkIdleTimeoutSeconds))
	}
	if o.NetworkIPAllowlist != nil {
		opts = append(opts, a.srv.WithCreateNetworkIPAllowlist(o.NetworkIPAllowlist))
	}
	if o.IdleTimeoutMinutes != nil {
		opts = append(opts, a.srv.WithCreateIdleTimeoutMinutes(*o.IdleTimeoutMinutes))
	}
	if o.Pitr != nil {
		opts = append(opts, a.srv.WithCreatePitr(*o.Pitr))
	}
	if o.PitrRetentionDays != nil {
		opts = append(opts, a.srv.WithCreatePitrRetentionDays(*o.PitrRetentionDays))
	}
	if o.StorageAutoscaling != nil {
		opts = append(opts, a.srv.WithCreateStorageAutoscaling(*o.StorageAutoscaling))
	}
	if o.StorageAutoscalingThresholdPercent != nil {
		opts = append(opts, a.srv.WithCreateStorageAutoscalingThresholdPercent(*o.StorageAutoscalingThresholdPercent))
	}
	if o.StorageAutoscalingMaxGb != nil {
		opts = append(opts, a.srv.WithCreateStorageAutoscalingMaxGb(*o.StorageAutoscalingMaxGb))
	}
	return a.srv.Create(databaseID, name, opts...)
}

func (a mysqlAPI) Get(databaseID string) (*models.DedicatedDatabase, error) {
	return a.srv.Get(databaseID)
}

func (a mysqlAPI) Update(databaseID string, o UpdateOptions) (*models.DedicatedDatabase, error) {
	var opts []mysql.UpdateOption
	if o.Name != nil {
		opts = append(opts, a.srv.WithUpdateName(*o.Name))
	}
	if o.Status != nil {
		opts = append(opts, a.srv.WithUpdateStatus(*o.Status))
	}
	if o.Specification != nil {
		opts = append(opts, a.srv.WithUpdateSpecification(*o.Specification))
	}
	if o.Replicas != nil {
		opts = append(opts, a.srv.WithUpdateReplicas(*o.Replicas))
	}
	if o.SyncMode != nil {
		opts = append(opts, a.srv.WithUpdateSyncMode(*o.SyncMode))
	}
	if o.NetworkIdleTimeoutSeconds != nil {
		opts = append(opts, a.srv.WithUpdateNetworkIdleTimeoutSeconds(*o.NetworkIdleTimeoutSeconds))
	}
	if o.NetworkIPAllowlist != nil {
		opts = append(opts, a.srv.WithUpdateNetworkIPAllowlist(o.NetworkIPAllowlist))
	}
	if o.IdleTimeoutMinutes != nil {
		opts = append(opts, a.srv.WithUpdateIdleTimeoutMinutes(*o.IdleTimeoutMinutes))
	}
	if o.Pitr != nil {
		opts = append(opts, a.srv.WithUpdatePitr(*o.Pitr))
	}
	if o.PitrRetentionDays != nil {
		opts = append(opts, a.srv.WithUpdatePitrRetentionDays(*o.PitrRetentionDays))
	}
	if o.StorageAutoscaling != nil {
		opts = append(opts, a.srv.WithUpdateStorageAutoscaling(*o.StorageAutoscaling))
	}
	if o.StorageAutoscalingThresholdPercent != nil {
		opts = append(opts, a.srv.WithUpdateStorageAutoscalingThresholdPercent(*o.StorageAutoscalingThresholdPercent))
	}
	if o.StorageAutoscalingMaxGb != nil {
		opts = append(opts, a.srv.WithUpdateStorageAutoscalingMaxGb(*o.StorageAutoscalingMaxGb))
	}
	if o.SQLAPIEnabled != nil {
		opts = append(opts, a.srv.WithUpdateSqlApiEnabled(*o.SQLAPIEnabled))
	}
	if o.SQLAPIAllowedStatements != nil {
		opts = append(opts, a.srv.WithUpdateSqlApiAllowedStatements(o.SQLAPIAllowedStatements))
	}
	if o.SQLAPIMaxRows != nil {
		opts = append(opts, a.srv.WithUpdateSqlApiMaxRows(*o.SQLAPIMaxRows))
	}
	if o.SQLAPIMaxBytes != nil {
		opts = append(opts, a.srv.WithUpdateSqlApiMaxBytes(*o.SQLAPIMaxBytes))
	}
	if o.SQLAPITimeoutSeconds != nil {
		opts = append(opts, a.srv.WithUpdateSqlApiTimeoutSeconds(*o.SQLAPITimeoutSeconds))
	}
	return a.srv.Update(databaseID, opts...)
}

func (a mysqlAPI) Delete(databaseID string) error {
	_, err := a.srv.Delete(databaseID)
	return err
}

func (a mysqlAPI) UpdateMaintenance(databaseID, day string, hourUTC int) (*models.DedicatedDatabase, error) {
	return a.srv.UpdateMaintenance(databaseID, day, hourUTC)
}

func (a mysqlAPI) CreateUpgrade(databaseID, targetVersion string) (*models.DedicatedDatabase, error) {
	return a.srv.CreateUpgrade(databaseID, targetVersion)
}

func (a mysqlAPI) ListSpecifications() (*models.DedicatedDatabaseSpecificationList, error) {
	return a.srv.ListSpecifications()
}

func (a mysqlAPI) CreateBackupPolicy(databaseID, policyID, name, schedule string, retention int, o CreateBackupPolicyOptions) (*models.BackupPolicy, error) {
	var opts []mysql.CreateBackupPolicyOption
	if o.Type != nil {
		opts = append(opts, a.srv.WithCreateBackupPolicyType(*o.Type))
	}
	if o.Enabled != nil {
		opts = append(opts, a.srv.WithCreateBackupPolicyEnabled(*o.Enabled))
	}
	return a.srv.CreateBackupPolicy(databaseID, policyID, name, schedule, retention, opts...)
}

func (a mysqlAPI) GetBackupPolicy(databaseID, policyID string) (*models.BackupPolicy, error) {
	return a.srv.GetBackupPolicy(databaseID, policyID)
}

func (a mysqlAPI) UpdateBackupPolicy(databaseID, policyID string, o UpdateBackupPolicyOptions) (*models.BackupPolicy, error) {
	var opts []mysql.UpdateBackupPolicyOption
	if o.Name != nil {
		opts = append(opts, a.srv.WithUpdateBackupPolicyName(*o.Name))
	}
	if o.Schedule != nil {
		opts = append(opts, a.srv.WithUpdateBackupPolicySchedule(*o.Schedule))
	}
	if o.Retention != nil {
		opts = append(opts, a.srv.WithUpdateBackupPolicyRetention(*o.Retention))
	}
	if o.Enabled != nil {
		opts = append(opts, a.srv.WithUpdateBackupPolicyEnabled(*o.Enabled))
	}
	return a.srv.UpdateBackupPolicy(databaseID, policyID, opts...)
}

func (a mysqlAPI) DeleteBackupPolicy(databaseID, policyID string) error {
	_, err := a.srv.DeleteBackupPolicy(databaseID, policyID)
	return err
}

func (a mysqlAPI) GetPooler(databaseID string) (*models.DedicatedDatabasePooler, error) {
	return a.srv.GetPooler(databaseID)
}

func (a mysqlAPI) UpdatePooler(databaseID string, o PoolerOptions) (*models.DedicatedDatabasePooler, error) {
	var opts []mysql.UpdatePoolerOption
	if o.Mode != nil {
		opts = append(opts, a.srv.WithUpdatePoolerMode(*o.Mode))
	}
	if o.MaxConnections != nil {
		opts = append(opts, a.srv.WithUpdatePoolerMaxConnections(*o.MaxConnections))
	}
	if o.DefaultPoolSize != nil {
		opts = append(opts, a.srv.WithUpdatePoolerDefaultPoolSize(*o.DefaultPoolSize))
	}
	if o.ReadWriteSplitting != nil {
		opts = append(opts, a.srv.WithUpdatePoolerReadWriteSplitting(*o.ReadWriteSplitting))
	}
	if o.PoolerCPURequest != nil {
		opts = append(opts, a.srv.WithUpdatePoolerPoolerCpuRequest(*o.PoolerCPURequest))
	}
	if o.PoolerCPULimit != nil {
		opts = append(opts, a.srv.WithUpdatePoolerPoolerCpuLimit(*o.PoolerCPULimit))
	}
	if o.PoolerMemoryRequest != nil {
		opts = append(opts, a.srv.WithUpdatePoolerPoolerMemoryRequest(*o.PoolerMemoryRequest))
	}
	if o.PoolerMemoryLimit != nil {
		opts = append(opts, a.srv.WithUpdatePoolerPoolerMemoryLimit(*o.PoolerMemoryLimit))
	}
	return a.srv.UpdatePooler(databaseID, opts...)
}

// ---------------------------------------------------------------------------
// MongoDB
// ---------------------------------------------------------------------------

type mongoAPI struct{ srv *mongo.Mongo }

func (a mongoAPI) Create(databaseID, name string, o CreateOptions) (*models.DedicatedDatabase, error) {
	var opts []mongo.CreateOption
	if o.Version != nil {
		opts = append(opts, a.srv.WithCreateVersion(*o.Version))
	}
	if o.Specification != nil {
		opts = append(opts, a.srv.WithCreateSpecification(*o.Specification))
	}
	if o.Replicas != nil {
		opts = append(opts, a.srv.WithCreateReplicas(*o.Replicas))
	}
	if o.SyncMode != nil {
		opts = append(opts, a.srv.WithCreateSyncMode(*o.SyncMode))
	}
	if o.NetworkIdleTimeoutSeconds != nil {
		opts = append(opts, a.srv.WithCreateNetworkIdleTimeoutSeconds(*o.NetworkIdleTimeoutSeconds))
	}
	if o.NetworkIPAllowlist != nil {
		opts = append(opts, a.srv.WithCreateNetworkIPAllowlist(o.NetworkIPAllowlist))
	}
	if o.IdleTimeoutMinutes != nil {
		opts = append(opts, a.srv.WithCreateIdleTimeoutMinutes(*o.IdleTimeoutMinutes))
	}
	if o.Pitr != nil {
		opts = append(opts, a.srv.WithCreatePitr(*o.Pitr))
	}
	if o.PitrRetentionDays != nil {
		opts = append(opts, a.srv.WithCreatePitrRetentionDays(*o.PitrRetentionDays))
	}
	if o.StorageAutoscaling != nil {
		opts = append(opts, a.srv.WithCreateStorageAutoscaling(*o.StorageAutoscaling))
	}
	if o.StorageAutoscalingThresholdPercent != nil {
		opts = append(opts, a.srv.WithCreateStorageAutoscalingThresholdPercent(*o.StorageAutoscalingThresholdPercent))
	}
	if o.StorageAutoscalingMaxGb != nil {
		opts = append(opts, a.srv.WithCreateStorageAutoscalingMaxGb(*o.StorageAutoscalingMaxGb))
	}
	return a.srv.Create(databaseID, name, opts...)
}

func (a mongoAPI) Get(databaseID string) (*models.DedicatedDatabase, error) {
	return a.srv.Get(databaseID)
}

func (a mongoAPI) Update(databaseID string, o UpdateOptions) (*models.DedicatedDatabase, error) {
	var opts []mongo.UpdateOption
	if o.Name != nil {
		opts = append(opts, a.srv.WithUpdateName(*o.Name))
	}
	if o.Status != nil {
		opts = append(opts, a.srv.WithUpdateStatus(*o.Status))
	}
	if o.Specification != nil {
		opts = append(opts, a.srv.WithUpdateSpecification(*o.Specification))
	}
	if o.Replicas != nil {
		opts = append(opts, a.srv.WithUpdateReplicas(*o.Replicas))
	}
	if o.SyncMode != nil {
		opts = append(opts, a.srv.WithUpdateSyncMode(*o.SyncMode))
	}
	if o.NetworkIdleTimeoutSeconds != nil {
		opts = append(opts, a.srv.WithUpdateNetworkIdleTimeoutSeconds(*o.NetworkIdleTimeoutSeconds))
	}
	if o.NetworkIPAllowlist != nil {
		opts = append(opts, a.srv.WithUpdateNetworkIPAllowlist(o.NetworkIPAllowlist))
	}
	if o.IdleTimeoutMinutes != nil {
		opts = append(opts, a.srv.WithUpdateIdleTimeoutMinutes(*o.IdleTimeoutMinutes))
	}
	if o.Pitr != nil {
		opts = append(opts, a.srv.WithUpdatePitr(*o.Pitr))
	}
	if o.PitrRetentionDays != nil {
		opts = append(opts, a.srv.WithUpdatePitrRetentionDays(*o.PitrRetentionDays))
	}
	if o.StorageAutoscaling != nil {
		opts = append(opts, a.srv.WithUpdateStorageAutoscaling(*o.StorageAutoscaling))
	}
	if o.StorageAutoscalingThresholdPercent != nil {
		opts = append(opts, a.srv.WithUpdateStorageAutoscalingThresholdPercent(*o.StorageAutoscalingThresholdPercent))
	}
	if o.StorageAutoscalingMaxGb != nil {
		opts = append(opts, a.srv.WithUpdateStorageAutoscalingMaxGb(*o.StorageAutoscalingMaxGb))
	}
	if o.SQLAPIEnabled != nil {
		opts = append(opts, a.srv.WithUpdateSqlApiEnabled(*o.SQLAPIEnabled))
	}
	if o.SQLAPIAllowedStatements != nil {
		opts = append(opts, a.srv.WithUpdateSqlApiAllowedStatements(o.SQLAPIAllowedStatements))
	}
	if o.SQLAPIMaxRows != nil {
		opts = append(opts, a.srv.WithUpdateSqlApiMaxRows(*o.SQLAPIMaxRows))
	}
	if o.SQLAPIMaxBytes != nil {
		opts = append(opts, a.srv.WithUpdateSqlApiMaxBytes(*o.SQLAPIMaxBytes))
	}
	if o.SQLAPITimeoutSeconds != nil {
		opts = append(opts, a.srv.WithUpdateSqlApiTimeoutSeconds(*o.SQLAPITimeoutSeconds))
	}
	return a.srv.Update(databaseID, opts...)
}

func (a mongoAPI) Delete(databaseID string) error {
	_, err := a.srv.Delete(databaseID)
	return err
}

func (a mongoAPI) UpdateMaintenance(databaseID, day string, hourUTC int) (*models.DedicatedDatabase, error) {
	return a.srv.UpdateMaintenance(databaseID, day, hourUTC)
}

func (a mongoAPI) CreateUpgrade(databaseID, targetVersion string) (*models.DedicatedDatabase, error) {
	return a.srv.CreateUpgrade(databaseID, targetVersion)
}

func (a mongoAPI) ListSpecifications() (*models.DedicatedDatabaseSpecificationList, error) {
	return a.srv.ListSpecifications()
}

func (a mongoAPI) CreateBackupPolicy(databaseID, policyID, name, schedule string, retention int, o CreateBackupPolicyOptions) (*models.BackupPolicy, error) {
	var opts []mongo.CreateBackupPolicyOption
	if o.Type != nil {
		opts = append(opts, a.srv.WithCreateBackupPolicyType(*o.Type))
	}
	if o.Enabled != nil {
		opts = append(opts, a.srv.WithCreateBackupPolicyEnabled(*o.Enabled))
	}
	return a.srv.CreateBackupPolicy(databaseID, policyID, name, schedule, retention, opts...)
}

func (a mongoAPI) GetBackupPolicy(databaseID, policyID string) (*models.BackupPolicy, error) {
	return a.srv.GetBackupPolicy(databaseID, policyID)
}

func (a mongoAPI) UpdateBackupPolicy(databaseID, policyID string, o UpdateBackupPolicyOptions) (*models.BackupPolicy, error) {
	var opts []mongo.UpdateBackupPolicyOption
	if o.Name != nil {
		opts = append(opts, a.srv.WithUpdateBackupPolicyName(*o.Name))
	}
	if o.Schedule != nil {
		opts = append(opts, a.srv.WithUpdateBackupPolicySchedule(*o.Schedule))
	}
	if o.Retention != nil {
		opts = append(opts, a.srv.WithUpdateBackupPolicyRetention(*o.Retention))
	}
	if o.Enabled != nil {
		opts = append(opts, a.srv.WithUpdateBackupPolicyEnabled(*o.Enabled))
	}
	return a.srv.UpdateBackupPolicy(databaseID, policyID, opts...)
}

func (a mongoAPI) DeleteBackupPolicy(databaseID, policyID string) error {
	_, err := a.srv.DeleteBackupPolicy(databaseID, policyID)
	return err
}

// Compile-time proof that each adapter covers the surface its engine supports.
var (
	_ databaseAPI  = postgresqlAPI{}
	_ poolerAPI    = postgresqlAPI{}
	_ extensionAPI = postgresqlAPI{}
	_ databaseAPI  = mysqlAPI{}
	_ poolerAPI    = mysqlAPI{}
	_ databaseAPI  = mongoAPI{}
)
