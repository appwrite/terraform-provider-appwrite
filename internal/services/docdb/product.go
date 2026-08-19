// Package docdb implements the Appwrite DocumentsDB and VectorsDB resources.
//
// The two products are separate route sets (/documentsdb and /vectorsdb) with
// separate SDK services, but their databases, collections, indexes and
// documents are the same shape. The only real differences are that a VectorsDB
// collection carries a required embedding dimension, and that a DocumentsDB
// collection accepts inline attributes and indexes at creation. The interfaces
// below adapt both onto one shape so each resource is written once and
// registered twice, in the same way internal/services/dedicated handles the
// three dedicated database engines.
package docdb

import (
	"github.com/appwrite/sdk-for-go/v7/appwrite"
	"github.com/appwrite/sdk-for-go/v7/documentsdb"
	"github.com/appwrite/sdk-for-go/v7/models"
	"github.com/appwrite/sdk-for-go/v7/vectorsdb"
	"github.com/appwrite/terraform-provider-appwrite/internal/common"
)

// Product identifies which of the two APIs a resource talks to. The value
// doubles as the Terraform resource name segment.
type Product string

const (
	ProductDocumentsDB Product = "documentsdb"
	ProductVectorsDB   Product = "vectorsdb"
)

// Label returns the product name as it reads in documentation.
func (p Product) Label() string {
	switch p {
	case ProductVectorsDB:
		return "VectorsDB"
	default:
		return "DocumentsDB"
	}
}

// DatabaseOptions holds the arguments accepted when creating or updating a
// database. A nil pointer means the attribute was not configured, so the option
// is omitted and the server default applies.
type DatabaseOptions struct {
	Enabled       *bool
	Specification *string
	Replicas      *int
	SyncMode      *string
}

// CollectionOptions holds the arguments accepted when creating or updating a
// collection. Dimension applies to VectorsDB only, and Purge to DocumentsDB
// only; each product's adapter ignores the field it has no route for.
type CollectionOptions struct {
	Permissions      []string
	DocumentSecurity *bool
	Enabled          *bool
	Dimension        *int
	Purge            *bool

	// Attributes are accepted only when the collection is created, and only by
	// DocumentsDB. There is no attribute route to add them afterwards.
	Attributes []interface{}
}

// IndexOptions holds the optional arguments for a new index.
type IndexOptions struct {
	Lengths []int
	Orders  []string
}

// Collection is the product-independent view of a collection. VectorsDB returns
// a distinct model that adds Dimension, so the two are normalised here rather
// than leaking the difference into the resource.
type Collection struct {
	ID               string
	DatabaseID       string
	Name             string
	Enabled          bool
	DocumentSecurity bool
	Permissions      []string
	Dimension        int
	CreatedAt        string
	UpdatedAt        string
}

// productAPI is the surface shared by both services.
type productAPI interface {
	ListSpecifications() (*models.DedicatedDatabaseSpecificationList, error)
	Create(databaseID, name string, opts DatabaseOptions) (*models.Database, error)
	Get(databaseID string) (*models.Database, error)
	Update(databaseID, name string, opts DatabaseOptions) (*models.Database, error)
	Delete(databaseID string) error

	CreateCollection(databaseID, collectionID, name string, opts CollectionOptions) (*Collection, error)
	GetCollection(databaseID, collectionID string) (*Collection, error)
	UpdateCollection(databaseID, collectionID, name string, opts CollectionOptions) (*Collection, error)
	DeleteCollection(databaseID, collectionID string) error

	CreateIndex(databaseID, collectionID, key, indexType string, attributes []string, opts IndexOptions) (*models.Index, error)
	GetIndex(databaseID, collectionID, key string) (*models.Index, error)
	DeleteIndex(databaseID, collectionID, key string) error

	CreateDocument(databaseID, collectionID, documentID string, data interface{}, permissions []string) (*models.Document, error)
	GetDocument(databaseID, collectionID, documentID string) (*models.Document, error)
	UpdateDocument(databaseID, collectionID, documentID string, data interface{}, permissions []string) (*models.Document, error)
	DeleteDocument(databaseID, collectionID, documentID string) error
}

// apiFor returns the adapter for a product, bound to a project client.
func apiFor(clients *common.AppwriteClients, product Product, projectID string) productAPI {
	clt := clients.ClientForProject(projectID)
	if product == ProductVectorsDB {
		return vectorsAPI{srv: appwrite.NewVectorsDB(clt)}
	}
	return documentsAPI{srv: appwrite.NewDocumentsDB(clt)}
}

// SupportsAttributes reports whether a product's collections accept typed
// attribute definitions. Only DocumentsDB does, and only at creation time.
func (p Product) SupportsAttributes() bool {
	return p == ProductDocumentsDB
}

// SupportsDimension reports whether a product's collections carry an embedding
// dimension. Only VectorsDB does, so the attribute is required there and absent
// everywhere else.
func (p Product) SupportsDimension() bool {
	return p == ProductVectorsDB
}

// ---------------------------------------------------------------------------
// DocumentsDB
// ---------------------------------------------------------------------------

type documentsAPI struct{ srv *documentsdb.DocumentsDB }

func (a documentsAPI) Create(databaseID, name string, o DatabaseOptions) (*models.Database, error) {
	var opts []documentsdb.CreateOption
	if o.Enabled != nil {
		opts = append(opts, a.srv.WithCreateEnabled(*o.Enabled))
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
	return a.srv.Create(databaseID, name, opts...)
}

func (a documentsAPI) ListSpecifications() (*models.DedicatedDatabaseSpecificationList, error) {
	return a.srv.ListSpecifications()
}

func (a documentsAPI) Get(databaseID string) (*models.Database, error) {
	return a.srv.Get(databaseID)
}

func (a documentsAPI) Update(databaseID, name string, o DatabaseOptions) (*models.Database, error) {
	var opts []documentsdb.UpdateOption
	if o.Enabled != nil {
		opts = append(opts, a.srv.WithUpdateEnabled(*o.Enabled))
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
	return a.srv.Update(databaseID, name, opts...)
}

func (a documentsAPI) Delete(databaseID string) error {
	_, err := a.srv.Delete(databaseID)
	return err
}

func (a documentsAPI) CreateCollection(databaseID, collectionID, name string, o CollectionOptions) (*Collection, error) {
	var opts []documentsdb.CreateCollectionOption
	if o.Permissions != nil {
		opts = append(opts, a.srv.WithCreateCollectionPermissions(o.Permissions))
	}
	if o.DocumentSecurity != nil {
		opts = append(opts, a.srv.WithCreateCollectionDocumentSecurity(*o.DocumentSecurity))
	}
	if o.Enabled != nil {
		opts = append(opts, a.srv.WithCreateCollectionEnabled(*o.Enabled))
	}
	if o.Attributes != nil {
		opts = append(opts, a.srv.WithCreateCollectionAttributes(o.Attributes))
	}
	collection, err := a.srv.CreateCollection(databaseID, collectionID, name, opts...)
	if err != nil {
		return nil, err
	}
	return collectionFromDocuments(collection), nil
}

func (a documentsAPI) GetCollection(databaseID, collectionID string) (*Collection, error) {
	collection, err := a.srv.GetCollection(databaseID, collectionID)
	if err != nil {
		return nil, err
	}
	return collectionFromDocuments(collection), nil
}

func (a documentsAPI) UpdateCollection(databaseID, collectionID, name string, o CollectionOptions) (*Collection, error) {
	var opts []documentsdb.UpdateCollectionOption
	if o.Permissions != nil {
		opts = append(opts, a.srv.WithUpdateCollectionPermissions(o.Permissions))
	}
	if o.DocumentSecurity != nil {
		opts = append(opts, a.srv.WithUpdateCollectionDocumentSecurity(*o.DocumentSecurity))
	}
	if o.Enabled != nil {
		opts = append(opts, a.srv.WithUpdateCollectionEnabled(*o.Enabled))
	}
	collection, err := a.srv.UpdateCollection(databaseID, collectionID, name, opts...)
	if err != nil {
		return nil, err
	}
	return collectionFromDocuments(collection), nil
}

func (a documentsAPI) DeleteCollection(databaseID, collectionID string) error {
	_, err := a.srv.DeleteCollection(databaseID, collectionID)
	return err
}

func (a documentsAPI) CreateIndex(databaseID, collectionID, key, indexType string, attributes []string, o IndexOptions) (*models.Index, error) {
	var opts []documentsdb.CreateIndexOption
	if o.Lengths != nil {
		opts = append(opts, a.srv.WithCreateIndexLengths(o.Lengths))
	}
	if o.Orders != nil {
		opts = append(opts, a.srv.WithCreateIndexOrders(o.Orders))
	}
	return a.srv.CreateIndex(databaseID, collectionID, key, indexType, attributes, opts...)
}

func (a documentsAPI) GetIndex(databaseID, collectionID, key string) (*models.Index, error) {
	return a.srv.GetIndex(databaseID, collectionID, key)
}

func (a documentsAPI) DeleteIndex(databaseID, collectionID, key string) error {
	_, err := a.srv.DeleteIndex(databaseID, collectionID, key)
	return err
}

func (a documentsAPI) CreateDocument(databaseID, collectionID, documentID string, data interface{}, permissions []string) (*models.Document, error) {
	var opts []documentsdb.CreateDocumentOption
	if permissions != nil {
		opts = append(opts, a.srv.WithCreateDocumentPermissions(permissions))
	}
	return a.srv.CreateDocument(databaseID, collectionID, documentID, data, opts...)
}

func (a documentsAPI) GetDocument(databaseID, collectionID, documentID string) (*models.Document, error) {
	return a.srv.GetDocument(databaseID, collectionID, documentID)
}

func (a documentsAPI) UpdateDocument(databaseID, collectionID, documentID string, data interface{}, permissions []string) (*models.Document, error) {
	opts := []documentsdb.UpdateDocumentOption{a.srv.WithUpdateDocumentData(data)}
	if permissions != nil {
		opts = append(opts, a.srv.WithUpdateDocumentPermissions(permissions))
	}
	return a.srv.UpdateDocument(databaseID, collectionID, documentID, opts...)
}

func (a documentsAPI) DeleteDocument(databaseID, collectionID, documentID string) error {
	_, err := a.srv.DeleteDocument(databaseID, collectionID, documentID)
	return err
}

func collectionFromDocuments(c *models.Collection) *Collection {
	return &Collection{
		ID:               c.Id,
		DatabaseID:       c.DatabaseId,
		Name:             c.Name,
		Enabled:          c.Enabled,
		DocumentSecurity: c.DocumentSecurity,
		Permissions:      c.Permissions,
		CreatedAt:        c.CreatedAt,
		UpdatedAt:        c.UpdatedAt,
	}
}

// ---------------------------------------------------------------------------
// VectorsDB
// ---------------------------------------------------------------------------

type vectorsAPI struct{ srv *vectorsdb.VectorsDB }

func (a vectorsAPI) Create(databaseID, name string, o DatabaseOptions) (*models.Database, error) {
	var opts []vectorsdb.CreateOption
	if o.Enabled != nil {
		opts = append(opts, a.srv.WithCreateEnabled(*o.Enabled))
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
	return a.srv.Create(databaseID, name, opts...)
}

func (a vectorsAPI) ListSpecifications() (*models.DedicatedDatabaseSpecificationList, error) {
	return a.srv.ListSpecifications()
}

func (a vectorsAPI) Get(databaseID string) (*models.Database, error) {
	return a.srv.Get(databaseID)
}

func (a vectorsAPI) Update(databaseID, name string, o DatabaseOptions) (*models.Database, error) {
	var opts []vectorsdb.UpdateOption
	if o.Enabled != nil {
		opts = append(opts, a.srv.WithUpdateEnabled(*o.Enabled))
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
	return a.srv.Update(databaseID, name, opts...)
}

func (a vectorsAPI) Delete(databaseID string) error {
	_, err := a.srv.Delete(databaseID)
	return err
}

// CreateCollection requires the embedding dimension. It is enforced as a
// required attribute in the schema, so a nil here is a programming error rather
// than user input.
func (a vectorsAPI) CreateCollection(databaseID, collectionID, name string, o CollectionOptions) (*Collection, error) {
	var opts []vectorsdb.CreateCollectionOption
	if o.Permissions != nil {
		opts = append(opts, a.srv.WithCreateCollectionPermissions(o.Permissions))
	}
	if o.DocumentSecurity != nil {
		opts = append(opts, a.srv.WithCreateCollectionDocumentSecurity(*o.DocumentSecurity))
	}
	if o.Enabled != nil {
		opts = append(opts, a.srv.WithCreateCollectionEnabled(*o.Enabled))
	}

	dimension := 0
	if o.Dimension != nil {
		dimension = *o.Dimension
	}
	collection, err := a.srv.CreateCollection(databaseID, collectionID, name, dimension, opts...)
	if err != nil {
		return nil, err
	}
	return collectionFromVectors(collection), nil
}

func (a vectorsAPI) GetCollection(databaseID, collectionID string) (*Collection, error) {
	collection, err := a.srv.GetCollection(databaseID, collectionID)
	if err != nil {
		return nil, err
	}
	return collectionFromVectors(collection), nil
}

func (a vectorsAPI) UpdateCollection(databaseID, collectionID, name string, o CollectionOptions) (*Collection, error) {
	var opts []vectorsdb.UpdateCollectionOption
	if o.Permissions != nil {
		opts = append(opts, a.srv.WithUpdateCollectionPermissions(o.Permissions))
	}
	if o.DocumentSecurity != nil {
		opts = append(opts, a.srv.WithUpdateCollectionDocumentSecurity(*o.DocumentSecurity))
	}
	if o.Enabled != nil {
		opts = append(opts, a.srv.WithUpdateCollectionEnabled(*o.Enabled))
	}
	if o.Dimension != nil {
		opts = append(opts, a.srv.WithUpdateCollectionDimension(*o.Dimension))
	}
	collection, err := a.srv.UpdateCollection(databaseID, collectionID, name, opts...)
	if err != nil {
		return nil, err
	}
	return collectionFromVectors(collection), nil
}

func (a vectorsAPI) DeleteCollection(databaseID, collectionID string) error {
	_, err := a.srv.DeleteCollection(databaseID, collectionID)
	return err
}

func (a vectorsAPI) CreateIndex(databaseID, collectionID, key, indexType string, attributes []string, o IndexOptions) (*models.Index, error) {
	var opts []vectorsdb.CreateIndexOption
	if o.Lengths != nil {
		opts = append(opts, a.srv.WithCreateIndexLengths(o.Lengths))
	}
	if o.Orders != nil {
		opts = append(opts, a.srv.WithCreateIndexOrders(o.Orders))
	}
	return a.srv.CreateIndex(databaseID, collectionID, key, indexType, attributes, opts...)
}

func (a vectorsAPI) GetIndex(databaseID, collectionID, key string) (*models.Index, error) {
	return a.srv.GetIndex(databaseID, collectionID, key)
}

func (a vectorsAPI) DeleteIndex(databaseID, collectionID, key string) error {
	_, err := a.srv.DeleteIndex(databaseID, collectionID, key)
	return err
}

func (a vectorsAPI) CreateDocument(databaseID, collectionID, documentID string, data interface{}, permissions []string) (*models.Document, error) {
	var opts []vectorsdb.CreateDocumentOption
	if permissions != nil {
		opts = append(opts, a.srv.WithCreateDocumentPermissions(permissions))
	}
	return a.srv.CreateDocument(databaseID, collectionID, documentID, data, opts...)
}

func (a vectorsAPI) GetDocument(databaseID, collectionID, documentID string) (*models.Document, error) {
	return a.srv.GetDocument(databaseID, collectionID, documentID)
}

func (a vectorsAPI) UpdateDocument(databaseID, collectionID, documentID string, data interface{}, permissions []string) (*models.Document, error) {
	opts := []vectorsdb.UpdateDocumentOption{a.srv.WithUpdateDocumentData(data)}
	if permissions != nil {
		opts = append(opts, a.srv.WithUpdateDocumentPermissions(permissions))
	}
	return a.srv.UpdateDocument(databaseID, collectionID, documentID, opts...)
}

func (a vectorsAPI) DeleteDocument(databaseID, collectionID, documentID string) error {
	_, err := a.srv.DeleteDocument(databaseID, collectionID, documentID)
	return err
}

func collectionFromVectors(c *models.VectorsdbCollection) *Collection {
	return &Collection{
		ID:               c.Id,
		DatabaseID:       c.DatabaseId,
		Name:             c.Name,
		Enabled:          c.Enabled,
		DocumentSecurity: c.DocumentSecurity,
		Permissions:      c.Permissions,
		Dimension:        c.Dimension,
		CreatedAt:        c.CreatedAt,
		UpdatedAt:        c.UpdatedAt,
	}
}

// Compile-time proof that both adapters cover the shared surface.
var (
	_ productAPI = documentsAPI{}
	_ productAPI = vectorsAPI{}
)
