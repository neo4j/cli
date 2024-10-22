package graphql_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/neo4j/cli/neo4j/aura/internal/test/testutils"
)

func TestUpdateGraphQLDataApiFlagsValidation(t *testing.T) {
	helper := testutils.NewAuraTestHelper(t)
	defer helper.Close()

	helper.SetConfigValue("aura.beta-enabled", "true")

	instanceId := "2f49c2b3"
	dataApiId := "afdb4e9d"

	tests := map[string]struct {
		executedCommand string
		expectedError   string
	}{"provide only one type defs flag": {
		executedCommand: fmt.Sprintf("data-api graphql update --output json --instance-id %s --type-definitions bla --type-definitions-file blabla %s", instanceId, dataApiId),
		expectedError:   "Error: only one of '--type-definitions' or '--type-definitions-file' flag can be provided",
	},
		"invalid type defs": {
			executedCommand: fmt.Sprintf("data-api graphql update --output json --instance-id %s --type-definitions bla %s", instanceId, dataApiId),
			expectedError:   "Error: provided type definitions are not valid base64",
		},
		"invalid feature subgraph enabled value": {
			executedCommand: fmt.Sprintf("data-api graphql update --output json --instance-id %s --feature-subgraph-enabled fd %s", instanceId, dataApiId),
			expectedError:   "Error: strconv.ParseBool: parsing \"fd\": invalid syntax",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			helper.ExecuteCommand(tt.executedCommand)
			helper.AssertErr(tt.expectedError)
		})
	}
}

func TestUpdateGraphQLDataApiWithResponse(t *testing.T) {
	instanceId := "2f49c2b3"
	dataApiId := "75a234b5"
	instanceUsername := "neo4j"
	instancePassword := "dfjglhssdopfrow"
	name := "my-data-api-2"
	typeDefs := "dHlwZS=="

	mockResponse := `{
		"data": {
			"id": "afdb4e9d",
			"name": "friendly-name-4",
			"status": "ready",
			"url": "https://afdb4e9d.28be6e4d8d3e836019.graphql.neo4j.io/graphql"
		}
	}`
	expectedResponse := `{
		"data": {
			"id": "afdb4e9d",
			"name": "friendly-name-4",
			"status": "ready",
			"url": "https://afdb4e9d.28be6e4d8d3e836019.graphql.neo4j.io/graphql"
        }
	}`

	tests := map[string]struct {
		mockResponse        string
		executeCommand      string
		expectedRequestBody string
		expectedResponse    string
	}{"update the name": {
		mockResponse:        mockResponse,
		executeCommand:      fmt.Sprintf("data-api graphql update --output json --instance-id %s --name %s %s", instanceId, name, dataApiId),
		expectedRequestBody: `{"name":"my-data-api-2"}`,
		expectedResponse:    expectedResponse,
	}, "update the password": {
		mockResponse:        mockResponse,
		executeCommand:      fmt.Sprintf("data-api graphql update --output json --instance-id %s --instance-password %s %s", instanceId, instancePassword, dataApiId),
		expectedRequestBody: `{"aura_instance":{"password":"dfjglhssdopfrow"}}`,
		expectedResponse:    expectedResponse,
	}, "update the username": {
		mockResponse:        mockResponse,
		executeCommand:      fmt.Sprintf("data-api graphql update --output json --instance-id %s --instance-username %s %s", instanceId, instanceUsername, dataApiId),
		expectedRequestBody: `{"aura_instance":{"username":"neo4j"}}`,
		expectedResponse:    expectedResponse,
	}, "update the password and instance": {
		mockResponse:        mockResponse,
		executeCommand:      fmt.Sprintf("data-api graphql update --output json --instance-id %s --instance-password %s --instance-username %s %s", instanceId, instancePassword, instanceUsername, dataApiId),
		expectedRequestBody: `{"aura_instance":{"password":"dfjglhssdopfrow","username":"neo4j"}}`,
		expectedResponse:    expectedResponse,
	}, "update the typeDefs": {
		mockResponse:        mockResponse,
		executeCommand:      fmt.Sprintf("data-api graphql update --output json --instance-id %s --type-definitions %s %s", instanceId, typeDefs, dataApiId),
		expectedRequestBody: `{"type_definitions":"dHlwZS=="}`,
		expectedResponse:    expectedResponse,
	}, "update the feature subgraph enabled bool to false": {
		mockResponse:        mockResponse,
		executeCommand:      fmt.Sprintf("data-api graphql update --output json --instance-id %s --feature-subgraph-enabled false %s", instanceId, dataApiId),
		expectedRequestBody: `{"features":{"subgraph":false}}`,
		expectedResponse:    expectedResponse,
	}, "update the feature subgraph enabled bool to true": {
		mockResponse:        mockResponse,
		executeCommand:      fmt.Sprintf("data-api graphql update --output json --instance-id %s --feature-subgraph-enabled true %s", instanceId, dataApiId),
		expectedRequestBody: `{"features":{"subgraph":true}}`,
		expectedResponse:    expectedResponse,
	}, "all in one request": {
		mockResponse:        mockResponse,
		executeCommand:      fmt.Sprintf("data-api graphql update --output json --instance-id %s --instance-password %s --instance-username %s --type-definitions %s --name %s --feature-subgraph-enabled true %s", instanceId, instancePassword, instanceUsername, typeDefs, name, dataApiId),
		expectedRequestBody: `{"aura_instance":{"password":"dfjglhssdopfrow","username":"neo4j"},"features":{"subgraph":true},"name":"my-data-api-2","type_definitions":"dHlwZS=="}`,
		expectedResponse:    expectedResponse,
	},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			helper := testutils.NewAuraTestHelper(t)
			defer helper.Close()

			helper.SetConfigValue("aura.beta-enabled", "true")

			mockHandler := helper.NewRequestHandlerMock(fmt.Sprintf("/v1/instances/%s/data-apis/graphql/%s", instanceId, dataApiId), http.StatusAccepted, tt.mockResponse)

			helper.ExecuteCommand(tt.executeCommand)

			mockHandler.AssertCalledTimes(1)
			mockHandler.AssertCalledWithMethod(http.MethodPatch)
			mockHandler.AssertCalledWithBody(tt.expectedRequestBody)

			helper.AssertOutJson(tt.expectedResponse)
		})
	}
}
