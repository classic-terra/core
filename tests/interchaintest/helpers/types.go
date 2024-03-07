package helpers

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Go based data types for querying on the contract.
// Execute types are not needed here. We just use strings. Could add though in the future and to_string it

type PacketMetadata struct {
	Forward *ForwardMetadata `json:"forward"`
}

type ForwardMetadata struct {
	Receiver       string        `json:"receiver"`
	Port           string        `json:"port"`
	Channel        string        `json:"channel"`
	Timeout        time.Duration `json:"timeout"`
	Retries        *uint8        `json:"retries,omitempty"`
	Next           *string       `json:"next,omitempty"`
	RefundSequence *uint64       `json:"refund_sequence,omitempty"`
}

// EntryPoint
type QueryMsg struct {
	// Tokenfactory Core
	GetConfig      *struct{}            `json:"get_config,omitempty"`
	GetBalance     *GetBalanceQuery     `json:"get_balance,omitempty"`
	GetAllBalances *GetAllBalancesQuery `json:"get_all_balances,omitempty"`

	// Unity Contract
	GetWithdrawalReadyTime *struct{} `json:"get_withdrawal_ready_time,omitempty"`

	// IBCHooks
	GetCount      *GetCountQuery      `json:"get_count,omitempty"`
	GetTotalFunds *GetTotalFundsQuery `json:"get_total_funds,omitempty"`
}

type CodeInfo struct {
	CodeID string `json:"code_id"`
}
type CodeInfosResponse struct {
	CodeInfos []CodeInfo `json:"code_infos"`
}

type QueryContractResponse struct {
	Contracts []string `json:"contracts"`
}

type GetAllBalancesQuery struct {
	Address string `json:"address"`
}
type GetAllBalancesResponse struct {
	// or is it wasm Coin type?
	Data []sdk.Coin `json:"data"`
}

type GetBalanceQuery struct {
	// {"get_balance":{"address":"terra1...","denom":"factory/terra1.../RcqfWz"}}
	Address string `json:"address"`
	Denom   string `json:"denom"`
}
type GetBalanceResponse struct {
	// or is it wasm Coin type?
	Data sdk.Coin `json:"data"`
}

type WithdrawalTimestampResponse struct {
	// {"data":{"withdrawal_ready_timestamp":"1686146048614053435"}}
	Data *WithdrawalTimestampObj `json:"data"`
}
type WithdrawalTimestampObj struct {
	WithdrawalReadyTimestamp string `json:"withdrawal_ready_timestamp"`
}

type GetTotalFundsQuery struct {
	// {"get_total_funds":{"addr":"terra1..."}}
	Addr string `json:"addr"`
}
type GetTotalFundsResponse struct {
	// {"data":{"total_funds":[{"denom":"ibc/04F5F501207C3626A2C14BFEF654D51C2E0B8F7CA578AB8ED272A66FE4E48097","amount":"1"}]}}
	Data *GetTotalFundsObj `json:"data"`
}
type GetTotalFundsObj struct {
	TotalFunds []WasmCoin `json:"total_funds"`
}

type WasmCoin struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

type GetCountQuery struct {
	// {"get_total_funds":{"addr":"terra1..."}}
	Addr string `json:"addr"`
}
type GetCountResponse struct {
	// {"data":{"count":0}}
	Data *GetCountObj `json:"data"`
}
type GetCountObj struct {
	Count int64 `json:"count"`
}
