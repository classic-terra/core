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
	
	// Debug: always print to see if hook is being called
	fmt.Printf("DEBUG: Hook called! chainID=%s, expectedChainID=%s, blockHeight=%d, valAddr=%s\n", 
		sdkCtx.ChainID(), ColumbusChainID, sdkCtx.BlockHeight(), valAddr.String())
	
	if sdkCtx.ChainID() != ColumbusChainID {
		fmt.Printf("DEBUG: Chain ID mismatch, skipping\n")
		return nil
	}

	// Skip validation during genesis (block height 0)  
	if sdkCtx.BlockHeight() == 0 {
		fmt.Printf("DEBUG: Genesis block, skipping\n")
		return nil
	}

	validator, err := h.sk.GetValidator(ctx, valAddr)
	if err != nil {
		fmt.Printf("DEBUG: Failed to get validator: %v\n", err)
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
		fmt.Printf("DEBUG: Failed to get all validators: %v\n", err)
		return nil
	}
	
	bondedCount := 0
	for _, val := range allValidators {
		if val.IsBonded() {
			valPower := sdk.TokensToConsensusPower(val.Tokens, h.sk.PowerReduction(ctx))
			totalPower += valPower
			bondedCount++
		}
	}

	fmt.Printf("DEBUG: valAddr=%s, validatorPower=%d, totalPower=%d, bondedCount=%d, bonded=%v\n", 
		valAddr.String(), validatorPower, totalPower, bondedCount, validator.IsBonded())

	if totalPower == 0 {
		fmt.Printf("DEBUG: Total power is zero, skipping\n")
		return nil
	}

	// Get validator delegation percent
	validatorDelegationPercent := math.LegacyNewDec(validatorPower).Quo(math.LegacyNewDec(totalPower))

	// Debug: print detailed calculation
	fmt.Printf("DEBUG: percent=%s, threshold=%s, will_fail=%v\n", 
		validatorDelegationPercent.String(), 
		math.LegacyNewDecWithPrec(20, 2).String(),
		validatorDelegationPercent.GT(math.LegacyNewDecWithPrec(20, 2)))

	if validatorDelegationPercent.GT(math.LegacyNewDecWithPrec(20, 2)) {
		fmt.Printf("DEBUG: Returning error - validator power over limit\n")
		return fmt.Errorf("validator power is over the allowed limit")
	}

	fmt.Printf("DEBUG: Hook passed validation\n")
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
