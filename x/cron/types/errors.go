package types

import errorsmod "cosmossdk.io/errors"

var (
	ErrInvalidScheduleName   = errorsmod.Register(ModuleName, 1, "schedule name is invalid")
	ErrInvalidSchedulePeriod = errorsmod.Register(ModuleName, 2, "schedule period is invalid")
	ErrInvalidScheduleStage  = errorsmod.Register(ModuleName, 3, "schedule execution stage is invalid")
	ErrDuplicateSchedule     = errorsmod.Register(ModuleName, 4, "schedule already exists")
	ErrScheduleNotFound      = errorsmod.Register(ModuleName, 5, "schedule not found")
	ErrInvalidAuthority      = errorsmod.Register(ModuleName, 6, "authority is invalid")
	ErrInvalidMsg            = errorsmod.Register(ModuleName, 7, "contract execute message is invalid")
	ErrInvalidLimit          = errorsmod.Register(ModuleName, 8, "limit cannot be zero")
)
