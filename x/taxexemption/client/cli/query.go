package cli

import (
	"context"

	"github.com/classic-terra/core/v2/x/taxexemption/types"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd() *cobra.Command {
	taxexemptionQueryCmd := &cobra.Command{
		Use:                        "taxexemption",
		Short:                      "Querying commands for the taxexemption module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	taxexemptionQueryCmd.AddCommand(
		GetCmdQueryZonelist(),
		GetCmdQueryExemptlist(),
	)

	return taxexemptionQueryCmd
}

func GetCmdQueryZonelist() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "zones",
		Args:  cobra.NoArgs,
		Short: "Query all burn tax zones",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			// Query store
			res, err := queryClient.TaxExemptionZonesList(context.Background(), &types.QueryTaxExemptionZonesRequest{Pagination: pageReq})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "tax exemption zone list")
	return cmd
}

func GetCmdQueryExemptlist() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "addresses [zone-name]",
		Args:  cobra.ExactArgs(1),
		Short: "Query all tax exemption addresses of a zone",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			zoneName := args[0]

			// Query store
			res, err := queryClient.TaxExemptionAddressList(context.Background(), &types.QueryTaxExemptionAddressRequest{
				ZoneName:   zoneName,
				Pagination: pageReq,
			})
			if err != nil {
				return err
			}
			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "burn tax exemption list")
	return cmd
}