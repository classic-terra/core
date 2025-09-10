use cosmwasm_std::{
    attr, to_binary, entry_point, Addr, BankMsg, Coin, Deps, DepsMut, Env, MessageInfo, Order,
    Response, StdError, StdResult, Uint128,
};
use terra_cosmwasm::{TerraMsgWrapper, TerraQuerier};
use crate::msg::{AllBalancesResponse, BalanceResponse, ExecuteMsg, InstantiateMsg, QueryMsg};
use crate::state::{bal_key, balances, balances_read};

const CONTRACT_NAME: &str = "crates.io:terra_vault_016";
const CONTRACT_VERSION: &str = "0.1.0";

#[entry_point]
pub fn instantiate(
    _deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    _msg: InstantiateMsg,
) -> StdResult<Response<TerraMsgWrapper>> {
    Ok(Response::new()
        .add_attribute("method", "instantiate")
        .add_attribute("contract", CONTRACT_NAME)
        .add_attribute("version", CONTRACT_VERSION))
}

#[entry_point]
pub fn execute(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> StdResult<Response<TerraMsgWrapper>> {
    match msg {
        ExecuteMsg::Deposit {} => execute_deposit(deps, env, info),
        ExecuteMsg::Withdraw { denom, amount } => execute_withdraw(deps, env, info, denom, amount),
        ExecuteMsg::WithdrawAll { denom } => {
            // Read using storage directly to avoid Deps/DepsMut mismatch
            let bal = read_balance(&*deps.storage, &info.sender, &denom)?;
            execute_withdraw(deps, env, info, denom, bal)
        }
    }
}

fn execute_deposit(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
) -> StdResult<Response<TerraMsgWrapper>> {
    if info.funds.is_empty() {
        return Err(StdError::generic_err("No funds sent"));
    }

    let sender = info.sender.clone();
    let mut store = balances(deps.storage);

    for Coin { denom, amount } in info.funds {
        let key = bal_key(&sender, &denom);
        let prev = store.may_load(&key)?.unwrap_or_default();
        store.save(&key, &(prev + amount))?;
    }

    Ok(Response::new()
        .add_attribute("action", "deposit")
        .add_attribute("from", sender.to_string()))
}

fn execute_withdraw(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    denom: String,
    amount: Uint128,
) -> StdResult<Response<TerraMsgWrapper>> {
    if amount.is_zero() {
        return Err(StdError::generic_err("amount must be > 0"));
    }
    let sender = info.sender.clone();

    // ✅ compute tax first (immutable borrow), before opening a mutable store borrow
    let tax = compute_tax(deps.as_ref(), &denom, amount)?;
    let net_expected = amount.checked_sub(tax)
        .map_err(|_| StdError::generic_err("tax exceeds amount"))?;

    // ✅ now take the mutable borrow and mutate storage
    let mut store = balances(deps.storage);
    let key = bal_key(&sender, &denom);
    let prev = store.may_load(&key)?.unwrap_or_default();
    if prev < amount {
        return Err(StdError::generic_err("insufficient balance"));
    }
    store.save(&key, &(prev - amount))?;

    let send = BankMsg::Send {
        to_address: sender.to_string(),
        amount: vec![Coin { denom: denom.clone(), amount }],
    };

    Ok(Response::new()
        .add_message(send)
        .add_attributes(vec![
            attr("action", "withdraw"),
            attr("to", sender.to_string()),
            attr("denom", denom),
            attr("requested", amount.to_string()),
            attr("tax", tax.to_string()),
            attr("net_expected", net_expected.to_string()),
        ]))
}

#[entry_point]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<cosmwasm_std::Binary> {
    match msg {
        QueryMsg::Balance { address, denom } => {
            let addr = deps.api.addr_validate(&address)?;
            let amount = read_balance(deps.storage, &addr, &denom)?;
            to_binary(&BalanceResponse { address: addr, denom, amount })
        }
        QueryMsg::AllBalances { address } => {
            let addr = deps.api.addr_validate(&address)?;
            let balances = read_all_balances(deps.storage, &addr)?;
            to_binary(&AllBalancesResponse { address: addr, balances })
        }
    }
}

// ---- storage readers (take &dyn Storage so usable from Deps *and* DepsMut) ----

fn read_balance(storage: &dyn cosmwasm_std::Storage, addr: &Addr, denom: &str) -> StdResult<Uint128> {
    let r = balances_read(storage);
    Ok(r.may_load(&crate::state::bal_key(addr, denom))?.unwrap_or_default())
}

fn read_all_balances(storage: &dyn cosmwasm_std::Storage, addr: &Addr) -> StdResult<Vec<(String, Uint128)>> {
    let r = balances_read(storage);
    let mut prefix = addr.as_str().as_bytes().to_vec();
    prefix.push(0x00);

    let mut out = Vec::new();
    // range wants Option<&[u8]>, not an owned Vec
    for item in r.range(Some(&prefix), None, Order::Ascending) {
        let (k, v) = item?;
        if !k.starts_with(&prefix) { break; }
        let denom = String::from_utf8(k[prefix.len()..].to_vec())
            .map_err(|_| StdError::generic_err("invalid denom utf8"))?;
        out.push((denom, v));
    }
    Ok(out)
}

// ---- Tax helpers (CW-0.16 Decimal quirks handled) ----

/// tax = min(amount * rate, cap)
fn compute_tax(deps: Deps, denom: &str, amount: Uint128) -> StdResult<Uint128> {
    let q = TerraQuerier::new(&deps.querier);
    let rate = q.query_tax_rate()?.rate; // Decimal
    let cap = q.query_tax_cap(denom.to_string())?.cap; // Uint128

    // Decimal * Uint128 -> Uint128 (truncated) in 0.16
    let by_rate = rate * amount;
    Ok(by_rate.min(cap))
}

/// Given desired NET for receiver, compute GROSS to send so chain tax leaves `net`.
/// For 0.16 Decimal, use `one - rate` (no checked_sub).
#[allow(dead_code)]
fn gross_up_for_net(deps: Deps, denom: &str, net: Uint128) -> StdResult<Uint128> {
    // Fast path: zero tax
    let q = TerraQuerier::new(&deps.querier);
    let rate = q.query_tax_rate()?.rate;
    let cap = q.query_tax_cap(denom.to_string())?.cap;
    if rate.is_zero() {
        return Ok(net);
    }

    // Define a monotonic function f(g) = g - tax(g) - net.
    // We need the smallest g such that f(g) >= 0.
    // Upper bound: either rate path or cap path. cap path implies gross = net + cap.
    let mut low = net; // cannot be less than net
    let mut high = net + cap; // safe upper bound when cap binds

    // Expand high if rate path dominates and cap is small
    // Ensure high is sufficient so that net_high >= net
    // Limit iterations to avoid long loops; 64 is plenty for Uint128.
    for _ in 0..8 {
        let tax = compute_tax(deps, denom, high)?;
        let net_high = high.checked_sub(tax).map_err(|_| StdError::generic_err("tax > gross"))?;
        if net_high < net {
            // Double the high bound
            high = high + (high - low) + Uint128::new(1);
        } else {
            break;
        }
    }

    // Binary search for minimal gross producing at least `net`
    for _ in 0..64 {
        let mid = low + ((high - low) / Uint128::new(2));
        let tax = compute_tax(deps, denom, mid)?;
        let net_mid = mid.checked_sub(tax).map_err(|_| StdError::generic_err("tax > gross"))?;
        if net_mid < net {
            low = mid + Uint128::new(1);
        } else {
            high = mid;
        }
        if low >= high { break; }
    }
    Ok(high)
}
