package types

// Market module event types
const (
	EventSwap = "swap"

	// Epoch processing events
	EventEpochBurn   = "epoch_burn"
	EventEpochRefill = "epoch_refill"

	AttributeKeyOffer     = "offer"
	AttributeKeyTrader    = "trader"
	AttributeKeyRecipient = "recipient"
	AttributeKeySwapCoin  = "swap_coin"
	AttributeKeySwapFee   = "swap_fee"

	// Common attributes
	AttributeKeyAmount     = "amount"
	AttributeKeyFromModule = "from_module"
	AttributeKeyToModule   = "to_module"
	AttributeKeyHeight     = "height"

	AttributeValueCategory = ModuleName
)
