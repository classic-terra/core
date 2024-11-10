package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/gov/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

// AddDeposit adds or updates a deposit of a specific depositor on a specific proposal.
// Activates voting period when appropriate and returns true in that case, else returns false.
func (keeper Keeper) AddDeposit(ctx sdk.Context, proposalID uint64, depositorAddr sdk.AccAddress, depositAmount sdk.Coins) (bool, error) {
	// Checks to see if proposal exists
	proposal, ok := keeper.GetProposal(ctx, proposalID)
	if !ok {
		return false, sdkerrors.Wrapf(types.ErrUnknownProposal, "%d", proposalID)
	}

	// Check if proposal is still depositable
	if (proposal.Status != v1.StatusDepositPeriod) && (proposal.Status != v1.StatusVotingPeriod) {
		return false, sdkerrors.Wrapf(types.ErrInactiveProposal, "%d", proposalID)
	}

	// update the governance module's account coins pool
	err := keeper.bankKeeper.SendCoinsFromAccountToModule(ctx, depositorAddr, types.ModuleName, depositAmount)
	if err != nil {
		return false, err
	}

	// Update proposal
	proposal.TotalDeposit = sdk.NewCoins(proposal.TotalDeposit...).Add(depositAmount...)
	keeper.SetProposal(ctx, proposal)

	// Check if deposit has provided sufficient total funds to transition the proposal into the voting period
	activatedVotingPeriod := false

	// HandleCheckLimitDeposit
	minDeposit := keeper.GetParams(ctx).MinDeposit
	requiredAmount := keeper.GetDepositLimitBaseUusd(ctx, proposalID)
	if requiredAmount.IsZero() {
		requiredAmount = minDeposit[0].Amount
	}

	// Should not set like this
	requiredDepositCoins := sdk.NewCoins(
		sdk.NewCoin(minDeposit[0].Denom, requiredAmount),
	)

	if proposal.Status == v1.StatusDepositPeriod && sdk.NewCoins(proposal.TotalDeposit...).IsAllGTE(requiredDepositCoins) && requiredAmount.GT(sdk.ZeroInt()) {
		keeper.ActivateVotingPeriod(ctx, proposal)

		activatedVotingPeriod = true
	}

	// Add or update deposit object
	deposit, found := keeper.GetDeposit(ctx, proposalID, depositorAddr)

	if found {
		deposit.Amount = sdk.NewCoins(deposit.Amount...).Add(depositAmount...)
	} else {
		deposit = v1.NewDeposit(proposalID, depositorAddr, depositAmount)
	}

	// called when deposit has been added to a proposal, however the proposal may not be active
	keeper.Hooks().AfterProposalDeposit(ctx, proposalID, depositorAddr)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeProposalDeposit,
			sdk.NewAttribute(sdk.AttributeKeyAmount, depositAmount.String()),
			sdk.NewAttribute(types.AttributeKeyProposalID, fmt.Sprintf("%d", proposalID)),
		),
	)

	keeper.SetDeposit(ctx, deposit)

	return activatedVotingPeriod, nil
}

// validateInitialDeposit validates if initial deposit is greater than or equal to the minimum
// required at the time of proposal submission. This threshold amount is determined by
// the deposit parameters. Returns nil on success, error otherwise.
func (keeper Keeper) validateInitialDeposit(ctx sdk.Context, initialDeposit sdk.Coins) error {
	params := keeper.GetParams(ctx)
	minInitialDepositRatio, err := sdk.NewDecFromStr(params.MinInitialDepositRatio)
	// set offset price change 3%
	offsetPrice := sdk.NewDecWithPrec(97, 2)
	if err != nil {
		return err
	}
	if minInitialDepositRatio.IsZero() {
		return nil
	}
	totalLuncDeposit, err := keeper.GetMinimumDepositBaseUusd(ctx)
	luncCoin := sdk.NewCoin(params.MinDeposit[0].Denom, totalLuncDeposit)
	minDepositCoins := params.MinDeposit
	minDepositCoins[0] = luncCoin

	if err != nil {
		return err
	}
	for i := range minDepositCoins {
		minDepositCoins[i].Amount = sdk.NewDecFromInt(minDepositCoins[i].Amount).Mul(minInitialDepositRatio).Mul(offsetPrice).TruncateInt()
	}
	if !initialDeposit.IsAllGTE(minDepositCoins) {
		return sdkerrors.Wrapf(types.ErrMinDepositTooSmall, "was (%s), need (%s)", initialDeposit, minDepositCoins)
	}
	return nil
}
