package cli

// import (
// 	"encoding/json"
// 	"os"

// 	"github.com/cosmos/cosmos-sdk/codec"
// 	sdk "github.com/cosmos/cosmos-sdk/types"
// )

// // proposal defines the new Msg-based proposal.
// type proposal struct {
// 	// Msgs defines an array of sdk.Msgs proto-JSON-encoded as Anys.
// 	Messages []json.RawMessage `json:"messages,omitempty"`
// 	Metadata string            `json:"metadata"`
// 	Deposit  string            `json:"deposit"`
// 	Title    string            `json:"title"`
// 	Summary  string            `json:"summary"`
// }

// // parseSubmitProposal reads and parses the proposal.
// func parseSubmitProposal(cdc codec.Codec, path string) ([]sdk.Msg, string, string, string, sdk.Coins, error) {
// 	var proposal proposal

// 	contents, err := os.ReadFile(path)
// 	if err != nil {
// 		return nil, "", "", "", nil, err
// 	}

// 	err = json.Unmarshal(contents, &proposal)
// 	if err != nil {
// 		return nil, "", "", "", nil, err
// 	}

// 	msgs := make([]sdk.Msg, len(proposal.Messages))
// 	for i, anyJSON := range proposal.Messages {
// 		var msg sdk.Msg
// 		err := cdc.UnmarshalInterfaceJSON(anyJSON, &msg)
// 		if err != nil {
// 			return nil, "", "", "", nil, err
// 		}

// 		msgs[i] = msg
// 	}

// 	deposit, err := sdk.ParseCoinsNormalized(proposal.Deposit)
// 	if err != nil {
// 		return nil, "", "", "", nil, err
// 	}

// 	return msgs, proposal.Metadata, proposal.Title, proposal.Summary, deposit, nil
// }
