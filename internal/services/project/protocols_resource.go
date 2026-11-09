package project

import (
	"github.com/appwrite/sdk-for-go/v7/models"
	sdkproject "github.com/appwrite/sdk-for-go/v7/project"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// projectProtocols are the protocols a project's API is served over, in the
// order the server lists them.
var projectProtocols = []toggleItem{
	{id: "rest", name: "the REST API"},
	{id: "graphql", name: "the GraphQL API", note: "Unlike `rest`, this applies to requests made with an API key too."},
	{id: "websocket", name: "Realtime over WebSocket"},
}

var protocolsKind = toggleKind{
	typeSuffix: "_project_protocols",
	noun:       "protocol",
	description: "Enables and disables the protocols a project's API is served over. Each protocol is managed only " +
		"when its attribute is set. Disabling REST does not block requests made with an API key, including the " +
		"provider's own; disabling GraphQL does.",
	items: projectProtocols,
	update: func(client *sdkproject.Project, id string, enabled bool) (*models.Project, error) {
		return client.UpdateProtocol(id, enabled)
	},
	current: func(project *models.Project) map[string]bool {
		current := make(map[string]bool, len(project.Protocols))
		for _, protocol := range project.Protocols {
			current[protocol.Id] = protocol.Enabled
		}
		return current
	},
}

func NewProtocolsResource() resource.Resource {
	return &toggleResource{kind: protocolsKind}
}
