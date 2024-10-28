---
sidebar_position: 1
---

# `x/gov`
# Custom Governace Module Updates

## Concepts

*Disclaimer: This is work in progress. Mechanisms are susceptible to change.*

The governance process is divided in a few steps that are outlined below:
* **Proposal submission:** Proposal is submitted to the blockchain with a
  deposit.
* **Vote:** When the deposit for a proposal reaches a minimum threshold equivalent to
 500 USD (`MinUusdDeposit`), proposal is confirmed and vote opens. 
* **Execution** After a period of time, the votes are tallied and depending
  on the result, the messages in the proposal will be executed.
### Proposal submission

#### Right to submit a proposal

Every account can submit proposals by sending a `MsgSubmitProposal` transaction.
Once a proposal is submitted, it is identified by its unique `proposalID`.
#### Proposal Messages

A proposal includes an array of `sdk.Msg`s which are executed automatically if the
proposal passes. The messages are executed by the governance `ModuleAccount` itself. Modules
such as `x/upgrade`, that want to allow certain messages to be executed by governance
only should add a whitelist within the respective msg server, granting the governance
module the right to execute the message once a quorum has been reached. The governance
module uses the `MsgServiceRouter` to check that these messages are correctly constructed
and have a respective path to execute on but do not perform a full validity check.
### Deposit

To prevent spam, proposals must be submitted with a deposit in the coins defined by the equation `luncMinDepositAmountBasedOnUusd` = `MinUusdDeposit` / real-time price of LUNC at the time of proposal submission.
```keeper reference
github.com/classic-terra/core/v3/custom/gov/keeper/proposal.go#L136-L149
```
The value of `luncMinDepositAmountBasedOnUusd` will be stored in KVStores, using a key defined as `UUSDMinKeyPrefix|proposalID`, where the proposal ID is appended to the prefix, represented as a single byte.
```types reference
github.com/classic-terra/core/v3/custom/gov/types/keys.go#L20-L22
```

When a proposal is submitted, it has to be accompanied with a deposit that must be
strictly positive, but can be inferior to `luncMinDepositAmountBasedOnUusd`. The submitter doesn't need
to pay for the entire deposit on their own. The newly created proposal is stored in
an *inactive proposal queue* and stays there until its deposit passes the `luncMinDepositAmountBasedOnUusd`.
Other token holders can increase the proposal's deposit by sending a `Deposit`
transaction. If a proposal doesn't pass the `luncMinDepositAmountBasedOnUusd` before the deposit end time
(the time when deposits are no longer accepted), the proposal will be destroyed: the
proposal will be removed from state and the deposit will be burned (see x/gov `EndBlocker`).
When a proposal deposit passes the `luncMinDepositAmountBasedOnUusd` threshold (even during the proposal
submission) before the deposit end time, the proposal will be moved into the
*active proposal queue* and the voting period will begin.

The deposit is kept in escrow and held by the governance `ModuleAccount` until the
proposal is finalized (passed or rejected).
#### Deposit refund and burn

When a proposal is finalized, the coins from the deposit are either refunded or burned
according to the final tally of the proposal:

* If the proposal is approved or rejected but *not* vetoed, each deposit will be
  automatically refunded to its respective depositor (transferred from the governance
  `ModuleAccount`).
* When the proposal is vetoed with greater than 1/3, deposits will be burned from the
  governance `ModuleAccount` and the proposal information along with its deposit
  information will be removed from state.
* All refunded or burned deposits are removed from the state. Events are issued when
  burning or refunding a deposit.
### Vote

#### Voting period

Once a proposal reaches `luncMinDepositAmountBasedOnUusd`, it immediately enters `Voting period`. We
define `Voting period` as the interval between the moment the vote opens and
the moment the vote closes. `Voting period` should always be shorter than
`Unbonding period` to prevent double voting. The initial value of
`Voting period` is 2 weeks.

## Stores

:::note
Stores are KVStores in the multi-store. The key to find the store is the first parameter in the list
:::
We will use one KVStore `Governance` to store five mappings:

* A mapping from `proposalID|'proposal'` to `Proposal`.
* A mapping from `proposalID|'addresses'|address` to `Vote`. This mapping allows
  us to query all addresses that voted on the proposal along with their vote by
  doing a range query on `proposalID:addresses`.
* A mapping from `ParamsKey|'Params'` to `Params`. This map allows to query all 
  x/gov params.
* A mapping from `VotingPeriodProposalKeyPrefix|proposalID` to a single byte. This allows
  us to know if a proposal is in the voting period or not with very low gas cost.
* A mapping from `UUSDMinKeyPrefix|proposalID` to a single byte. This allows us to determine the minimum amount of LUNC that a specific proposal must deposit to pass the deposit phase.
  
For pseudocode purposes, here are the two function we will use to read or write in stores:

* `load(StoreKey, Key)`: Retrieve item stored at key `Key` in store found at key `StoreKey` in the multistore
* `store(StoreKey, Key, value)`: Write value `Value` at key `Key` in store found at key `StoreKey` in the multistore

## Parameters

The governance module contains the following parameters:

| Key                           | Type             | Example                                 |
|-------------------------------|------------------|-----------------------------------------|
| min_deposit       | array (coins)    | [{"denom":"uatom","amount":"10000000"}] |
| max_deposit_period            | string (time ns) | "172800000000000" (17280s)              |
| voting_period                 | string (time ns) | "172800000000000" (17280s)              |
| quorum                        | string (dec)     | "0.334000000000000000"                  |
| threshold                     | string (dec)     | "0.500000000000000000"                  |
| veto                          | string (dec)     | "0.334000000000000000"                  |
| burn_proposal_deposit_prevote | bool             | false                                   |
| burn_vote_quorum              | bool             | false                                   |
| burn_vote_veto                | bool             | true                                    |
| min_uusd_deposit              | coins            |{"denom":"uusd","amount":"500000"}       |
**NOTE**: 
Aiming to establish a clearer and more consistent minimum deposit requirement for governance proposals, the default value of `min_uusd_deposit` is currently set to 500 USD. However, this value can be updated by the community in the future. This approach allows the module to adapt to fluctuations in the value of LUNC over time, ensuring a consistent threshold for proposals.

## Client

### CLI

A user can query and interact with the `gov` module using the CLI.
We have added two new commands 

##### MinimalDeposit
The `MinimalDeposit` command enables users to query the minimum LUNC deposit required for a specific proposal, calculated based on the UUSD value. 


```bash
terrad q gov min-deposit [proposal-id] 
```
Example:

```bash
terrad q gov min-deposit 1 cosmos1..
```

##### CustomParams
The `CustomParams` command allows users to query the custom parameters of the module.


```bash
terrad q gov custom-params 
```
Example:

```bash
terrad q gov custom-params
```
## Future Improvements
We have successfully upgraded the chain by increasing the consensus version to 5. Moving forward,
 should the community require updates related to these matters, the BLV Team will be available to 
 assist with the implementation process.

BlvLas proposal: https://commonwealth.im/terra-luna-classic-lunc/discussion/24630-proposal-to-improve-the-gov-module-mechanism-by-blv-labs
