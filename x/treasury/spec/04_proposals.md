## Proposals

The Treasury module defines special proposals which allow the [Tax Rate](./02_state.md#TaxRate) and [Reward Weight](./02_state.md#RewardWeight) values in the `KVStore` to be voted on and changed accordingly, subject to the [policy constraints](./03_end_block.md#PolicyConstraints) imposed by `pc.Clamp()`.

The treasury module will define two proposals to add or remove tax exemption list. Transaction among addresses in tax exemption list will not be taxed.

### TaxRateUpdateProposal

```go
type TaxRateUpdateProposal struct {
	Title       string  // Title of the Proposal
	Description string  // Description of the Proposal
	TaxRate     sdk.Dec // target TaxRate
}
```

::: details JSON Example

```json
{
  "type": "treasury/TaxRateUpdateProposal",
  "value": {
    "title": "proposal title",
    "description": "proposal description",
    "tax_rate": "0.001000000000000000"
  }
}
```

### RewardWeightUpdateProposal

```go
type RewardWeightUpdateProposal struct {
	Title        string  // Title of the Proposal
	Description  string  // Description of the Proposal
	RewardWeight sdk.Dec // target RewardWeight
}
```

::: details JSON Example

```json
{
  "type": "treasury/RewardWeightUpdateProposal",
  "value": {
    "title": "proposal title",
    "description": "proposal description",
    "reward_weight": "0.001000000000000000"
  }
}
```

### AddBurnTaxExemptionAddressProposal / RemoveBurnTaxExemptionAddressProposal (removed)

These proposals were removed in upgrade `v14_5` (core v4.2.0). They are no
longer routed by `x/gov`; submitting them fails with `no handler exists`. The
legacy burn tax exemption list they modified had already been migrated into the
`Binance` zone of `x/taxexemption` in `v12` and is deleted in `v14_5`.

The proto types remain registered so that historical proposals stored in
`x/gov` can still be decoded. Tax exemptions are managed through the
`x/taxexemption` governance messages (`MsgAddTaxExemptionZone`,
`MsgModifyTaxExemptionZone`, `MsgRemoveTaxExemptionZone`,
`MsgAddTaxExemptionAddress`, `MsgRemoveTaxExemptionAddress`).
