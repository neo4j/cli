package customermanagedkey

import (
	"fmt"

	"github.com/neo4j/cli/pkg/aura/api"
	"github.com/spf13/cobra"
)

var listTenantId string

var ListCmd = &cobra.Command{
	Use:   "list",
	Short: "Returns a list of customer managed keys",
	Long: `This subcommand returns a list containing a summary of each of your customer managed keys. To find out more about a specific key, retrieve the details using the get subcommand.

You can filter keys in a particular tenant using --tenant-id. If the tenant flag is not specified, this endpoint lists all keys a user has access to across all tenants.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if listTenantId != "" {
			return api.MakeRequest(cmd, "GET", fmt.Sprintf("/customer-managed-keys?tenantId=%s", listTenantId), nil)
		} else {
			return api.MakeRequest(cmd, "GET", "/customer-managed-keys", nil)
		}
	},
}

func init() {
	ListCmd.Flags().StringVar(&listTenantId, "tenant-id", "", "An optional Tenant ID to filter customer managed keys in a tenant")
}
