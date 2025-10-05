package types

import (
	"context"
	stdmath "math"
	"math/rand"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/cometbft/cometbft/crypto/secp256k1"
	tmprotocrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"
)

const OracleDecPrecision = 8

func GenerateRandomTestCase() (rates []float64, valValAddrs []sdk.ValAddress, stakingKeeper DummyStakingKeeper) {
	valValAddrs = []sdk.ValAddress{}
	mockValidators := []MockValidator{}

	base := stdmath.Pow10(OracleDecPrecision)

	r := rand.New(rand.NewSource(int64(time.Now().Nanosecond())))
	numInputs := 10 + (r.Int() % 100)
	for i := 0; i < numInputs; i++ {
		rate := float64(int64(r.Float64()*base)) / base
		rates = append(rates, rate)

		pubKey := secp256k1.GenPrivKey().PubKey()
		valValAddr := sdk.ValAddress(pubKey.Address())
		valValAddrs = append(valValAddrs, valValAddr)

		power := r.Int63()%1000 + 1
		mockValidator := NewMockValidator(valValAddr, power)
		mockValidators = append(mockValidators, mockValidator)
	}

	stakingKeeper = NewDummyStakingKeeper(mockValidators)

	return
}

var _ StakingKeeper = DummyStakingKeeper{}

// DummyStakingKeeper dummy staking keeper to test ballot
type DummyStakingKeeper struct {
	validators []MockValidator
}

// NewDummyStakingKeeper returns new DummyStakingKeeper instance
func NewDummyStakingKeeper(validators []MockValidator) DummyStakingKeeper {
	return DummyStakingKeeper{
		validators: validators,
	}
}

func (sk DummyStakingKeeper) Validators() []MockValidator {
	return sk.validators
}

func (sk DummyStakingKeeper) Validator(_ context.Context, address sdk.ValAddress) (stakingtypes.ValidatorI, error) {
	for _, validator := range sk.validators {
		if validator.operator.Equals(address) {
			return validator, nil
		}
	}

	return nil, nil
}

func (DummyStakingKeeper) TotalBondedTokens(context.Context) (math.Int, error) {
	return math.ZeroInt(), nil
}

func (DummyStakingKeeper) Slash(context.Context, sdk.ConsAddress, int64, int64, math.LegacyDec) (math.Int, error) {
	return math.ZeroInt(), nil
}

func (DummyStakingKeeper) ValidatorsPowerStoreIterator(context.Context) (storetypes.Iterator, error) {
	return storetypes.KVStoreReversePrefixIterator(nil, nil), nil
}

func (DummyStakingKeeper) Jail(context.Context, sdk.ConsAddress) error {
	return nil
}

func (sk DummyStakingKeeper) GetLastValidatorPower(ctx sdk.Context, operator sdk.ValAddress) (power int64, err error) {
	validator, err := sk.Validator(ctx, operator)
	if err != nil {
		return 0, err
	}
	return validator.GetConsensusPower(sdk.DefaultPowerReduction), nil
}

// MaxValidators returns the maximum amount of bonded validators
func (DummyStakingKeeper) MaxValidators(context.Context) (uint32, error) {
	return 100, nil
}

// PowerReduction - is the amount of staking tokens required for 1 unit of consensus-engine power
func (DummyStakingKeeper) PowerReduction(context.Context) (res math.Int) {
	res = sdk.DefaultPowerReduction
	return
}

type MockValidator struct {
	power    int64
	operator sdk.ValAddress
}

var _ stakingtypes.ValidatorI = MockValidator{}

func (MockValidator) IsJailed() bool                          { return false }
func (MockValidator) GetMoniker() string                      { return "" }
func (MockValidator) GetStatus() stakingtypes.BondStatus      { return stakingtypes.Bonded }
func (MockValidator) IsBonded() bool                          { return true }
func (MockValidator) IsUnbonded() bool                        { return false }
func (MockValidator) IsUnbonding() bool                       { return false }
func (v MockValidator) GetOperator() string                   { return v.operator.String() }
func (MockValidator) ConsPubKey() (cryptotypes.PubKey, error) { return nil, nil }
func (MockValidator) TmConsPublicKey() (tmprotocrypto.PublicKey, error) {
	return tmprotocrypto.PublicKey{}, nil
}
func (MockValidator) GetConsAddr() ([]byte, error) { return nil, nil }
func (v MockValidator) GetTokens() math.Int {
	return sdk.TokensFromConsensusPower(v.power, sdk.DefaultPowerReduction)
}

func (v MockValidator) GetBondedTokens() math.Int {
	return sdk.TokensFromConsensusPower(v.power, sdk.DefaultPowerReduction)
}
func (v MockValidator) GetConsensusPower(_ math.Int) int64             { return v.power }
func (v *MockValidator) SetConsensusPower(power int64)                 { v.power = power }
func (v MockValidator) GetCommission() math.LegacyDec                  { return math.LegacyZeroDec() }
func (v MockValidator) GetMinSelfDelegation() math.Int                 { return math.OneInt() }
func (v MockValidator) GetDelegatorShares() math.LegacyDec             { return math.LegacyNewDec(v.power) }
func (v MockValidator) TokensFromShares(math.LegacyDec) math.LegacyDec { return math.LegacyZeroDec() }
func (v MockValidator) TokensFromSharesTruncated(math.LegacyDec) math.LegacyDec {
	return math.LegacyZeroDec()
}

func (v MockValidator) TokensFromSharesRoundUp(math.LegacyDec) math.LegacyDec {
	return math.LegacyZeroDec()
}

func (v MockValidator) SharesFromTokens(_ math.Int) (math.LegacyDec, error) {
	return math.LegacyZeroDec(), nil
}

func (v MockValidator) SharesFromTokensTruncated(_ math.Int) (math.LegacyDec, error) {
	return math.LegacyZeroDec(), nil
}

func NewMockValidator(valAddr sdk.ValAddress, power int64) MockValidator {
	return MockValidator{
		power:    power,
		operator: valAddr,
	}
}
