# TaxExemption Module Flowchart

This diagram illustrates the structure and flow of the TaxExemption module.

```mermaid
flowchart TD
    subgraph TaxExemption Module
        A[Transaction Begins] --> B{Tax Exemption Check}
        B -->|Address is exempt| C[Skip Tax]
        B -->|Address is not exempt| D[Apply Tax]
        
        E[AddTaxExemptionZone] --> F[Store zone data in KVStore]
        G[RemoveTaxExemptionZone] --> H[Delete zone data from KVStore]
        I[ModifyTaxExemptionZone] --> J[Update zone data in KVStore]
        K[AddTaxExemptionAddress] --> L[Add address to zone in KVStore]
        M[RemoveTaxExemptionAddress] --> N[Remove address from zone in KVStore]
    end
    
    subgraph Zone Structure
        Z1[Zone] --> Z2[Name]
        Z1 --> Z3[Outgoing - Allow tax-free outgoing tx]
        Z1 --> Z4[Incoming - Allow tax-free incoming tx]
        Z1 --> Z5[CrossZone - Allow cross-zone tax-free tx]
    end
    
    subgraph IsExemptedFromTax Logic
        IE1[IsExemptedFromTax] --> IE2{Both addresses have zones?}
        IE2 -->|Yes| IE3{Same zone?}
        IE2 -->|No| IE4{Only sender has zone?}
        IE2 -->|No| IE5{Only recipient has zone?}
        IE2 -->|No| IE6[Not tax exempt]
        
        IE3 -->|Yes| IE7[Tax exempt]
        IE3 -->|No| IE8{Different zones rule check}
        
        IE8 -->|Sender: CrossZone & Outgoing| IE9[Tax exempt]
        IE8 -->|Recipient: CrossZone & Incoming| IE9
        IE8 -->|Otherwise| IE6
        
        IE4 -->|Sender zone has Outgoing| IE9
        IE4 -->|Otherwise| IE6
        
        IE5 -->|Recipient zone has Incoming| IE9
        IE5 -->|Otherwise| IE6
    end
    
    subgraph Transaction Processing
        TX1[Transaction with Transfer] --> TX2{Message Type?}
        TX2 -->|MsgSend| TX3[Check IsExemptedFromTax]
        TX2 -->|MsgMultiSend| TX4[Check IsExemptedFromTax for each input/output]
        TX2 -->|MsgExecuteContract| TX5[Check IsExemptedFromTax for sender/contract]
        TX2 -->|Other Messages| TX6[Apply specific rules]
        
        TX3 -->|Exempt| TX7[Skip tax]
        TX3 -->|Not exempt| TX8[Apply tax]
        
        TX4 -->|All inputs exempt| TX7
        TX4 -->|Some inputs not exempt| TX8
        
        TX5 -->|Exempt| TX7
        TX5 -->|Not exempt| TX8
    end
    
    subgraph Governance Control
        GOV1[Governance Proposal] --> GOV2{Proposal Type}
        GOV2 -->|AddTaxExemptionZone| E
        GOV2 -->|RemoveTaxExemptionZone| G
        GOV2 -->|ModifyTaxExemptionZone| I
        GOV2 -->|AddTaxExemptionAddress| K
        GOV2 -->|RemoveTaxExemptionAddress| M
    end
```

## TaxExemption Module Summary

The TaxExemption module enables specific addresses to be exempt from taxes in the blockchain network by organizing them into "zones" with configurable properties:

### Key Components

1. **Zone Structure**:
   - **Name**: Unique identifier for the zone
   - **Outgoing**: Allows tax-free transfers sent from addresses in this zone
   - **Incoming**: Allows tax-free transfers received by addresses in this zone
   - **CrossZone**: Allows tax-free transfers between different zones

2. **Core Functions**:
   - `AddTaxExemptionZone`: Creates a new zone with specific properties
   - `RemoveTaxExemptionZone`: Removes an existing zone
   - `ModifyTaxExemptionZone`: Updates properties of a zone
   - `AddTaxExemptionAddress`: Adds an address to a zone
   - `RemoveTaxExemptionAddress`: Removes an address from a zone
   - `IsExemptedFromTax`: Core logic that determines if a transaction is tax-exempt

3. **Tax Exemption Logic**:
   - The decision for tax exemption depends on whether the sender and recipient addresses belong to zones
   - Rules vary based on zone properties (outgoing, incoming, cross-zone)

4. **Governance Control**:
   - All zone and address modifications are controlled by governance
   - Only governance can add/remove/modify zones and addresses 