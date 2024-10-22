package graphql

import (
	"fmt"
	"net/http"

	"github.com/neo4j/cli/common/clicfg"
	"github.com/neo4j/cli/neo4j/aura/internal/api"
	"github.com/neo4j/cli/neo4j/aura/internal/output"
	"github.com/spf13/cobra"
)

func NewUpdateCmd(cfg *clicfg.Config) *cobra.Command {
	const (
		instanceIdFlag             = "instance-id"
		nameFlag                   = "name"
		instanceUsernameFlag       = "instance-username"
		instancePasswordFlag       = "instance-password"
		typeDefsFlag               = "type-definitions"
		typeDefsFileFlag           = "type-definitions-file"
		featureSubgraphEnabledFlag = "feature-subgraph-enabled"
		awaitFlag                  = "await"
	)

	var (
		instanceId             string
		name                   string
		instanceUsername       string
		instancePassword       string
		typeDefs               string
		typeDefsFile           string
		featureSubgraphEnabled string
		await                  bool
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Edit a GraphQL Data API",
		Long:  "This endpoint edits a specific GraphQL Data API.",
		Args:  cobra.ExactArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			typeDefs, _ := cmd.Flags().GetString(typeDefsFlag)
			typeDefsFile, _ := cmd.Flags().GetString(typeDefsFileFlag)
			if typeDefs != "" && typeDefsFile != "" {
				return fmt.Errorf("only one of '--%s' or '--%s' flag can be provided", typeDefsFlag, typeDefsFileFlag)
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			body := map[string]any{}

			if name != "" {
				body["name"] = name
			}
			if typeDefs != "" {
				body["type_definitions"] = typeDefs
			}
			if typeDefsFile != "" {
				//
				// TODO: get type defs from file! and encode base64.
				//
				body["type_definitions"] = typeDefsFile
			}
			if instanceUsername != "" || instancePassword != "" {
				auraInstance := map[string]string{}

				if instanceUsername != "" {
					auraInstance["username"] = instanceUsername
				}
				if instancePassword != "" {
					auraInstance["password"] = instancePassword
				}

				body["aura_instance"] = auraInstance
			}
			if featureSubgraphEnabled != "" {
				body["features"] = map[string]string{
					"subgraph": featureSubgraphEnabled,
				}
			}

			cmd.SilenceUsage = true
			path := fmt.Sprintf("/instances/%s/data-apis/graphql/%s", instanceId, args[0])

			resBody, statusCode, err := api.MakeRequest(cfg, path, &api.RequestConfig{
				Method:   http.MethodPatch,
				PostBody: body,
			})
			if err != nil {
				return err
			}

			if statusCode == http.StatusAccepted {
				err = output.PrintBody(cmd, cfg, resBody, []string{"id", "name", "status", "url"})
				if err != nil {
					return err
				}

				if await {
					cmd.Println("Waiting for GraphQL Data API to be updated...")
					pollResponse, err := api.PollGraphQLDataApi(cfg, instanceId, args[0], api.GraphQLDataApiStatusUpdating)
					if err != nil {
						return err
					}

					cmd.Println("GraphQL Data API Status:", pollResponse.Data.Status)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&instanceId, instanceIdFlag, "", "The ID of the instance to get the Data API for")
	cmd.MarkFlagRequired("instance-id")

	cmd.Flags().StringVar(&name, nameFlag, "", "The name of the Data API")

	cmd.Flags().StringVar(&instanceUsername, instanceUsernameFlag, "", "The username of the instance this GraphQL Data API will be connected to")

	cmd.Flags().StringVar(&instancePassword, instancePasswordFlag, "", "The password of the instance this GraphQL Data API will be connected to")

	cmd.Flags().StringVar(&typeDefs, typeDefsFlag, "", "The GraphQL type definitions, NOTE: must be base64 encoded")

	cmd.Flags().StringVar(&typeDefsFile, typeDefsFileFlag, "", "Path to the local GraphQL type definitions file, e.x. path/to/typeDefs.graphql")

	cmd.Flags().StringVar(&featureSubgraphEnabled, featureSubgraphEnabledFlag, "", "Wether or not GraphQL subgraph is enabled, use 'true' or 'false'")

	cmd.Flags().BoolVar(&await, awaitFlag, false, "Waits until created GraphQL Data API is ready.")

	return cmd
}
