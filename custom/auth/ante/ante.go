package ante

import (
	errorsmod "cosmossdk.io/errors"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
	distributionkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"

	dyncommante "github.com/classic-terra/core/v3/x/dyncomm/ante"
	dyncommkeeper "github.com/classic-terra/core/v3/x/dyncomm/keeper"
	tax2gasante "github.com/classic-terra/core/v3/x/tax2gas/ante"
	tax2gaskeeper "github.com/classic-terra/core/v3/x/tax2gas/keeper"
	tax2gastypes "github.com/classic-terra/core/v3/x/tax2gas/types"
	"github.com/cosmos/cosmos-sdk/codec"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	ibcante "github.com/cosmos/ibc-go/v7/modules/core/ante"
	ibckeeper "github.com/cosmos/ibc-go/v7/modules/core/keeper"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
)

// HandlerOptions are the options required for constructing a default SDK AnteHandler.
type HandlerOptions struct {
	AccountKeeper          ante.AccountKeeper
	BankKeeper             BankKeeper
	ExtensionOptionChecker ante.ExtensionOptionChecker
	FeegrantKeeper         tax2gastypes.FeegrantKeeper
	OracleKeeper           OracleKeeper
	TreasuryKeeper         TreasuryKeeper
	SignModeHandler        signing.SignModeHandler
	SigGasConsumer         ante.SignatureVerificationGasConsumer
	TxFeeChecker           ante.TxFeeChecker
	IBCKeeper              ibckeeper.Keeper
	WasmKeeper             *wasmkeeper.Keeper
	DistributionKeeper     distributionkeeper.Keeper
	GovKeeper              govkeeper.Keeper
	WasmConfig             *wasmtypes.WasmConfig
	TXCounterStoreKey      storetypes.StoreKey
	DyncommKeeper          dyncommkeeper.Keeper
	StakingKeeper          *stakingkeeper.Keeper
	Tax2Gaskeeper          tax2gaskeeper.Keeper
	Cdc                    codec.BinaryCodec
}

// NewAnteHandler returns an AnteHandler that checks and increments sequence
// numbers, checks signatures & account numbers, and deducts fees from the first
// signer.
func NewAnteHandler(options HandlerOptions) (sdk.AnteHandler, error) {
	if options.AccountKeeper == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "account keeper is required for ante builder")
	}

	if options.BankKeeper == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "bank keeper is required for ante builder")
	}

	if options.OracleKeeper == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "oracle keeper is required for ante builder")
	}

	if options.TreasuryKeeper == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "treasury keeper is required for ante builder")
	}

	if options.SignModeHandler == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "sign mode handler is required for ante builder")
	}

	if options.WasmConfig == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "wasm config is required for ante builder")
	}

	if options.TXCounterStoreKey == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "tx counter key is required for ante builder")
	}

	return sdk.ChainAnteDecorators(
		//ante.NewSetUpContextDecorator(), // outermost AnteDecorator. SetUpContext must be called first
		//wasmkeeper.NewLimitSimulationGasDecorator(options.WasmConfig.SimulationGasLimit),
		tax2gaskeeper.NewTax2GasDecorator(options.WasmConfig.SimulationGasLimit),
		wasmkeeper.NewCountTXDecorator(options.TXCounterStoreKey),
		ante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(options.AccountKeeper),
		// SpammingPreventionDecorator prevents spamming oracle vote tx attempts at same height
		NewSpammingPreventionDecorator(options.OracleKeeper),
		// MinInitialDepositDecorator prevents submitting governance proposal low initial deposit
		NewMinInitialDepositDecorator(options.GovKeeper, options.TreasuryKeeper),
		ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		tax2gasante.NewFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, options.TreasuryKeeper, options.Tax2Gaskeeper),
		dyncommante.NewDyncommDecorator(options.Cdc, options.DyncommKeeper, options.StakingKeeper),

		// Do not add any other decorators below this point unless explicitly explain.
		ante.NewSetPubKeyDecorator(options.AccountKeeper), // SetPubKeyDecorator must be called before all signature verification decorators
		ante.NewValidateSigCountDecorator(options.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
		ante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),
		ante.NewIncrementSequenceDecorator(options.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(&options.IBCKeeper),
	), nil
}
