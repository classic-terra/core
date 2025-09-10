use cosmwasm_std::{Addr, Uint128};
use cosmwasm_storage::{bucket, bucket_read, Bucket, ReadonlyBucket};

pub static KEY_BALANCES: &[u8] = b"balances";

pub fn balances<'a>(storage: &'a mut dyn cosmwasm_std::Storage) -> Bucket<'a, Uint128> {
    bucket(storage, KEY_BALANCES)
}
pub fn balances_read<'a>(storage: &'a dyn cosmwasm_std::Storage) -> ReadonlyBucket<'a, Uint128> {
    bucket_read(storage, KEY_BALANCES)
}

pub fn bal_key(addr: &Addr, denom: &str) -> Vec<u8> {
    let mut k = addr.as_str().as_bytes().to_vec();
    k.push(0x00);
    k.extend_from_slice(denom.as_bytes());
    k
}
