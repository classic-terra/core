package types

import (
	"fmt"

	"cosmossdk.io/math"
	core "github.com/classic-terra/core/v4/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

// Parameter keys
var (
	// Terra liquidity pool(usdr unit) made available per ${PoolRecoveryPeriod} (usdr unit)
	KeyBasePool = []byte("BasePool")
	// The period required to recover BasePool
	KeyPoolRecoveryPeriod = []byte("PoolRecoveryPeriod")
	// Min spread
	KeyMinStabilitySpread = []byte("MinStabilitySpread")
	// EpochLengthBlocks governs how many blocks constitute a market epoch
	KeyEpochLengthBlocks = []byte("EpochLengthBlocks")
	// Fraction of swap fee to burn
	KeySwapFeeBurnRate = []byte("SwapFeeBurnRate")
	// Fraction of swap fee to send to Community Pool
	KeySwapFeeCommunityRate = []byte("SwapFeeCommunityRate")
	// Maximum oracle age in seconds
	KeyMaxOracleAgeSeconds = []byte("MaxOracleAgeSeconds")
	// TWAP lookback window in blocks
	KeyTWAPLookbackWindow = []byte("TWAPLookbackWindow")
	// Maximum TWAP deviation
	KeyMaxTWAPDeviation = []byte("MaxTWAPDeviation")
	// Daily cap factor
	KeyDailyCapFactor = []byte("DailyCapFactor")
)

// Default parameter values
var (
	DefaultBasePool           = math.LegacyNewDec(1000000 * core.MicroUnit) // 1000,000sdr = 1000,000,000,000usdr
	DefaultPoolRecoveryPeriod = core.BlocksPerDay                           // 14,400
	DefaultMinStabilitySpread = math.LegacyNewDecWithPrec(2, 2)             // 2%
	DefaultEpochLengthBlocks  = 30 * core.BlocksPerDay                      // 30 days worth of blocks
	// Default fee distribution: 0% burn, 0% community pool, 100% to oracle (remainder)
	DefaultSwapFeeBurnRate      = math.LegacyZeroDec()
	DefaultSwapFeeCommunityRate = math.LegacyZeroDec()
	// Default oracle freshness: 75 seconds (25 blocks * 3s)
	DefaultMaxOracleAgeSeconds = uint64(75)
	// Default TWAP window: 45 blocks (~2.25 minutes at 3s/block)
	DefaultTWAPLookbackWindow = uint64(45)
	// Default TWAP deviation: 10%
	DefaultMaxTWAPDeviation = math.LegacyNewDecWithPrec(10, 2) // 0.10
	// Default daily cap: 10% of pool balance per day
	DefaultDailyCapFactor = math.LegacyNewDecWithPrec(10, 2) // 0.10
)

var _ paramstypes.ParamSet = &Params{}

// DefaultParams creates default market module parameters
func DefaultParams() Params {
	return Params{
		BasePool:             DefaultBasePool,
		PoolRecoveryPeriod:   DefaultPoolRecoveryPeriod,
		MinStabilitySpread:   DefaultMinStabilitySpread,
		EpochLengthBlocks:    DefaultEpochLengthBlocks,
		SwapFeeBurnRate:      DefaultSwapFeeBurnRate,
		SwapFeeCommunityRate: DefaultSwapFeeCommunityRate,
		MaxOracleAgeSeconds:  DefaultMaxOracleAgeSeconds,
		TwapLookbackWindow:   DefaultTWAPLookbackWindow,
		MaxTwapDeviation:     DefaultMaxTWAPDeviation,
		DailyCapFactor:       DefaultDailyCapFactor,
	}
}

// ParamKeyTable returns the parameter key table.
func ParamKeyTable() paramstypes.KeyTable {
	return paramstypes.NewKeyTable().RegisterParamSet(&Params{})
}

// String implements fmt.Stringer interface
func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// ParamSetPairs implements the ParamSet interface and returns all the key/value pairs
// pairs of market module's parameters.
func (p *Params) ParamSetPairs() paramstypes.ParamSetPairs {
	return paramstypes.ParamSetPairs{
		paramstypes.NewParamSetPair(KeyBasePool, &p.BasePool, validateBasePool),
		paramstypes.NewParamSetPair(KeyPoolRecoveryPeriod, &p.PoolRecoveryPeriod, validatePoolRecoveryPeriod),
		paramstypes.NewParamSetPair(KeyMinStabilitySpread, &p.MinStabilitySpread, validateMinStabilitySpread),
		paramstypes.NewParamSetPair(KeyEpochLengthBlocks, &p.EpochLengthBlocks, validateEpochLengthBlocks),
		paramstypes.NewParamSetPair(KeySwapFeeBurnRate, &p.SwapFeeBurnRate, validateFraction),
		paramstypes.NewParamSetPair(KeySwapFeeCommunityRate, &p.SwapFeeCommunityRate, validateFraction),
		paramstypes.NewParamSetPair(KeyMaxOracleAgeSeconds, &p.MaxOracleAgeSeconds, validateMaxOracleAgeSeconds),
		paramstypes.NewParamSetPair(KeyTWAPLookbackWindow, &p.TwapLookbackWindow, validateTWAPLookbackWindow),
		paramstypes.NewParamSetPair(KeyMaxTWAPDeviation, &p.MaxTwapDeviation, validateFraction),
		paramstypes.NewParamSetPair(KeyDailyCapFactor, &p.DailyCapFactor, validateFraction),
	}
}

// Validate a set of params
func (p Params) Validate() error {
	if p.BasePool.IsNegative() {
		return fmt.Errorf("mint base pool should be positive or zero, is %s", p.BasePool)
	}
	if p.PoolRecoveryPeriod == 0 {
		return fmt.Errorf("pool recovery period should be positive, is %d", p.PoolRecoveryPeriod)
	}
	if p.MinStabilitySpread.IsNegative() || p.MinStabilitySpread.GT(math.LegacyOneDec()) {
		return fmt.Errorf("market minimum stability spead should be a value between [0,1], is %s", p.MinStabilitySpread)
	}
	if p.EpochLengthBlocks == 0 {
		return fmt.Errorf("epoch length blocks should be positive, is %d", p.EpochLengthBlocks)
	}

	// Fee distribution fractions must be within [0,1] and sum <= 1
	if err := validateFraction(p.SwapFeeBurnRate); err != nil {
		return fmt.Errorf("swap fee burn rate invalid: %w", err)
	}
	if err := validateFraction(p.SwapFeeCommunityRate); err != nil {
		return fmt.Errorf("swap fee community rate invalid: %w", err)
	}
	if p.SwapFeeBurnRate.Add(p.SwapFeeCommunityRate).GT(math.LegacyOneDec()) {
		return fmt.Errorf("sum of burn and community rates must be <= 1: %s", p.SwapFeeBurnRate.Add(p.SwapFeeCommunityRate))
	}

	if p.MaxOracleAgeSeconds == 0 {
		return fmt.Errorf("max oracle age seconds must be positive, is %d", p.MaxOracleAgeSeconds)
	}
	if p.TwapLookbackWindow == 0 {
		return fmt.Errorf("TWAP lookback window must be positive, is %d", p.TwapLookbackWindow)
	}
	if err := validateFraction(p.MaxTwapDeviation); err != nil {
		return fmt.Errorf("max TWAP deviation invalid: %w", err)
	}
	if err := validateFraction(p.DailyCapFactor); err != nil {
		return fmt.Errorf("daily cap factor invalid: %w", err)
	}

	return nil
}

func validateBasePool(i interface{}) error {
	v, ok := i.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNegative() {
		return fmt.Errorf("mint base pool must be positive or zero: %s", v)
	}

	return nil
}

func validatePoolRecoveryPeriod(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v <= 0 {
		return fmt.Errorf("pool recovery period must be positive: %d", v)
	}

	return nil
}

func validateMinStabilitySpread(i interface{}) error {
	v, ok := i.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNegative() {
		return fmt.Errorf("min spread must be positive or zero: %s", v)
	}

	if v.GT(math.LegacyOneDec()) {
		return fmt.Errorf("min spread is too large: %s", v)
	}

	return nil
}

func validateEpochLengthBlocks(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v == 0 {
		return fmt.Errorf("epoch length blocks must be positive: %d", v)
	}
	return nil
}

// validateFraction ensures a decimal is in [0, 1]
func validateFraction(i interface{}) error {
	v, ok := i.(math.LegacyDec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v.IsNegative() {
		return fmt.Errorf("fraction must be >= 0: %s", v)
	}
	if v.GT(math.LegacyOneDec()) {
		return fmt.Errorf("fraction must be <= 1: %s", v)
	}
	return nil
}

func validateMaxOracleAgeSeconds(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v == 0 {
		return fmt.Errorf("max oracle age seconds must be positive: %d", v)
	}
	return nil
}

func validateTWAPLookbackWindow(i interface{}) error {
	v, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if v == 0 {
		return fmt.Errorf("TWAP lookback window must be positive: %d", v)
	}
	return nil
}
