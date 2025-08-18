package types

// Tax module event types
const (
    EventTaxCommunity      = "tax_community"
    EventTaxOracle         = "tax_oracle"
    EventTaxMarketRedirect = "tax_market_redirect"
    EventTaxBurn           = "tax_burn"

    // Common attributes
    AttributeKeyAmount     = "amount"
    AttributeKeyFromModule = "from_module"
    AttributeKeyToModule   = "to_module"
    AttributeKeyHeight     = "height"
)
