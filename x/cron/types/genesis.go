package types

import "fmt"

func DefaultGenesisState() *GenesisState {
	return &GenesisState{Params: DefaultParams()}
}

func ValidateGenesis(gs *GenesisState) error {
	return gs.Validate()
}

func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	seen := make(map[string]struct{}, len(gs.ScheduleList))
	for _, schedule := range gs.ScheduleList {
		if err := validateSchedule(schedule); err != nil {
			return err
		}
		if _, exists := seen[schedule.Name]; exists {
			return fmt.Errorf("%w: %s", ErrDuplicateSchedule, schedule.Name)
		}
		seen[schedule.Name] = struct{}{}
	}

	return nil
}

func ValidateSchedule(schedule Schedule) error {
	return validateSchedule(schedule)
}

func validateSchedule(schedule Schedule) error {
	if schedule.Name == "" {
		return ErrInvalidScheduleName
	}
	if schedule.Period == 0 {
		return ErrInvalidSchedulePeriod
	}
	if len(schedule.Msgs) == 0 {
		return ErrInvalidMsg
	}
	for _, msg := range schedule.Msgs {
		if err := validateMsgExecuteContract(msg); err != nil {
			return err
		}
	}
	if _, ok := ExecutionStage_name[int32(schedule.ExecutionStage)]; !ok {
		return ErrInvalidScheduleStage
	}
	return nil
}

func ValidateMsgExecuteContract(msg MsgExecuteContract) error {
	return validateMsgExecuteContract(msg)
}

func validateMsgExecuteContract(msg MsgExecuteContract) error {
	if msg.Contract == "" {
		return ErrInvalidMsg
	}
	if msg.Msg == "" {
		return ErrInvalidMsg
	}
	return nil
}
