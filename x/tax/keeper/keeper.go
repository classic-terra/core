package keeper

import (
	"fmt"

	slog "cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"

	"github.com/classic-terra/core/v3/x/tax/types"
	treasurykeeper "github.com/classic-terra/core/v3/x/treasury/keeper"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	distributionKeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
)

type Keeper struct {
	storeKey storetypes.StoreKey
	cdc      codec.BinaryCodec

	bankKeeper         bankkeeper.Keeper
	treasuryKeeper     treasurykeeper.Keeper
	distributionKeeper distributionKeeper.Keeper

	// the address capable of executing a MsgUpdateParams message. Typically, this
	// should be the x/gov module account.
	authority string
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	bankKeeper bankkeeper.Keeper,
	treasuryKeeper treasurykeeper.Keeper,
	distributionKeeper distributionKeeper.Keeper,
	authority string,
) Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Errorf("invalid bank authority address: %w", err))
	}

	return Keeper{cdc: cdc, storeKey: storeKey, bankKeeper: bankKeeper, treasuryKeeper: treasuryKeeper, distributionKeeper: distributionKeeper, authority: authority}
}

// InitGenesis initializes the tax module's state from a provided genesis
// state.
func (k Keeper) InitGenesis(ctx sdk.Context, genState *types.GenesisState) {
	if err := genState.Validate(); err != nil {
		panic(err)
	}

	k.SetParams(ctx, genState.Params)
}

// ExportGenesis returns the tax module's exported genesis.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	return &types.GenesisState{
		Params: k.GetParams(ctx),
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) slog.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetAuthority returns the x/tax module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

func (k Keeper) GetGasPrices(ctx sdk.Context) sdk.DecCoins {
	return k.GetParams(ctx).GasPrices.Sort()
}

func (k Keeper) GetBurnTaxRate(ctx sdk.Context) sdkmath.LegacyDec {
	return k.GetParams(ctx).BurnTaxRate
}

func (k Keeper) ComputeTax(ctx sdk.Context, amount sdk.Coins) sdk.Coins {
	// use unified tax computation logic; simulate=false in keeper context
	return types.ComputeTaxes(ctx, amount, k.GetBurnTaxRate(ctx), false, k.treasuryKeeper)
}

// DeductTax deducts tax from the sender and processes tax splits
// If it was not yet paid in the current block
func (k Keeper) DeductTax(
	ctx sdk.Context,
	sender sdk.AccAddress,
	amount sdk.Coins,
	skipDeduct bool,
) (sdk.Coins, error) {
	ctx.Logger().Info("Deducting tax", "sender", sender, "amount", amount, ctx.Value(types.ContextKeyTaxReverseCharge))

	if !k.IsReverseCharge(ctx, false) {
		// do not emit here as we hopefully did it earlier already
		return amount, nil
	}

	taxes := k.ComputeTax(ctx, amount)
	netAmount := amount.Sub(taxes...)

	if !taxes.IsZero() && !skipDeduct {
		// Deduct the total tax amount from the sender and send to FeeCollector
		if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, sender, authtypes.FeeCollectorName, taxes); err != nil {
			return nil, err
		}

		// Process tax splits (burn, oracle, community)
		if err := k.ProcessTaxSplits(ctx, taxes); err != nil {
			return nil, err
		}

		// Record tax proceeds
		k.treasuryKeeper.RecordEpochTaxProceeds(ctx, taxes)
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				types.EventTypeTax,
				sdk.NewAttribute(types.AttributeKeyReverseCharge, types.AttributeValueReverseCharge),
				sdk.NewAttribute(types.AttributeKeyTaxAmount, taxes.String()),
			),
		)
	}

	return netAmount, nil
}

func (k Keeper) GetEffectiveGasPrices(ctx sdk.Context) sdk.DecCoins {
	minGasPrices := ctx.MinGasPrices()
	taxGasPrices := k.GetGasPrices(ctx)
	if taxGasPrices.IsZero() {
		return minGasPrices
	}

	gasPrices := make(sdk.DecCoins, len(taxGasPrices))

	for i, gasPrice := range taxGasPrices {
		minAmt := minGasPrices.AmountOf(gasPrice.Denom)
		maxAmt := gasPrice.Amount
		if minAmt.GT(maxAmt) {
			maxAmt = minAmt
		}
		gasPrices[i] = sdk.DecCoin{Denom: gasPrice.Denom, Amount: maxAmt}
	}

	return gasPrices
}

func (k Keeper) GetGasPriceForDenom(ctx sdk.Context, denom string) sdkmath.LegacyDec {
	for _, gasPrice := range k.GetGasPrices(ctx) {
		if gasPrice.Denom == denom {
			return gasPrice.Amount
		}
	}

	return sdkmath.LegacyZeroDec()
}

func (k Keeper) IsReverseCharge(ctx sdk.Context, emit bool) bool {
	if !ctx.Value(types.ContextKeyTaxReverseCharge).(bool) {
		if emit {
			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					types.EventTypeTax,
					sdk.NewAttribute(types.AttributeKeyReverseCharge, types.AttributeValueNoReverseCharge),
				),
			)
		}
		return false
	}

	return true
}
