package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/classic-terra/core/v4/x/cron/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
)

func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		GetCmdAddSchedule(),
		GetCmdRemoveSchedule(),
		GetCmdUpdateParams(),
	)

	return cmd
}

func GetCmdAddSchedule() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-schedule [name] [period] [execution-stage] [contract] [msg-json]",
		Args:  cobra.ExactArgs(5),
		Short: "Create a cron schedule",
		Long: strings.TrimSpace(`
Create a cron schedule that executes one Wasm contract message periodically.

$ terrad tx cron add-schedule my-job 10 end-blocker terra1contract... '{"ping":{}}'
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			period, err := strconv.ParseUint(args[1], 10, 64)
			if err != nil {
				return err
			}

			stage, err := parseExecutionStage(args[2])
			if err != nil {
				return err
			}

			msgs := []types.MsgExecuteContract{
				{
					Contract: args[3],
					Msg:      args[4],
				},
			}

			msg := &types.MsgAddSchedule{
				Authority:      clientCtx.GetFromAddress().String(),
				Name:           args[0],
				Period:         period,
				Msgs:           msgs,
				ExecutionStage: stage,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func GetCmdRemoveSchedule() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-schedule [name]",
		Args:  cobra.ExactArgs(1),
		Short: "Remove a cron schedule",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := &types.MsgRemoveSchedule{
				Authority: clientCtx.GetFromAddress().String(),
				Name:      args[0],
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func GetCmdUpdateParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-params [limit]",
		Args:  cobra.ExactArgs(1),
		Short: "Update cron params",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			limit, err := strconv.ParseUint(args[0], 10, 64)
			if err != nil {
				return err
			}

			msg := &types.MsgUpdateParams{
				Authority: clientCtx.GetFromAddress().String(),
				Params:    types.NewParams(limit),
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func parseExecutionStage(v string) (types.ExecutionStage, error) {
	switch strings.ToLower(v) {
	case "begin", "begin-blocker", "begin_blocker":
		return types.ExecutionStage_EXECUTION_STAGE_BEGIN_BLOCKER, nil
	case "end", "end-blocker", "end_blocker":
		return types.ExecutionStage_EXECUTION_STAGE_END_BLOCKER, nil
	default:
		return 0, fmt.Errorf("invalid execution stage %q", v)
	}
}
