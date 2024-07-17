# Tax2gas

## Testcases

- Normal tx success
- Forward tx should minus the amount to tx origin
- Multiple forward works
- Out of gas should return the tax and not consumed gas
- Error forward tx should return the fund
- Multiple msg types should works
- Grant msg should work for multiple contracts crosscall
- Allow pay with multiple fees should work

| No | Name | Scenario | Expect Result | Covered by |
|----|----------|-------------------|---------------|------------|
| 1 | Normal transaction should success | User transfer or make some special transactions which send coins to different address | Tax should be deducted with correct amount| |
| 2 | Forward transaction should deduct the amount to tx origin | User execute contract that will trigger an execute msg to another contract | - User should be the tx origin of the execute msg<br>- Tax should be deducted with correct amount | |
| 3 | Multiple forward works | Contracts will trigger another contracts multiple times | - User should be the tx origin of the execute msg<br>- Tax should be deducted with correct amount | |
| 4 | Out of gas should return the tax and not consumed gas | User make some transactions with limited gas amount that will lead to cause `out of gas` error | Tax and not consumed gas should be revert to user | |
| 5 | Error forward tx should return the tax and not consumed gas | User execute contract that will trigger an execute msg to another contract. The execute msg to another contract will be failed | Tax and not consumed gas should be revert to user | |
| 6 | Multiple msg types should works | 