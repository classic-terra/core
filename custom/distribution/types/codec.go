package types

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"

	govtypes "github.com/classic-terra/core/v3/custom/gov/types"
)

// Wrapper type for backward compatibility
type LegacyMsgSetWithdrawAddress struct {
	types.MsgSetWithdrawAddress
}

// Wrapper type for backward compatibility
type LegacyMsgFundCommunityPool struct {
	types.MsgFundCommunityPool
}

// Wrapper type for backward compatibility
type LegacyMsgWithdrawDelegatorReward struct {
	types.MsgWithdrawDelegatorReward
}

// Wrapper type for backward compatibility
type LegacyMsgWithdrawValidatorCommission struct {
	types.MsgWithdrawValidatorCommission
}

// Wrapper type for backward compatibility
type LegacyCommunityPoolSpendProposal struct {
	types.CommunityPoolSpendProposal
}

func (l LegacyMsgSetWithdrawAddress) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.MsgSetWithdrawAddress)
}

func (l *LegacyMsgSetWithdrawAddress) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.MsgSetWithdrawAddress)
}

func (l LegacyMsgFundCommunityPool) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.MsgFundCommunityPool)
}

func (l *LegacyMsgFundCommunityPool) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.MsgFundCommunityPool)
}

func (l LegacyMsgWithdrawDelegatorReward) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.MsgWithdrawDelegatorReward)
}

func (l *LegacyMsgWithdrawDelegatorReward) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.MsgWithdrawDelegatorReward)
}

func (l LegacyMsgWithdrawValidatorCommission) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.MsgWithdrawValidatorCommission)
}

func (l *LegacyMsgWithdrawValidatorCommission) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.MsgWithdrawValidatorCommission)
}

func (l LegacyCommunityPoolSpendProposal) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.CommunityPoolSpendProposal)
}

func (l *LegacyCommunityPoolSpendProposal) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &l.CommunityPoolSpendProposal)
}

// RegisterLegacyAminoCodec registers the necessary x/distribution interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	types.RegisterLegacyAminoCodec(cdc)

	// distribution/MsgWithdrawDelegationReward and distribution/MsgWithdrawValidatorCommission
	// will not be supported by Ledger signing due to the overflow of length of the name
	cdc.RegisterConcrete(&LegacyMsgWithdrawDelegatorReward{}, "distribution/MsgWithdrawDelegationReward", nil)
	cdc.RegisterConcrete(&LegacyMsgWithdrawValidatorCommission{}, "distribution/MsgWithdrawValidatorCommission", nil)
	legacy.RegisterAminoMsg(cdc, &LegacyMsgSetWithdrawAddress{}, "distribution/MsgModifyWithdrawAddress")
	legacy.RegisterAminoMsg(cdc, &LegacyMsgFundCommunityPool{}, "distribution/MsgFundCommunityPool")
	cdc.RegisterConcrete(&types.CommunityPoolSpendProposal{}, "distribution/CommunityPoolSpendProposal", nil)
}

var (
	amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewAminoCodec(amino)
)

func init() {
	RegisterLegacyAminoCodec(amino)
	cryptocodec.RegisterCrypto(amino)
	amino.Seal()

	govtypes.RegisterProposalTypeCodec(types.CommunityPoolSpendProposal{}, "distribution/CommunityPoolSpendProposal")
}
