package staking

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

const (
	ColumbusChainID = "columbus-5"
)

var _ stakingtypes.StakingHooks = &TerraStakingHooks{}

// TerraStakingHooks implements staking hooks to enforce validator power limit
type TerraStakingHooks struct {
	sk stakingkeeper.Keeper
}

func NewTerraStakingHooks(sk stakingkeeper.Keeper) *TerraStakingHooks {
	return &TerraStakingHooks{sk: sk}
}

// Implement required staking hooks interface methods
func (h TerraStakingHooks) BeforeDelegationCreated(_ sdk.Context, _ sdk.AccAddress, _ sdk.ValAddress) error {
	return nil
}

func (h TerraStakingHooks) BeforeDelegationSharesModified(_ sdk.Context, _ sdk.AccAddress, _ sdk.ValAddress) error {
	return nil
}

// Other required hook methods with empty implementations
func (h TerraStakingHooks) AfterDelegationModified(ctx sdk.Context, _ sdk.AccAddress, valAddr sdk.ValAddress) error {
	if ctx.ChainID() != ColumbusChainID {
		return nil
	}

	validator, found := h.sk.GetValidator(ctx, valAddr)
	if !found {
		return nil
	}

	// Get validator's current power (after delegation modified)
	validatorPower := sdk.TokensToConsensusPower(validator.Tokens, h.sk.PowerReduction(ctx))

	// Get the total power of the validator set
	totalPower := h.sk.GetLastTotalPower(ctx)
	if totalPower.IsZero() {
		return nil
	}

	// Get validator delegation percent
	validatorDelegationPercent := sdk.NewDec(validatorPower).QuoInt64(totalPower.Int64())

	if validatorDelegationPercent.GT(sdk.NewDecWithPrec(20, 2)) {
		panic("validator power is over the allowed limit")
	}

	return nil
}

func (h TerraStakingHooks) BeforeValidatorSlashed(_ sdk.Context, _ sdk.ValAddress, _ sdk.Dec) error {
	return nil
}

func (h TerraStakingHooks) BeforeValidatorModified(_ sdk.Context, _ sdk.ValAddress) error {
	return nil
}

func (h TerraStakingHooks) AfterValidatorBonded(_ sdk.Context, _ sdk.ConsAddress, _ sdk.ValAddress) error {
	return nil
}

func (h TerraStakingHooks) AfterValidatorBeginUnbonding(_ sdk.Context, _ sdk.ConsAddress, _ sdk.ValAddress) error {
	return nil
}

func (h TerraStakingHooks) AfterValidatorRemoved(_ sdk.Context, _ sdk.ConsAddress, _ sdk.ValAddress) error {
	return nil
}

func (h TerraStakingHooks) AfterUnbondingInitiated(_ sdk.Context, _ uint64) error {
	return nil
}

// Add this method to TerraStakingHooks
func (h TerraStakingHooks) AfterValidatorCreated(_ sdk.Context, _ sdk.ValAddress) error {
	return nil
}

// Add the missing method
func (h TerraStakingHooks) BeforeDelegationRemoved(_ sdk.Context, _ sdk.AccAddress, _ sdk.ValAddress) error {
	return nil
}
