package ante

import (
	treasury "github.com/terra-money/core/x/treasury/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distribution "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
)

// TaxPowerUpgradeHeight is when taxes are allowed to go into effect
// This will still need a parameter change proposal, but can be activated
// anytime after this height
const (
	TaxPowerUpgradeHeight = 9346889
	TaxPowerSplitHeight   = 123456789
	WhitelistHeight       = 12345678910
)

// BurnTaxFeeDecorator will immediately burn the collected Tax
type BurnTaxFeeDecorator struct {
	TreasuryKeeper     TreasuryKeeper
	DistributionKeeper distribution.Keeper
	BankKeeper         BankKeeper
}

// NewBurnTaxFeeDecorator returns new tax fee decorator instance
func NewBurnTaxFeeDecorator(treasuryKeeper TreasuryKeeper, bankKeeper BankKeeper, distributionKeeper distribution.Keeper) BurnTaxFeeDecorator {
	return BurnTaxFeeDecorator{
		TreasuryKeeper:     treasuryKeeper,
		DistributionKeeper: distributionKeeper,
		BankKeeper:         bankKeeper,
	}
}

var BurnTaxAddressWhitelist = map[string]byte{
	"terra10atxpzafqfjy58z0dvugmd9zf63fycr6uvwhjm": 1,
	"terra1jrq7xa63a4qgpdgtj70k8yz5p32ps9r7mlj3yr": 1,
	"terra15s66unmdcpknuxxldd7fsr44skme966tdckq8c": 1,
	"terra1u0p7xuwlg0zsqgntagdkyjyumsegd8agzhug99": 1,
	"terra1fax8l6srhew5tu2mavmu83js3v7vsqf9yr4fv7": 1,
	"terra132wegs0kf9q65t9gsm3g2y06l98l2k4treepkq": 1,
	"terra1l89hprfccqxgzzypjzy3fnp7vqpnkqg5vvqgjc": 1,
	"terra1ns7lfvrxzter4d2yl9tschdwntcxa25vtsvd8a": 1,
	"terra1vuvju6la7pj6t8d8zsx4g8ea85k2cg5u62cdhl": 1,
	"terra1lzdux37s4anmakvg7pahzh03zlf43uveq83wh2": 1,
	"terra1ky3qcf7v45n6hwfmkm05acwczvlq8ahnq778wf": 1,
	"terra17m8tkde0mav43ckeehp537rsz4usqx5jayhf08": 1,
	"terra1urj8va62jeygra7y3a03xeex49mjddh3eul0qa": 1,
	"terra10wyptw59xc52l86pg86sy0xcm3nm5wg6a3cf7l": 1,
	"terra1sujaqwaw7ls9fh6a4x7n06nv7fxx5xexwlnrkf": 1,
	"terra1qg59nhvag222kp6fyzxt83l4sw02huymqnklww": 1,
	"terra1dxxnwxlpjjkl959v5xrghx0dtvut60eef6vcch": 1,
	"terra1y246m036et7vu69nsg4kapelj0tywe8vsmp34d": 1,
	"terra1j39c9sjr0zpjnrfjtthuua0euecv7txavxvq36": 1,
	"terra1t0jthtq9zhm4ldtvs9epp02zp23f355wu6zrzq": 1,
	"terra12dxclvqrgt7w3s7gtwpdkxgymexv8stgqcr0yu": 1,
	"terra1az3dsad74pwhylrrexnn5qylzj783uyww2s7xz": 1,
	"terra1ttq26dq4egr5exmhd6gezerrxhlutx9u90uncn": 1,
	"terra13e9670yuvfs06hctt9pmgjnz0yw28p0wgnhrqn": 1,
	"terra1skmktm537pfaycgu9jx4fqryjt6pf77ycpesw0": 1,
	"terra14q8cazgt58y2xkd26mlukemwth0cnvfqmgz2qk": 1,
	"terra163vzxz9wwy320ccwy73qe6h33yzg2yhyvv5nsf": 1,
	"terra1kj43wfnvrgc2ep94dgmwvnzv8vnkkxrxmrnhkp": 1,
	"terra1gu6re549pn0mdpshtv75t3xugn347jghlhul73": 1,
	"terra1gft3qujlq04yza3s2r238mql2yn3xxqepzt2up": 1,
	"terra174pe7qe7g867spzdfs5f4rf9fuwmm42zf4hykf": 1,
	"terra1ju68sg6k39t385sa0fazqvjgh6m6gkmsmp4lln": 1,
	"terra1dlh7k4hcnsrvlfuzhdzx3ctynj7s8dde9zmdyd": 1,
	"terra18wcdhpzpteharlkks5n6k7ent0mjyftvcpm6ee": 1,
	"terra1xmkwsauuk3kafua9k23hrkfr76gxmwdfq5c09d": 1,
	"terra1t957gces65xd6p8g4cuqnyd0sy5tzku59njydd": 1,
	"terra1s4rd0y5e4gasf0krdm2w8sjhsmh030m74f2x9v": 1,
	"terra15jya6ugxp65y80y5h82k4gv90pd7acv58xp6jj": 1,
	"terra14yqy9warjkxyecda5kf5a68qlknf4ve4sh7sa6": 1,
	"terra1yxras4z0fs9ugsg2hew9334k65uzejwcslyx0y": 1,
	"terra1p0vl4s4gp46vy6dm352s2fgtw6hccypph7zc3u": 1,
	"terra1hhj92twle9x8rjkr3yffujexsy5ldexak5rglz": 1,
	"terra18vnrzlzm2c4xfsx382pj2xndqtt00rvhu24sqe": 1,
}

// AnteHandle handles msg tax fee checking
func (btfd BurnTaxFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	// Do not proceed if you are below this block height
	currHeight := ctx.BlockHeight()
	if currHeight < TaxPowerUpgradeHeight {
		return next(ctx, tx, simulate)
	}

	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	msgs := feeTx.GetMsgs()

	// At this point we have already run the DeductFees AnteHandler and taken the fees from the sending account
	// Now we remove the taxes from the gas reward and immediately burn it

	if !simulate {
		// Compute taxes again.
		var taxMsgs []sdk.Msg

		if currHeight >= WhitelistHeight {
			for _, msg := range msgs {
				if taxMsg := checkMessageWhitelist(msg); taxMsg != nil {
					taxMsgs = append(taxMsgs, taxMsg)
				}
			}
		}

		taxes := FilterMsgAndComputeTax(ctx, btfd.TreasuryKeeper, taxMsgs...)

		// Record tax proceeds
		if !taxes.IsZero() {
			if currHeight >= TaxPowerSplitHeight {
				feePool := btfd.DistributionKeeper.GetFeePool(ctx)

				for _, taxCoin := range taxes {
					splitTaxRate := btfd.TreasuryKeeper.GetBurnSplitRate(ctx)
					splitcoinAmount := splitTaxRate.MulInt(taxCoin.Amount).RoundInt()

					splitCoin := sdk.NewCoin(taxCoin.Denom, splitcoinAmount)
					taxCoin.Amount = taxCoin.Amount.Sub(splitCoin.Amount)

					if splitCoin.Amount.IsPositive() {
						feePool.CommunityPool = feePool.CommunityPool.Add(sdk.NewDecCoinFromCoin(splitCoin))
					}
				}

				btfd.DistributionKeeper.SetFeePool(ctx, feePool)
			}

			err = btfd.BankKeeper.SendCoinsFromModuleToModule(ctx, types.FeeCollectorName, treasury.BurnModuleName, taxes)

			if err != nil {
				return ctx, sdkerrors.Wrapf(sdkerrors.ErrInsufficientFunds, err.Error())
			}
		}
	}

	return next(ctx, tx, simulate)
}

func checkMessageWhitelist(msg sdk.Msg) sdk.Msg {
	var whitelistedRecipients []string
	var whitelistedSenders []string
	recipientWhitelistCount := 0
	senderWhitelistCount := 0
	binancePositionInSet := map[int]byte{}

	switch v := msg.(type) {
	case *banktypes.MsgSend:
		whitelistedSenders = append(whitelistedSenders, v.FromAddress)
		whitelistedRecipients = append(whitelistedRecipients, v.ToAddress)
	case *banktypes.MsgMultiSend:
		for _, input := range v.Inputs {
			whitelistedSenders = append(whitelistedSenders, input.Address)
		}

		for _, output := range v.Outputs {
			whitelistedRecipients = append(whitelistedRecipients, output.Address)
		}
	default:
		return msg
		// TODO: We might want to return an error if we cannot match the msg types, but as such I think that means we also need to cover MsgSetSendEnabled & MsgUpdateParams
		// return ctx, sdkerrors.Wrap(sdkerrors.ErrInvalidType, "Unsupported message type")
	}

	// extract subset
	for i, sender := range whitelistedSenders {
		if _, ok := BurnTaxAddressWhitelist[sender]; ok {
			senderWhitelistCount += 1
			binancePositionInSet[i] = 1
		}
	}

	for i, recipient := range whitelistedRecipients {
		if _, ok := BurnTaxAddressWhitelist[recipient]; ok {
			recipientWhitelistCount += 1
			binancePositionInSet[i] = 1
		}
	}

	// filter out case 1 -> 5, only 6 -> 9 left
	if senderWhitelistCount == len(whitelistedSenders) || recipientWhitelistCount == len(whitelistedRecipients) {
		return nil
	}

	// filter out case 9, only 6 -> 8 left
	if !(senderWhitelistCount == 0 && recipientWhitelistCount == 0) {
		newTaxMsg := banktypes.MsgMultiSend{
			Inputs:  []banktypes.Input{},
			Outputs: []banktypes.Output{},
		}

		// if not binance pair, add pair to taxMsg for tax calculation
		msgMultiSend := msg.(*banktypes.MsgMultiSend)
		for i := 0; i < len(whitelistedSenders); i++ {
			if _, ok := binancePositionInSet[i]; ok {
				continue
			}

			newTaxMsg.Inputs = append(newTaxMsg.Inputs, msgMultiSend.Inputs[i])
			newTaxMsg.Outputs = append(newTaxMsg.Outputs, msgMultiSend.Outputs[i])
		}

		return &newTaxMsg
	}

	return msg
}
