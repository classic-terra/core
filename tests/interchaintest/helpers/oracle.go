package helpers

import (
	"context"

	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
)
const validatorKeyName = "validator"
func ExecOracleMsgAggragatePrevote(ctx context.Context, validator *cosmos.ChainNode, salt string, exchangeRate string) (error) {
	args := []string{
		 "oracle", "aggregate-prevote",
		 salt,
		exchangeRate,

	}

	command := validator.TxCommand(validatorKeyName, args...)

	_, _,  err := validator.Exec(ctx, command, nil)
	if err != nil {
		return err
	}
	return nil
}

func ExecOracleMsgAggregateVote(ctx context.Context, validator *cosmos.ChainNode, salt string, exchangeRate string) error {
	args := []string{
		 "oracle", "aggregate-vote",
		 salt,
		exchangeRate,

	}

	command := validator.TxCommand(validatorKeyName, args...)

	_, _,  err := validator.Exec(ctx, command, nil)
	if err != nil {
		return err
	}
	return nil
}