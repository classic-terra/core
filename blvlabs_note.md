Proposal to improve the Gov Module Mechanism by BLV Labs (KYCed)

Enhancing the Oracle Module to Optimize the Calculation of Minimum Deposit for Proposals in the Gov Module
Problem:
As we know, the cryptocurrency market is always very volatile with large fluctuations, and so is #lunc, the price can also fluctuate with large amplitude, falling sharply or rising sharply

In the current gov module logic, when creating a proposal, the creator will have to deposit #lunc (5 million lunc) first to push the proposal to the voting stage. So what will happen if the lunc price drops sharply? or rises sharply? This leads to the value of the proposal is not guaranteed. When the price is too small, it can lead to bad guys being able to spam many proposals online.

So we decided to propose an improvement to the gov module to fix this problem:

Objective:
Use Oracle module to update and calculate the minimum deposit required to create a proposal in the Gov module.

In the event of a sharp drop or spike in LUNC price, the system will automatically increase the minimum margin to ensure the value of the proposal is maintained.

The proposal value is kept at $500 (as suggested by the community), which will change the required lunc amount.

Benefit:
This ensures that the required margin amount remains stable, unaffected by market price fluctuations, thus helping the network operate stably without being affected by price changes.

Additionally, this mechanism prevents the network from being spammed with proposals if the LUNC price drops too low, which could allow bad actors to flood the network with spam proposals.

Updated (28/8/2024)
After a quick check of v0.50, we decided to implement the logic as v0.50’s Canceling gov Proposal logic this helps avoid any conflicts if upgrading to v0.50

Updated (16/9/2024)
After listening to feedback from the community and validators, we have updated our proposal:

Focus on feature #1: Enhancing the Oracle Module to Optimize the Calculation of Minimum Deposit for Proposals in the Gov Module

Remove features #2 & #3, as this feature has been developed on SDK v.0.50

In the first proposal, we want to focus on small improvements and will complete well to demonstrate the team’s capabilities, as we are a new team on the Lunc Ecosystem We will continue to propose more complex improvements and features in the next proposals after completing this first proposal

We hope that the community will trust and support us with the common goal of Lunc blockchain continuing to develop.

Development Plan
Phase 1: Complete Enhancement 1 (*)
Research and delve into the Oracle and Gov modules.
Add supplementary API logic to the Oracle module to determine the price of LUNC.
Use the Oracle module API to add the minimum deposit calculation mechanism in the Gov module.
Complete the feature.
Conduct testing and write documentation.
Estimated Time: 4 weeks

Timeline: 4 weeks
Total Budget: $5,000
NOTED: This is a text proposal, not a community spend.  We will put another proposal up for funding once we complete the work listed above.