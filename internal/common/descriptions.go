package common

// Shared attribute descriptions.
//
// The same attribute described three different ways across three resources is
// how reference documentation stops being trustworthy: a reader cannot tell
// whether the wording differs because the behavior differs. These constants
// exist so that it cannot drift, and so that changing the wording is one edit
// rather than twenty.
//
// Resources reach these through ProjectIDAttribute and OrganizationIDAttribute;
// data sources and ephemeral resources build their own attribute from a
// different schema package and reference the constant directly.
const (
	ProjectIDDescription      = "The Appwrite project ID. Defaults to the provider-level project_id."
	OrganizationIDDescription = "The Appwrite organization ID. Defaults to the provider-level organization_id."

	DatabaseIDDescription          = "The database ID."
	DedicatedDatabaseIDDescription = "The dedicated database ID."

	MemoryMBDescription      = "The allocated memory in MB."
	CPUMillicoresDescription = "The allocated CPU in millicores."

	// ForcesReplacementNote is appended to every argument that cannot be changed
	// in place.
	//
	// The provider has 170 such attributes across its resources and not one of
	// them said so in its documentation, which means a reader had no way to tell
	// which edit recreates their database. Enforced by
	// TestForceNewAttributesDocumentReplacement, so a new ForceNew argument
	// cannot ship undocumented.
	ForcesReplacementNote = "Changing this forces a new resource to be created."
)
