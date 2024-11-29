package v2custom

import (
	fmt "fmt"
	time "time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/gov/codec"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	"github.com/classic-terra/core/v3/types/assets"
)

// Default governance params
var (
	DefaultMinUusdDepositTokens = sdk.NewInt(500_000_000) // Minimal uusd deposit for a proposal to enter voting period
)

var _ sdk.Msg = &MsgUpdateParams{}

// Route implements the sdk.Msg interface.
func (msg MsgUpdateParams) Route() string { return govtypes.RouterKey }

// Type implements the sdk.Msg interface.
func (msg MsgUpdateParams) Type() string { return sdk.MsgTypeURL(&msg) }

// ValidateBasic implements the sdk.Msg interface.
func (msg MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid authority address: %s", err)
	}

	return msg.Params.ValidateBasic()
}

// GetSignBytes returns the message bytes to sign over.
func (msg MsgUpdateParams) GetSignBytes() []byte {
	bz := codec.ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// GetSigners returns the expected signers for a MsgUpdateParams.
func (msg MsgUpdateParams) GetSigners() []sdk.AccAddress {
	authority, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{authority}
}

// NewParams creates a new Params instance with given values.
func NewParams(
	minDeposit sdk.Coins, maxDepositPeriod, votingPeriod time.Duration,
	quorum, threshold, vetoThreshold, minInitialDepositRatio string, burnProposalDeposit, burnVoteQuorum, burnVoteVeto bool,
	minUusdDeposit sdk.Coin,
) Params {
	return Params{
		MinDeposit:                 minDeposit,
		MaxDepositPeriod:           &maxDepositPeriod,
		VotingPeriod:               &votingPeriod,
		Quorum:                     quorum,
		Threshold:                  threshold,
		VetoThreshold:              vetoThreshold,
		MinInitialDepositRatio:     minInitialDepositRatio,
		BurnProposalDepositPrevote: burnProposalDeposit,
		BurnVoteQuorum:             burnVoteQuorum,
		BurnVoteVeto:               burnVoteVeto,
		MinUusdDeposit:             minUusdDeposit,
	}
}

// DefaultParams returns the default governance params
func DefaultParams() Params {
	return NewParams(
		sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, govv1.DefaultMinDepositTokens)),
		govv1.DefaultPeriod,
		govv1.DefaultPeriod,
		govv1.DefaultQuorum.String(),
		govv1.DefaultThreshold.String(),
		govv1.DefaultVetoThreshold.String(),
		govv1.DefaultMinInitialDepositRatio.String(),
		govv1.DefaultBurnProposalPrevote,
		govv1.DefaultBurnVoteQuorom,
		govv1.DefaultBurnVoteVeto,
		sdk.NewCoin(assets.MicroUSDDenom, DefaultMinUusdDepositTokens), // 1,000,000 microLuna
	)
}

// ValidateBasic performs basic validation on governance parameters.
func (p Params) ValidateBasic() error {
	if minDeposit := sdk.Coins(p.MinDeposit); minDeposit.Empty() || !minDeposit.IsValid() {
		return fmt.Errorf("invalid minimum deposit: %s", minDeposit)
	}

	if p.MaxDepositPeriod == nil {
		return fmt.Errorf("maximum deposit period must not be nil: %d", p.MaxDepositPeriod)
	}

	if p.MaxDepositPeriod.Seconds() <= 0 {
		return fmt.Errorf("maximum deposit period must be positive: %d", p.MaxDepositPeriod)
	}

	quorum, err := sdk.NewDecFromStr(p.Quorum)
	if err != nil {
		return fmt.Errorf("invalid quorum string: %w", err)
	}
	if quorum.IsNegative() {
		return fmt.Errorf("quorom cannot be negative: %s", quorum)
	}
	if quorum.GT(math.LegacyOneDec()) {
		return fmt.Errorf("quorom too large: %s", p.Quorum)
	}

	threshold, err := sdk.NewDecFromStr(p.Threshold)
	if err != nil {
		return fmt.Errorf("invalid threshold string: %w", err)
	}
	if !threshold.IsPositive() {
		return fmt.Errorf("vote threshold must be positive: %s", threshold)
	}
	if threshold.GT(math.LegacyOneDec()) {
		return fmt.Errorf("vote threshold too large: %s", threshold)
	}

	vetoThreshold, err := sdk.NewDecFromStr(p.VetoThreshold)
	if err != nil {
		return fmt.Errorf("invalid vetoThreshold string: %w", err)
	}
	if !vetoThreshold.IsPositive() {
		return fmt.Errorf("veto threshold must be positive: %s", vetoThreshold)
	}
	if vetoThreshold.GT(math.LegacyOneDec()) {
		return fmt.Errorf("veto threshold too large: %s", vetoThreshold)
	}

	if p.VotingPeriod == nil {
		return fmt.Errorf("voting period must not be nil: %d", p.VotingPeriod)
	}

	if p.VotingPeriod.Seconds() <= 0 {
		return fmt.Errorf("voting period must be positive: %s", p.VotingPeriod)
	}

	minInitialDepositRatio, err := math.LegacyNewDecFromStr(p.MinInitialDepositRatio)
	if err != nil {
		return fmt.Errorf("invalid mininum initial deposit ratio of proposal: %w", err)
	}
	if minInitialDepositRatio.IsNegative() {
		return fmt.Errorf("mininum initial deposit ratio of proposal must be positive: %s", minInitialDepositRatio)
	}
	if minInitialDepositRatio.GT(math.LegacyOneDec()) {
		return fmt.Errorf("mininum initial deposit ratio of proposal is too large: %s", minInitialDepositRatio)
	}

	if p.MinUusdDeposit.IsZero() || !p.MinUusdDeposit.IsValid() {
		return fmt.Errorf("invalid minimum uusd deposit: %s", p.MinUusdDeposit)
	}

	return nil
}
