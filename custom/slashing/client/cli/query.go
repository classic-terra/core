package cli

import (
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/spf13/cobra"
)

// GetQueryCmd returns the slashing query commands.
func GetQueryCmd() *cobra.Command {
	slashingQueryCmd := &cobra.Command{
		Use:                        slashingtypes.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", slashingtypes.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	slashingQueryCmd.AddCommand(
		GetCmdQueryParams(),
		GetCmdQuerySigningInfo(),
		GetCmdQuerySigningInfos(),
	)

	return slashingQueryCmd
}

// GetCmdQueryParams returns the current slashing parameters.
func GetCmdQueryParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Short: "Query the current slashing parameters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := slashingtypes.NewQueryClient(clientCtx)
			res, err := queryClient.Params(cmd.Context(), &slashingtypes.QueryParamsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// GetCmdQuerySigningInfo queries a validator's signing information.
func GetCmdQuerySigningInfo() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "signing-info [validator-conspub/address]",
		Short: "Query a validator's signing information",
		Long:  "Query a validator's signing information, with a pubkey ('<appd> comet show-validator') or a validator consensus address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			consAddr, err := normalizeConsAddressArg(clientCtx, args[0])
			if err != nil {
				return err
			}

			queryClient := slashingtypes.NewQueryClient(clientCtx)
			res, err := queryClient.SigningInfo(
				cmd.Context(),
				&slashingtypes.QuerySigningInfoRequest{ConsAddress: consAddr},
			)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// GetCmdQuerySigningInfos queries signing information for all validators.
func GetCmdQuerySigningInfos() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "signing-infos",
		Short: "Query signing information of all validators",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := slashingtypes.NewQueryClient(clientCtx)
			res, err := queryClient.SigningInfos(
				cmd.Context(),
				&slashingtypes.QuerySigningInfosRequest{Pagination: pageReq},
			)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	flags.AddPaginationFlagsToCmd(cmd, "signing infos")

	return cmd
}

func normalizeConsAddressArg(clientCtx client.Context, arg string) (string, error) {
	arg = strings.TrimSpace(arg)

	if _, err := sdk.ConsAddressFromBech32(arg); err == nil {
		return arg, nil
	}

	var pubKey cryptotypes.PubKey
	if err := clientCtx.Codec.UnmarshalInterfaceJSON([]byte(arg), &pubKey); err == nil && pubKey != nil {
		return sdk.GetConsAddress(pubKey).String(), nil
	}

	return "", fmt.Errorf("invalid consensus address or pubkey JSON: %q", arg)
}
