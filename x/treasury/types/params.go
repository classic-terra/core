package types

import (
	"fmt"

	"gopkg.in/yaml.v2"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"

	core "github.com/terra-money/core/types"
)

// Parameter keys
var (
	KeyTaxPolicy               = []byte("TaxPolicy")
	KeyRewardPolicy            = []byte("RewardPolicy")
	KeySeigniorageBurdenTarget = []byte("SeigniorageBurdenTarget")
	KeyMiningIncrement         = []byte("MiningIncrement")
	KeyWindowShort             = []byte("WindowShort")
	KeyWindowLong              = []byte("WindowLong")
	KeyWindowProbation         = []byte("WindowProbation")
	KeyBurnTaxSplit            = []byte("BurnTaxSplit")
	KeyBurnTaxWhitelist		   = []byte("BurnTaxWhitelist")
)

// Default parameter values
var (
	DefaultTaxPolicy = PolicyConstraints{
		RateMin:       sdk.NewDecWithPrec(5, 4),                                             // 0.05%
		RateMax:       sdk.NewDecWithPrec(1, 2),                                             // 1%
		Cap:           sdk.NewCoin(core.MicroSDRDenom, sdk.OneInt().MulRaw(core.MicroUnit)), // 1 SDR Tax cap
		ChangeRateMax: sdk.NewDecWithPrec(25, 5),                                            // 0.025%
	}
	DefaultRewardPolicy = PolicyConstraints{
		RateMin:       sdk.NewDecWithPrec(5, 2),             // 5%
		RateMax:       sdk.NewDecWithPrec(50, 2),            // 50%
		ChangeRateMax: sdk.NewDecWithPrec(25, 3),            // 2.5%
		Cap:           sdk.NewCoin("unused", sdk.ZeroInt()), // UNUSED
	}
	DefaultSeigniorageBurdenTarget = sdk.NewDecWithPrec(67, 2)  // 67%
	DefaultMiningIncrement         = sdk.NewDecWithPrec(107, 2) // 1.07 mining increment; exponential growth
	DefaultWindowShort             = uint64(4)                  // a month
	DefaultWindowLong              = uint64(52)                 // a year
	DefaultWindowProbation         = uint64(12)                 // 3 month
	DefaultTaxRate                 = sdk.NewDecWithPrec(1, 3)   // 0.1%
	DefaultRewardWeight            = sdk.NewDecWithPrec(5, 2)   // 5%
	DefaultBurnTaxSplit            = sdk.NewDecWithPrec(5, 1)   // 50%
	DefaultBurnTaxWhitelist		   = []string{
		"terra10atxpzafqfjy58z0dvugmd9zf63fycr6uvwhjm",
		"terra1jrq7xa63a4qgpdgtj70k8yz5p32ps9r7mlj3yr",
		"terra15s66unmdcpknuxxldd7fsr44skme966tdckq8c",
		"terra1u0p7xuwlg0zsqgntagdkyjyumsegd8agzhug99",
		"terra1fax8l6srhew5tu2mavmu83js3v7vsqf9yr4fv7",
		"terra132wegs0kf9q65t9gsm3g2y06l98l2k4treepkq",
		"terra1l89hprfccqxgzzypjzy3fnp7vqpnkqg5vvqgjc",
		"terra1ns7lfvrxzter4d2yl9tschdwntcxa25vtsvd8a",
		"terra1vuvju6la7pj6t8d8zsx4g8ea85k2cg5u62cdhl",
		"terra1lzdux37s4anmakvg7pahzh03zlf43uveq83wh2",
		"terra1ky3qcf7v45n6hwfmkm05acwczvlq8ahnq778wf",
		"terra17m8tkde0mav43ckeehp537rsz4usqx5jayhf08",
		"terra1urj8va62jeygra7y3a03xeex49mjddh3eul0qa",
		"terra10wyptw59xc52l86pg86sy0xcm3nm5wg6a3cf7l",
		"terra1sujaqwaw7ls9fh6a4x7n06nv7fxx5xexwlnrkf",
		"terra1qg59nhvag222kp6fyzxt83l4sw02huymqnklww",
		"terra1dxxnwxlpjjkl959v5xrghx0dtvut60eef6vcch",
		"terra1y246m036et7vu69nsg4kapelj0tywe8vsmp34d",
		"terra1j39c9sjr0zpjnrfjtthuua0euecv7txavxvq36",
		"terra1t0jthtq9zhm4ldtvs9epp02zp23f355wu6zrzq",
		"terra12dxclvqrgt7w3s7gtwpdkxgymexv8stgqcr0yu",
		"terra1az3dsad74pwhylrrexnn5qylzj783uyww2s7xz",
		"terra1ttq26dq4egr5exmhd6gezerrxhlutx9u90uncn",
		"terra13e9670yuvfs06hctt9pmgjnz0yw28p0wgnhrqn",
		"terra1skmktm537pfaycgu9jx4fqryjt6pf77ycpesw0",
		"terra14q8cazgt58y2xkd26mlukemwth0cnvfqmgz2qk",
		"terra163vzxz9wwy320ccwy73qe6h33yzg2yhyvv5nsf",
		"terra1kj43wfnvrgc2ep94dgmwvnzv8vnkkxrxmrnhkp",
		"terra1gu6re549pn0mdpshtv75t3xugn347jghlhul73",
		"terra1gft3qujlq04yza3s2r238mql2yn3xxqepzt2up",
		"terra174pe7qe7g867spzdfs5f4rf9fuwmm42zf4hykf",
		"terra1ju68sg6k39t385sa0fazqvjgh6m6gkmsmp4lln",
		"terra1dlh7k4hcnsrvlfuzhdzx3ctynj7s8dde9zmdyd",
		"terra18wcdhpzpteharlkks5n6k7ent0mjyftvcpm6ee",
		"terra1xmkwsauuk3kafua9k23hrkfr76gxmwdfq5c09d",
		"terra1t957gces65xd6p8g4cuqnyd0sy5tzku59njydd",
		"terra1s4rd0y5e4gasf0krdm2w8sjhsmh030m74f2x9v",
		"terra15jya6ugxp65y80y5h82k4gv90pd7acv58xp6jj",
		"terra14yqy9warjkxyecda5kf5a68qlknf4ve4sh7sa6",
		"terra1yxras4z0fs9ugsg2hew9334k65uzejwcslyx0y",
		"terra1p0vl4s4gp46vy6dm352s2fgtw6hccypph7zc3u",
		"terra1hhj92twle9x8rjkr3yffujexsy5ldexak5rglz",
		"terra18vnrzlzm2c4xfsx382pj2xndqtt00rvhu24sqe",
	}
)

var _ paramstypes.ParamSet = &Params{}

// DefaultParams creates default treasury module parameters
func DefaultParams() Params {
	return Params{
		TaxPolicy:               DefaultTaxPolicy,
		RewardPolicy:            DefaultRewardPolicy,
		SeigniorageBurdenTarget: DefaultSeigniorageBurdenTarget,
		MiningIncrement:         DefaultMiningIncrement,
		WindowShort:             DefaultWindowShort,
		WindowLong:              DefaultWindowLong,
		WindowProbation:         DefaultWindowProbation,
		BurnTaxSplit:            DefaultBurnTaxSplit,
		BurnTaxWhitelist:		 DefaultBurnTaxWhitelist,
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
// pairs of treasury module's parameters.

func (p *Params) ParamSetPairs() paramstypes.ParamSetPairs {
	return paramstypes.ParamSetPairs{
		paramstypes.NewParamSetPair(KeyTaxPolicy, &p.TaxPolicy, validateTaxPolicy),
		paramstypes.NewParamSetPair(KeyRewardPolicy, &p.RewardPolicy, validateRewardPolicy),
		paramstypes.NewParamSetPair(KeySeigniorageBurdenTarget, &p.SeigniorageBurdenTarget, validateSeigniorageBurdenTarget),
		paramstypes.NewParamSetPair(KeyMiningIncrement, &p.MiningIncrement, validateMiningIncrement),
		paramstypes.NewParamSetPair(KeyWindowShort, &p.WindowShort, validateWindowShort),
		paramstypes.NewParamSetPair(KeyWindowLong, &p.WindowLong, validateWindowLong),
		paramstypes.NewParamSetPair(KeyWindowProbation, &p.WindowProbation, validateWindowProbation),
		paramstypes.NewParamSetPair(KeyBurnTaxSplit, &p.BurnTaxSplit, validateBurnTaxSplit),
		paramstypes.NewParamSetPair(KeyBurnTaxWhitelist, &p.BurnTaxWhitelist, validateBurnTaxWhitelist),
	}
}

// Validate performs basic validation on treasury parameters.
func (p Params) Validate() error {
	if p.TaxPolicy.RateMax.LT(p.TaxPolicy.RateMin) {
		return fmt.Errorf("treasury TaxPolicy.RateMax %s must be greater than TaxPolicy.RateMin %s",
			p.TaxPolicy.RateMax, p.TaxPolicy.RateMin)
	}

	if p.TaxPolicy.RateMin.IsNegative() {
		return fmt.Errorf("treasury parameter TaxPolicy.RateMin must be zero or positive: %s", p.TaxPolicy.RateMin)
	}

	if !p.TaxPolicy.Cap.IsValid() {
		return fmt.Errorf("treasury parameter TaxPolicy.Cap is invalid")
	}

	if p.TaxPolicy.ChangeRateMax.IsNegative() {
		return fmt.Errorf("treasury parameter TaxPolicy.ChangeRateMax must be positive: %s", p.TaxPolicy.ChangeRateMax)
	}

	if p.RewardPolicy.RateMax.LT(p.RewardPolicy.RateMin) {
		return fmt.Errorf("treasury RewardPolicy.RateMax %s must be greater than RewardPolicy.RateMin %s",
			p.RewardPolicy.RateMax, p.RewardPolicy.RateMin)
	}

	if p.RewardPolicy.RateMin.IsNegative() {
		return fmt.Errorf("treasury parameter RewardPolicy.RateMin must be positive: %s", p.RewardPolicy.RateMin)
	}

	if p.RewardPolicy.ChangeRateMax.IsNegative() {
		return fmt.Errorf("treasury parameter RewardPolicy.ChangeRateMax must be positive: %s", p.RewardPolicy.ChangeRateMax)
	}

	if p.SeigniorageBurdenTarget.IsNegative() {
		return fmt.Errorf("treasury parameter SeigniorageBurdenTarget must be positive: %s", p.SeigniorageBurdenTarget)
	}

	if p.MiningIncrement.IsNegative() {
		return fmt.Errorf("treasury parameter MiningIncrement must be positive: %s", p.MiningIncrement)
	}

	if p.WindowLong <= p.WindowShort {
		return fmt.Errorf("treasury parameter WindowLong must be bigger than WindowShort: (%d, %d)", p.WindowLong, p.WindowShort)
	}

	return nil
}

func validateTaxPolicy(i interface{}) error {
	v, ok := i.(PolicyConstraints)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.RateMin.IsNegative() {
		return fmt.Errorf("rate min must be positive: %s", v)
	}

	if v.RateMax.LT(v.RateMin) {
		return fmt.Errorf("rate max must be bigger than rate min: %s", v)
	}

	if !v.Cap.IsValid() {
		return fmt.Errorf("cap is invalid: %s", v)
	}

	if v.ChangeRateMax.IsNegative() {
		return fmt.Errorf("max change rate must be positive: %s", v)
	}

	return nil
}

func validateRewardPolicy(i interface{}) error {
	v, ok := i.(PolicyConstraints)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.RateMin.IsNegative() {
		return fmt.Errorf("rate min must be positive: %s", v)
	}

	if v.RateMax.LT(v.RateMin) {
		return fmt.Errorf("rate max must be bigger than rate min: %s", v)
	}

	if v.ChangeRateMax.IsNegative() {
		return fmt.Errorf("max change rate must be positive: %s", v)
	}

	return nil
}

func validateSeigniorageBurdenTarget(i interface{}) error {
	v, ok := i.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNegative() {
		return fmt.Errorf("seigniorage burden target must be positive: %s", v)
	}

	return nil
}

func validateMiningIncrement(i interface{}) error {
	v, ok := i.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNegative() {
		return fmt.Errorf("mining increment must be positive: %s", v)
	}

	return nil
}

func validateWindowShort(i interface{}) error {
	_, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return nil
}

func validateWindowLong(i interface{}) error {
	_, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return nil
}

func validateWindowProbation(i interface{}) error {
	_, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return nil
}

func validateBurnTaxSplit(i interface{}) error {
	v, ok := i.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNegative() {
		return fmt.Errorf("burn tax split must be positive: %s", v)
	}

	if v.GTE(sdk.NewDec(1)) {
		return fmt.Errorf("burn tax split can not greater than 1")
	}

	return nil
}

func validateBurnTaxWhitelist(i interface{}) error {

	for _, addr := range(i.( []string )) {
		_, err := sdk.AccAddressFromBech32(addr)
		if err != nil {
			return err
		}
	}

	return nil

}
