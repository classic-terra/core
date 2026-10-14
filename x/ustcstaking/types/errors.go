package types

import "cosmossdk.io/errors"

var (
	ErrInvalidDenom         = errors.Register(ModuleName, 2, "invalid denomination")
	ErrInvalidLockTier      = errors.Register(ModuleName, 3, "invalid lock tier")
	ErrDuplicateLockTier    = errors.Register(ModuleName, 4, "duplicate lock tier")
	ErrInvalidAuthority     = errors.Register(ModuleName, 5, "invalid authority")
	ErrInvalidRewardState   = errors.Register(ModuleName, 6, "invalid reward state")
	ErrInvalidPosition      = errors.Register(ModuleName, 7, "invalid position")
	ErrInvalidGenesis       = errors.Register(ModuleName, 8, "invalid genesis")
	ErrNoActiveShares       = errors.Register(ModuleName, 9, "no active shares")
	ErrRewardPoolInsolvent  = errors.Register(ModuleName, 10, "reward pool is insolvent")
	ErrPositionNotFound     = errors.Register(ModuleName, 11, "position not found")
	ErrModulePaused         = errors.Register(ModuleName, 12, "module is paused")
	ErrUnauthorized         = errors.Register(ModuleName, 13, "unauthorized")
	ErrPositionOwner        = errors.Register(ModuleName, 14, "position owner mismatch")
	ErrInvalidPositionState = errors.Register(ModuleName, 15, "invalid position state")
	ErrNotMatured           = errors.Register(ModuleName, 16, "position has not matured")
	ErrPositionIDExhausted  = errors.Register(ModuleName, 17, "position ID exhausted")
	ErrInvalidLockSnapshot  = errors.Register(ModuleName, 18, "invalid lock snapshot")
	ErrCorruptOwnerIndex    = errors.Register(ModuleName, 19, "corrupt owner index")
)
