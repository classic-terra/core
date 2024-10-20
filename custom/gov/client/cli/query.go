package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	v2lunc1types "github.com/classic-terra/core/v3/custom/gov/types/v2lunc1"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/version"
	govcli "github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	"github.com/cosmos/cosmos-sdk/x/gov/types"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd() *cobra.Command {
	// Group gov queries under a subcommand
	govQueryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for the governance module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	govQueryCmd.AddCommand(
		GetCmdQueryMinimalDeposit(),
		govcli.GetCmdQueryProposal(),
		govcli.GetCmdQueryProposals(),
		govcli.GetCmdQueryVote(),
		govcli.GetCmdQueryVotes(),
		govcli.GetCmdQueryParams(),
		govcli.GetCmdQueryParam(),
		govcli.GetCmdQueryProposer(),
		govcli.GetCmdQueryDeposit(),
		govcli.GetCmdQueryDeposits(),
		govcli.GetCmdQueryTally(),
	)

	return govQueryCmd
}

// GetCmdQueryMinimalDeposit implements the query proposal command.
func GetCmdQueryMinimalDeposit() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-minimal-deposit [proposal-id]",
		Args:  cobra.ExactArgs(1),
		Short: "Query minimal deposit by min uusd of a single proposal",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query minimal deposit for a proposal. You can find the
proposal-id by running "%s query gov proposals".

Example:
$ %s query gov get-minimal-deposit 1
`,
				version.AppName, version.AppName,
			),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := v2lunc1types.NewQueryClient(clientCtx)

			// validate that the proposal id is a uint
			proposalID, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("proposal-id %s not a valid uint, please input a valid proposal-id", args[0])
			}

			// Query the proposal
			res, err := queryClient.ProposalMinimalLUNCByUusd(
				cmd.Context(),
				&v2lunc1types.QueryProposalRequest{ProposalId: proposalID},
			)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(&res.MinimalDeposit)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
