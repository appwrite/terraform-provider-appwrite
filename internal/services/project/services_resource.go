package project

import (
	"github.com/appwrite/sdk-for-go/v7/models"
	sdkproject "github.com/appwrite/sdk-for-go/v7/project"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// projectServices are the services Appwrite lets a project disable, in the
// order the server lists them. They are the entries of the server's service
// configuration marked optional; the rest (console, webhooks, tokens and so on)
// cannot be switched off, and the route rejects them.
var projectServices = []toggleItem{
	{id: "account", name: "the Account service", note: "Users call it to sign up, sign in and manage their own account."},
	{id: "avatars", name: "the Avatars service"},
	{id: "databases", name: "the legacy Databases service (collections and documents)"},
	{id: "tablesdb", name: "the TablesDB service (tables and rows)"},
	{id: "locale", name: "the Locale service"},
	{id: "health", name: "the Health service"},
	{id: "project", name: "the Project service"},
	{id: "storage", name: "the Storage service"},
	{id: "teams", name: "the Teams service"},
	{id: "users", name: "the Users service"},
	{id: "vcs", name: "the VCS service", note: "It connects Git repositories to functions and sites."},
	{id: "sites", name: "the Sites service"},
	{id: "functions", name: "the Functions service"},
	{id: "proxy", name: "the Proxy service"},
	{id: "graphql", name: "the GraphQL service", note: "This does not switch the GraphQL endpoint itself; `graphql` on appwrite_project_protocols does."},
	{id: "migrations", name: "the Migrations service"},
	{id: "messaging", name: "the Messaging service"},
	{id: "advisor", name: "the Advisor service"},
}

var servicesKind = toggleKind{
	typeSuffix: "_project_services",
	noun:       "service",
	description: "Enables and disables the API services of a project. Each service is managed only when its " +
		"attribute is set. Requests made with an API key, including the provider's own, are not blocked by a " +
		"disabled service; it applies to requests from users and guests.",
	items: projectServices,
	update: func(client *sdkproject.Project, id string, enabled bool) (*models.Project, error) {
		return client.UpdateService(id, enabled)
	},
	current: func(project *models.Project) map[string]bool {
		current := make(map[string]bool, len(project.Services))
		for _, service := range project.Services {
			current[service.Id] = service.Enabled
		}
		return current
	},
}

func NewServicesResource() resource.Resource {
	return &toggleResource{kind: servicesKind}
}
