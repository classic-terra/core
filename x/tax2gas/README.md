# Tax2gas

## Testcases

- Normal tx success
- Not supported tx will not be deducted tax amount
- Special IBC tx will be bypass when gas usage is not exceeded
- Forward tx should minus the amount to tx origin
- Multiple forward works
- Error forward tx should return the fund
- Out of gas should return the tax and not consumed gas
- Grant msg should work
- Allow pay with multiple fees should work
- Try to pay with non value token denom should fail

| No | Name | Scenario | Expect Result | Covered by |
|----|----------|-------------------|---------------|------------|
| 1 | Normal transaction should success | User transfer or make some special transactions which send coins to different address | Tax should be deducted with correct amount| [TestFeeTax](../../tests/e2e/e2e_test.go#L108) <br> [TestFeeTaxWasm](../../tests/e2e/e2e_test.go#L158)|
| 2 | Not supported tx will not be deducted tax amount | User transfer or make some special transactions that not in the tax list | Tax shouldn't be deducted with correct amount| [TestFeeTaxNotSupport](../../tests/e2e/e2e_test.go#L306) |
| 3 | Special IBC tx will be bypass when gas limit is not exceeded | User make IBC transactions that happen both cases:  <br> - Gas usage does not exceeded `maxTotalBypassMinFeeMsgGasUsage`  <br> -Gas usage exceeded `maxTotalBypassMinFeeMsgGasUsage` | Bypass when gas limit not exceeded and deduct fee when exceed | 🛑 Not figure out the way to make update client in e2e, should be test in testnet |
| 4 | Forward transaction should deduct the amount to tx origin | User execute contract that will trigger an execute msg to another contract | - User should be the tx origin of the execute msg<br>- Tax should be deducted with correct amount | [TestFeeTaxForwardWasm](../../tests/e2e/e2e_test.go#L428) |
| 5 | Multiple forward works | Contracts will trigger another contracts multiple times | - User should be the tx origin of the execute msg<br>- Tax should be deducted with correct amount | [TestFeeTaxForwardWasm](../../tests/e2e/e2e_test.go#L428) |
| 6 | Error forward tx should return the tax and not consumed gas | User execute contract that will trigger an execute msg to another contract. The execute msg to another contract will be failed | Tax and not consumed gas should be revert to user | [TestFeeTaxForwardWasm](../../tests/e2e/e2e_test.go#L428) |
| 7 | Out of gas should return the tax and not consumed gas | User make some transactions with limited gas amount that will lead to cause `out of gas` error | Tax and not consumed gas should be revert to user | 🛑 Not figure out the way to make `out of gas` error occur, should be test in testnet  |
| 8 | Grant msg should work | User grant multiple type of permissions to different transactions | Grant permission msg will only can deduct one denom in ante handler and one denom in post hanlder | [TestFeeTaxGrant](../../tests/e2e/e2e_test.go#L214) |
| 9 | Allow pay with multiple fees should work | User make transaction with multiple coins as fee | Fee can be paid by multiple denom, if one denom is not enough, then it will deduct other denom |  [TestFeeTaxMultipleDenoms](../../tests/e2e/e2e_test.go#L380) |
| 10 | Try to pay with non value token denom should fail | User make transaction that use a different denom as fee | That denom should be reject and the tx should only accept denom listed in params | [TestFeeTaxNotAcceptDenom](../../tests/e2e/e2e_test.go#L531) |