package graphql

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/neo4j/cli/common/clicfg"
	"github.com/neo4j/cli/neo4j/aura/internal/api"
	"github.com/neo4j/cli/neo4j/aura/internal/output"
	"github.com/spf13/cobra"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

func NewCreateCmd(cfg *clicfg.Config) *cobra.Command {
	const (
		instanceIdFlag       = "instance-id"
		nameFlag             = "name"
		instanceUsernameFlag = "instance-username"
		instancePasswordFlag = "instance-password"
		typeDefsFlag         = "type-definitions"
		typeDefsFileFlag     = "type-definitions-file"
		awaitFlag            = "await"
	)

	var (
		instanceId       string
		name             string
		instanceUsername string
		instancePassword string
		typeDefs         string
		typeDefsFile     string
		await            bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Creates a new GraphQL Data API",
		Long: `This endpoint starts the creation process of an Aura GraphQL Data API.

Creating a GraphQL Data API is an asynchronous operation. Use --await flag to wait for the GraphQL Data API to be ready. Once the status transitions from "creating" to "ready" you may begin to use your GraphQL Data API.

This endpoint returns your GraphQL Data API ID, API key, and connection URL for you to use once the GraphQL Data API is running. It is important to store the API key as it is not currently possible to get this or update it.

If you lose your API key, you will need to create a new Authentication provider.. This will not result in any loss of data.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			typeDefs, _ := cmd.Flags().GetString(typeDefsFlag)
			typeDefsFile, _ := cmd.Flags().GetString(typeDefsFileFlag)
			if typeDefs == "" && typeDefsFile == "" {
				return fmt.Errorf("either '--%s' or '--%s' flag needs to be provided", typeDefsFlag, typeDefsFileFlag)
			} else if typeDefs != "" && typeDefsFile != "" {
				return fmt.Errorf("only one of '--%s' or '--%s' flag can be provided", typeDefsFlag, typeDefsFileFlag)
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{
				"name": name,
				"aura_instance": map[string]string{
					"username": instanceUsername,
					"password": instancePassword,
				},
				"security": map[string]any{
					"authentication_providers": []map[string]any{
						{
							"type":    "api-key",
							"name":    "default",
							"enabled": true,
						},
					},
				},
			}

			typeDefsForBody, err := getTypeDefsFromFlag(typeDefs, typeDefsFile, typeDefsFlag, typeDefsFileFlag)
			if err != nil {
				return err
			}
			body["type_definitions"] = typeDefsForBody

			cmd.SilenceUsage = true
			path := fmt.Sprintf("/instances/%s/data-apis/graphql", instanceId)
			resBody, statusCode, err := api.MakeRequest(cfg, path, &api.RequestConfig{
				PostBody: body,
				Method:   http.MethodPost,
			})
			if err != nil {
				return err
			}

			// NOTE: GraphQL Data API create should not return OK (200), it always returns 202, checking both just in case
			if statusCode == http.StatusAccepted || statusCode == http.StatusOK {

				cmd.Println("###############################")
				cmd.Println("# An API key was created. It is important to _store_ the API key as it is not currently possible to get it or update it.")
				cmd.Println("#")
				cmd.Println("# If you lose your API key, you will need to create a new Authentication provider.")
				cmd.Println("# This will not result in any loss of data.")
				cmd.Println("###############################")

				err = output.PrintBody(cmd, cfg, resBody, []string{"id", "name", "status", "url", "authentication_providers"})
				if err != nil {
					return err
				}

				if await {
					cmd.Println("Waiting for GraphQL Data API to be ready...")
					var response api.CreateGraphQLDataApiResponse
					if err := json.Unmarshal(resBody, &response); err != nil {
						return err
					}

					pollResponse, err := api.PollGraphQLDataApi(cfg, instanceId, response.Data.Id, api.GraphQLDataApiStatusCreating)
					if err != nil {
						return err
					}

					cmd.Println("GraphQL Data API Status:", pollResponse.Data.Status)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&instanceId, instanceIdFlag, "", "The ID of the instance to create the GraphQL Data APIs for")
	cmd.MarkFlagRequired(instanceIdFlag)

	cmd.Flags().StringVar(&name, nameFlag, "", "The name of the Data API")
	cmd.MarkFlagRequired(nameFlag)

	cmd.Flags().StringVar(&instanceUsername, instanceUsernameFlag, "", "The username of the instance this GraphQL Data API will be connected to")
	cmd.MarkFlagRequired(instanceUsernameFlag)

	cmd.Flags().StringVar(&instancePassword, instancePasswordFlag, "", "The password of the instance this GraphQL Data API will be connected to")
	cmd.MarkFlagRequired(instancePasswordFlag)

	cmd.Flags().StringVar(&typeDefs, typeDefsFlag, "", "The GraphQL type definitions, NOTE: must be base64 encoded")

	cmd.Flags().StringVar(&typeDefsFile, typeDefsFileFlag, "", "Path to a local GraphQL type definitions file, e.x. path/to/typeDefs.graphql. Must be of file type .graphql")

	cmd.Flags().BoolVar(&await, awaitFlag, false, "Waits until created GraphQL Data API is ready.")

	return cmd
}

func isValidTypeDefs(typDefs string) error {
	_, err := gqlparser.LoadSchema(&ast.Source{
		Input: typDefs,
	})
	if err != nil {
		return fmt.Errorf("provided type definitions are invalid, error(s): %s", err.Error())
	}
	return nil
}

func getTypeDefsFromFlag(typeDefs string, typeDefsFile string, typeDefsFlag string, typeDefsFileFlag string) (string, error) {
	typeDefsForBody := ""
	if typeDefs != "" {
		decodedTypeDefs, err := base64.StdEncoding.DecodeString(typeDefs)
		if err != nil {
			return "", errors.New("provided type definitions are not valid base64")
		}

		err = isValidTypeDefs(string(decodedTypeDefs))
		if err != nil {
			return "", err
		}

		// type defs in request body need to be base 64 encoded
		typeDefsForBody = typeDefs
	} else if typeDefsFile != "" {
		base64EncodedTypeDefs, err := ResolveTypeDefsFileFlagValue(typeDefsFile)
		if err != nil {
			return "", err
		}

		typeDefsForBody = base64EncodedTypeDefs
	} else {
		return "", fmt.Errorf("neither '--%s' nor '--%s' flag value is provided", typeDefsFlag, typeDefsFileFlag)
	}

	return typeDefsForBody, nil
}

func ResolveTypeDefsFileFlagValue(typeDefsFileFlagValue string) (string, error) {
	// typeDefsFileFlagValue is assessed as a local file
	fileInfo, err := os.Stat(typeDefsFileFlagValue)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("type definitions file '%s' does not exist", typeDefsFileFlagValue)
	}
	if !strings.HasSuffix(fileInfo.Name(), ".graphql") {
		return "", fmt.Errorf("type definitions file '%s' must have file type '.graphql'", typeDefsFileFlagValue)
	}

	fileData, err := os.ReadFile(typeDefsFileFlagValue)
	if err != nil {
		return "", fmt.Errorf("reading type definitions file failed with error: %s", err)
	}

	err = isValidTypeDefs(string(fileData))
	if err != nil {
		return "", err
	}

	base64EncodedTypeDefs := base64.StdEncoding.EncodeToString([]byte(fileData))
	if base64EncodedTypeDefs == "" {
		return "", errors.New("read type definitions file is empty")
	}

	return base64EncodedTypeDefs, nil
}
