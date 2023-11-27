package cli

import (
	"fmt"
	"strings"

	"github.com/classic-terra/core/v2/x/taxexemption/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/spf13/cobra"
)

func ProposalAddTaxExemptionZoneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-tax-exemption-zone [zone] [addresses] --exempt-incoming [true|false] --exempt-outgoing [true|false] --exempt-cross-zone [true|false] --title [text] --description [text]",
		Short: "Submit an add tax exemption zone proposal",
		Long: fmt.Sprintf(`Submit a proposal to add a tax exemption zone.
Example:
$ %s tx gov submit-proposal add-tax-exemption-zone zonexyz terra1dczz24r33fwlj0q5ra7rcdryjpk9hxm8rwy39t,terra1qt8mrv72gtvmnca9z6ftzd7slqhaf8m60aa7ye --exempt-incoming true --exempt-outgoing false --exempt-cross-zone false --title "add tax exemption zone" --description "add tax exemption zone"
			`, version.AppName),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			zoneName := args[0]
			addresses := strings.Split(args[1], ",")

			exemptIncoming, err := cmd.Flags().GetBool("exempt-incoming")
			if err != nil {
				return fmt.Errorf("exempt incoming: %s", err)
			}

			exemptOutgoing, err := cmd.Flags().GetBool("exempt-outgoing")
			if err != nil {
				return fmt.Errorf("exempt outgoing: %s", err)
			}

			exemptCrossZone, err := cmd.Flags().GetBool("exempt-cross-zone")
			if err != nil {
				return fmt.Errorf("exempt cross zone: %s", err)
			}

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

			content := types.AddTaxExemptionZoneProposal{
				Title:       proposalTitle,
				Description: proposalDescr,
				Zone:        zoneName,
				Outgoing:    exemptOutgoing,
				Incoming:    exemptIncoming,
				CrossZone:   exemptCrossZone,
				Addresses:   addresses,
			}

			msg, err := govv1beta1.NewMsgSubmitProposal(&content, deposit, clientCtx.GetFromAddress())
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
	cmd.Flags().Bool("exempt-incoming", false, "Exempt incoming tx from tax")
	cmd.Flags().Bool("exempt-outgoing", false, "Exempt outgoing tx from tax")
	cmd.Flags().Bool("exempt-cross-zone", false, "Exempt cross zone tx from tax")

	cmd.Flags().String(cli.FlagTitle, "", "Title of proposal")
	cmd.Flags().String(cli.FlagDescription, "", "Description of proposal")
	cmd.Flags().String(cli.FlagDeposit, "", "Deposit of proposal")
	return cmd
}

func ProposalRemoveTaxExemptionZoneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-tax-exemption-zone [zone] --title [text] --description [text]",
		Short: "Submit a remove tax exemption zone proposal",
		Long: fmt.Sprintf(`Submit a proposal to remove a tax exemption zone.
Example:
$ %s tx gov submit-proposal remove-tax-exemption-zone zonexyz --title "remove tax exemption zone" --description "remove tax exemption zone"
			`, version.AppName),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			zoneName := args[0]

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

			content := types.RemoveTaxExemptionZoneProposal{
				Title:       proposalTitle,
				Description: proposalDescr,
				Zone:        zoneName,
			}

			msg, err := govv1beta1.NewMsgSubmitProposal(&content, deposit, clientCtx.GetFromAddress())
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

func ProposalModifyTaxExemptionZoneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "modify-tax-exemption-zone [zone] --exempt-incoming [true|false] --exempt-outgoing [true|false] --exempt-cross-zone [true|false] --title [text] --description [text]",
		Short: "Submit an modify tax exemption zone proposal",
		Long: fmt.Sprintf(`Submit a proposal to modify a tax exemption zone.
Example:
$ %s tx gov submit-proposal modify-tax-exemption-zone zonexyz --exempt-incoming false --exempt-outgoing true --exempt-cross-zone false --title "add tax exemption zone" --description "add tax exemption zone"
			`, version.AppName),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			zoneName := args[0]

			exemptIncoming, err := cmd.Flags().GetBool("exempt-incoming")
			if err != nil {
				return fmt.Errorf("exempt incoming: %s", err)
			}

			exemptOutgoing, err := cmd.Flags().GetBool("exempt-outgoing")
			if err != nil {
				return fmt.Errorf("exempt outgoing: %s", err)
			}

			exemptCrossZone, err := cmd.Flags().GetBool("exempt-cross-zone")
			if err != nil {
				return fmt.Errorf("exempt cross zone: %s", err)
			}

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

			content := types.ModifyTaxExemptionZoneProposal{
				Title:       proposalTitle,
				Description: proposalDescr,
				Zone:        zoneName,
				Outgoing:    exemptOutgoing,
				Incoming:    exemptIncoming,
				CrossZone:   exemptCrossZone,
			}

			msg, err := govv1beta1.NewMsgSubmitProposal(&content, deposit, clientCtx.GetFromAddress())
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
	cmd.Flags().Bool("exempt-incoming", false, "Exempt incoming tx from tax")
	cmd.Flags().Bool("exempt-outgoing", false, "Exempt outgoing tx from tax")
	cmd.Flags().Bool("exempt-cross-zone", false, "Exempt cross zone tx from tax")

	cmd.Flags().String(cli.FlagTitle, "", "Title of proposal")
	cmd.Flags().String(cli.FlagDescription, "", "Description of proposal")
	cmd.Flags().String(cli.FlagDeposit, "", "Deposit of proposal")
	return cmd
}

func ProposalAddTaxExemptionAddressCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-tax-exemption-address [zone] [addresses] --title [text] --description [text]",
		Short: "Submit an add tax exemption address proposal",
		Long: fmt.Sprintf(`Submit a proposal to add addresses for tax exemption to a zone.
Example:
$ %s tx gov submit-proposal add-tax-exemption-address zonexyz terra1dczz24r33fwlj0q5ra7rcdryjpk9hxm8rwy39t,terra1qt8mrv72gtvmnca9z6ftzd7slqhaf8m60aa7ye --title "add tax exemption address" --description "add address to tax exemption list"
			`, version.AppName),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			zoneName := args[0]
			addresses := strings.Split(args[1], ",")

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

			content := types.AddTaxExemptionAddressProposal{
				Title:       proposalTitle,
				Description: proposalDescr,
				Zone:        zoneName,
				Addresses:   addresses,
			}

			msg, err := govv1beta1.NewMsgSubmitProposal(&content, deposit, clientCtx.GetFromAddress())
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

func ProposalRemoveTaxExemptionAddressCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove-tax-exemption-address [zone] [addresses] --title [text] --description [text]",
		Short: "Submit a remove tax exemption address proposal",
		Long: fmt.Sprintf(`Submit a proposal to remove addresses from tax exemption for a zone.
Example:
$ %s tx gov submit-proposal remove-tax-exemption-address zonexyz terra1dczz24r33fwlj0q5ra7rcdryjpk9hxm8rwy39t,terra1qt8mrv72gtvmnca9z6ftzd7slqhaf8m60aa7ye --title "remove tax exemption address" --description "remove address from tax exemption list for zone xyz"
			`, version.AppName),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			zoneName := args[0]
			addresses := strings.Split(args[1], ",")

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

			content := types.RemoveTaxExemptionAddressProposal{
				Title:       proposalTitle,
				Description: proposalDescr,
				Zone:        zoneName,
				Addresses:   addresses,
			}

			msg, err := govv1beta1.NewMsgSubmitProposal(&content, deposit, clientCtx.GetFromAddress())
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
