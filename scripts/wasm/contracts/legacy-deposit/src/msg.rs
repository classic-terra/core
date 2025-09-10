use cosmwasm_std::{Addr, Uint128};
use schemars::JsonSchema;
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct InstantiateMsg {}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub enum ExecuteMsg {
    /// Deposit all native coins attached to this message into sender's balances
    Deposit {},

    /// Withdraw requested amount of a native denom from sender's balance.
    /// The contract will send `amount` and the chain will deduct tax,
    /// so the receiver gets `amount - min(amount*rate, cap)`.
    Withdraw { denom: String, amount: Uint128 },

    /// (Optional) Withdraw all of one denom
    WithdrawAll { denom: String },
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub enum QueryMsg {
    Balance { address: String, denom: String },
    AllBalances { address: String },
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct BalanceResponse {
    pub address: Addr,
    pub denom: String,
    pub amount: Uint128,
}

#[derive(Serialize, Deserialize, Clone, Debug, PartialEq, JsonSchema)]
pub struct AllBalancesResponse {
    pub address: Addr,
    pub balances: Vec<(String, Uint128)>, // (denom, amount)
}
