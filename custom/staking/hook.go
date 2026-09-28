package staking

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
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
func (h TerraStakingHooks) BeforeDelegationCreated(_ context.Context, _ sdk.AccAddress, _ sdk.ValAddress) error {
	return nil
}

func (h TerraStakingHooks) BeforeDelegationSharesModified(_ context.Context, _ sdk.AccAddress, _ sdk.ValAddress) error {
	return nil
}

// Other required hook methods with empty implementations
func (h TerraStakingHooks) AfterDelegationModified(ctx context.Context, _ sdk.AccAddress, valAddr sdk.ValAddress) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if sdkCtx.ChainID() != ColumbusChainID {
		return nil
	}

	// Skip validation during genesis (block height 0)
	if sdkCtx.BlockHeight() == 0 {
		return nil
	}

	validator, err := h.sk.GetValidator(ctx, valAddr)
	if err != nil {
		return nil
	}

	// Get validator's current power (after delegation modified)
	validatorPower := sdk.TokensToConsensusPower(validator.Tokens, h.sk.PowerReduction(ctx))

	// Calculate total power by summing all bonded validators' current power
	// This gives us the current total power including any pending changes
	totalPower := int64(0)

	// Get all validators and sum the power of bonded ones
	allValidators, err := h.sk.GetAllValidators(ctx)
	if err != nil {
		return nil
	}

	for _, val := range allValidators {
		if val.IsBonded() {
			valPower := sdk.TokensToConsensusPower(val.Tokens, h.sk.PowerReduction(ctx))
			totalPower += valPower
		}
	}

	if totalPower == 0 {
		return nil
	}

	// Get validator delegation percent
	validatorDelegationPercent := math.LegacyNewDec(validatorPower).Quo(math.LegacyNewDec(totalPower))

	if validatorDelegationPercent.GT(math.LegacyNewDecWithPrec(20, 2)) {
		return fmt.Errorf("validator power is over the allowed limit")
	}

	return nil
}

func (h TerraStakingHooks) BeforeValidatorSlashed(_ context.Context, _ sdk.ValAddress, _ math.LegacyDec) error {
	return nil
}

func (h TerraStakingHooks) BeforeValidatorModified(_ context.Context, _ sdk.ValAddress) error {
	return nil
}

func (h TerraStakingHooks) AfterValidatorBonded(_ context.Context, _ sdk.ConsAddress, _ sdk.ValAddress) error {
	return nil
}

func (h TerraStakingHooks) AfterValidatorBeginUnbonding(_ context.Context, _ sdk.ConsAddress, _ sdk.ValAddress) error {
	return nil
}

func (h TerraStakingHooks) AfterValidatorRemoved(_ context.Context, _ sdk.ConsAddress, _ sdk.ValAddress) error {
	return nil
}

func (h TerraStakingHooks) AfterUnbondingInitiated(_ context.Context, _ uint64) error {
	return nil
}

// Add this method to TerraStakingHooks
func (h TerraStakingHooks) AfterValidatorCreated(_ context.Context, _ sdk.ValAddress) error {
	return nil
}

// Add the missing method
func (h TerraStakingHooks) BeforeDelegationRemoved(_ context.Context, _ sdk.AccAddress, _ sdk.ValAddress) error {
	return nil
}
