package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

// DefaultParams intentionally contains no lock tiers. Governance must configure
// at least one tier before users can create positions.
func DefaultParams() Params {
	authority := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	return Params{
		BondDenom: BondDenom,
		LockTiers: []LockTier{},
		Authority: authority,
		Paused:    false,
	}
}

func (p Params) Validate() error {
	if p.BondDenom != BondDenom {
		return ErrInvalidDenom.Wrapf("expected %s, got %s", BondDenom, p.BondDenom)
	}
	if err := validateAuthority(p.Authority); err != nil {
		return ErrInvalidAuthority.Wrapf("authority: %s", err)
	}
	seen := make(map[uint32]struct{}, len(p.LockTiers))
	for _, tier := range p.LockTiers {
		if tier.Id == 0 {
			return ErrInvalidLockTier.Wrap("tier id must be positive")
		}
		if _, exists := seen[tier.Id]; exists {
			return ErrDuplicateLockTier.Wrapf("tier id %d", tier.Id)
		}
		seen[tier.Id] = struct{}{}
		if tier.Duration == nil || *tier.Duration <= 0 {
			return ErrInvalidLockTier.Wrapf("tier %d duration must be positive", tier.Id)
		}
		if tier.Multiplier.IsNil() || !tier.Multiplier.IsPositive() {
			return ErrInvalidLockTier.Wrapf("tier %d multiplier must be positive", tier.Id)
		}
	}
	return nil
}

func validateAuthority(value string) error {
	if value == "" {
		return fmt.Errorf("address is empty")
	}
	_, err := sdk.AccAddressFromBech32(value)
	if err != nil {
		return err
	}
	if value != authtypes.NewModuleAddress(govtypes.ModuleName).String() {
		return fmt.Errorf("authority must be the governance module account")
	}
	return nil
}
