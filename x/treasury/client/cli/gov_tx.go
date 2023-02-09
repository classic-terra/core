package cli

import (
	"fmt"
	"strings"

	"github.com/classic-terra/core/x/treasury/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/spf13/cobra"
)

func ProposalAddWhitelistCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "whitelist-add [addresses] --title [text] --description [text]",
		Short: "Submit a set whitelist address proposal",
		Long: fmt.Sprintf(`Submit a proposal to add whitelist addresses for tax exemption.
Example:
$ %s tx gov submit-proposal whitelist-add terra1dczz24r33fwlj0q5ra7rcdryjpk9hxm8rwy39t,terra1qt8mrv72gtvmnca9z6ftzd7slqhaf8m60aa7ye --title "whitelist" --description "whitelist"
			`, version.AppName),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			addresses := strings.Split(args[0], ",")

			proposalTitle, err := cmd.Flags().GetString(cli.FlagTitle)
			if err != nil {
				return fmt.Errorf("proposal title: %s", err)
			}
			proposalDescr, err := cmd.Flags().GetString(cli.FlagDescription)
			if err != nil {
				return fmt.Errorf("proposal description: %s", err)
			}
			depositArg, err := cmd.Flags().GetString(cli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositArg)
			if err != nil {
				return err
			}

			content := types.SetWhitelistAddressProposal{
				Title:            proposalTitle,
				Description:      proposalDescr,
				WhitelistAddress: addresses,
			}

			msg, err := govtypes.NewMsgSubmitProposal(&content, deposit, clientCtx.GetFromAddress())
			if err != nil {
				return err
			}
			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	// proposal flags
	cmd.Flags().String(cli.FlagTitle, "", "Title of proposal")
	cmd.Flags().String(cli.FlagDescription, "", "Description of proposal")
	cmd.Flags().String(cli.FlagDeposit, "", "Deposit of proposal")
	return cmd
}

func ProposalRemoveWhitelistCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "whitelist-remove [addresses] --title [text] --description [text]",
		Short: "Submit a remove whitelist address proposal",
		Long: fmt.Sprintf(`Submit a proposal to remove whitelist addresses for tax exemption.
Example:
$ %s tx gov submit-proposal whitelist-remove terra1dczz24r33fwlj0q5ra7rcdryjpk9hxm8rwy39t,terra1qt8mrv72gtvmnca9z6ftzd7slqhaf8m60aa7ye --title "whitelist" --description "whitelist"
			`, version.AppName),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			addresses := strings.Split(args[0], ",")

			proposalTitle, err := cmd.Flags().GetString(cli.FlagTitle)
			if err != nil {
				return fmt.Errorf("proposal title: %s", err)
			}
			proposalDescr, err := cmd.Flags().GetString(cli.FlagDescription)
			if err != nil {
				return fmt.Errorf("proposal description: %s", err)
			}
			depositArg, err := cmd.Flags().GetString(cli.FlagDeposit)
			if err != nil {
				return err
			}
			deposit, err := sdk.ParseCoinsNormalized(depositArg)
			if err != nil {
				return err
			}

			content := types.RemoveWhitelistAddressProposal{
				Title:            proposalTitle,
				Description:      proposalDescr,
				WhitelistAddress: addresses,
			}

			msg, err := govtypes.NewMsgSubmitProposal(&content, deposit, clientCtx.GetFromAddress())
			if err != nil {
				return err
			}
			if err = msg.ValidateBasic(); err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	// proposal flags
	cmd.Flags().String(cli.FlagTitle, "", "Title of proposal")
	cmd.Flags().String(cli.FlagDescription, "", "Description of proposal")
	cmd.Flags().String(cli.FlagDeposit, "", "Deposit of proposal")
	return cmd
}
