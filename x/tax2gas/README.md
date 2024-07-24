# Tax2gas

## Testcases

- Normal tx success
- Not supported tx will not be deducted tax amount
- Special IBC tx will be bypass when gas usage is not exceeded
- Forward tx should minus the amount to tx origin
- Multiple forward works
- Error forward tx should return the fund
- Out of gas should return the tax and not consumed gas
- Multiple msg types should works
- Grant msg should work
- Allow pay with multiple fees should work

| No | Name | Scenario | Expect Result | Covered by |
|----|----------|-------------------|---------------|------------|
| 1 | Normal transaction should success | User transfer or make some special transactions which send coins to different address | Tax should be deducted with correct amount| [TestFeeTax](../../tests/e2e/e2e_test.go#L108) <br> [TestFeeTaxWasm](../../tests/e2e/e2e_test.go#L157)|
| 2 | Not supported tx will not be deducted tax amount | User transfer or make some special transactions that not in the tax list | Tax shouldn't be deducted with correct amount| |
| 3 | Special IBC tx will be bypass when gas limit is not exceeded | User make IBC transactions that happen both cases:  <br> - Gas usage does not exceeded `maxTotalBypassMinFeeMsgGasUsage`  <br> -Gas usage exceeded `maxTotalBypassMinFeeMsgGasUsage` | Bypass when gas limit not exceeded and deduct fee when exceed | |
| 4 | Forward transaction should deduct the amount to tx origin | User execute contract that will trigger an execute msg to another contract | - User should be the tx origin of the execute msg<br>- Tax should be deducted with correct amount | |
| 5 | Multiple forward works | Contracts will trigger another contracts multiple times | - User should be the tx origin of the execute msg<br>- Tax should be deducted with correct amount | |
| 6 | Error forward tx should return the tax and not consumed gas | User execute contract that will trigger an execute msg to another contract. The execute msg to another contract will be failed | Tax and not consumed gas should be revert to user | |
| 7 | Out of gas should return the tax and not consumed gas | User make some transactions with limited gas amount that will lead to cause `out of gas` error | Tax and not consumed gas should be revert to user | 🛑 Not figure out the way to make `out of gas` error occur, should be test in testnet  |
| 8 | Multiple msg types should works | - Test multiple tx type that's in the tax list or not <br>- Contract dispatch multiple tx types | Chain will deduct tax with correct amount | |
| 9 | Grant msg should work | User grant multiple type of permissions to different transactions | Grant permission msg will only can deduct with one denom | [TestFeeTaxGrant](../../tests/e2e/e2e_test.go#L212) |
| 10 | Allow pay with multiple fees should work | User make transaction with multiple coins as fee | Fee can be paid by multiple denom, if one denom is not enough, then it will deduct other denom | |
