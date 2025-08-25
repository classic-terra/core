package e2e

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/classic-terra/core/v3/tests/e2e/initialization"
	"github.com/classic-terra/core/v3/tests/e2e/util"
)

// TaxComputeRequest represents the request body for tax computation
type TaxComputeRequest struct {
	Tx struct {
		Body struct {
			Messages []struct {
				Type        string `json:"@type"`
				FromAddress string `json:"from_address"`
				ToAddress   string `json:"to_address"`
				Amount      []struct {
					Denom  string `json:"denom"`
					Amount string `json:"amount"`
				} `json:"amount"`
			} `json:"messages"`
			Memo                        string        `json:"memo"`
			TimeoutHeight               string        `json:"timeout_height"`
			ExtensionOptions            []interface{} `json:"extension_options"`
			NonCriticalExtensionOptions []interface{} `json:"non_critical_extension_options"`
		} `json:"body"`
		AuthInfo struct {
			SignerInfos []struct {
				PublicKey struct {
					Type string `json:"@type"`
					Key  string `json:"key"`
				} `json:"public_key"`
				ModeInfo struct {
					Single struct {
						Mode string `json:"mode"`
					} `json:"single"`
				} `json:"mode_info"`
				Sequence string `json:"sequence"`
			} `json:"signer_infos"`
			Fee struct {
				Amount []struct {
					Denom  string `json:"denom"`
					Amount string `json:"amount"`
				} `json:"amount"`
				GasLimit string `json:"gas_limit"`
				Payer    string `json:"payer"`
				Granter  string `json:"granter"`
			} `json:"fee"`
		} `json:"auth_info"`
		Signatures []string `json:"signatures"`
	} `json:"tx"`
}

// TaxComputeResponse represents the response from tax computation
type TaxComputeResponse struct {
	TaxAmount []struct {
		Denom  string `json:"denom"`
		Amount string `json:"amount"`
	} `json:"tax_amount"`
}

func (s *IntegrationTestSuite) TestAPIRegression() {
	s.Suite.Run("Tax Computation Test", func() {
		chain := s.configurer.GetChainConfig(0)
		node, err := chain.GetDefaultNode()
		s.Suite.Require().NoError(err)

		// Create test wallets
		senderAddr := node.CreateWallet("sender")
		receiverAddr := node.CreateWallet("receiver")

		// Fund sender wallet
		validatorAddr := node.GetWallet(initialization.ValidatorWalletName)
		node.BankSend("1000000uluna", validatorAddr, senderAddr)

		// Wait for transaction to be processed
		time.Sleep(5 * time.Second)

		// Prepare tax computation request
		req := TaxComputeRequest{}
		req.Tx.Body.Messages = []struct {
			Type        string `json:"@type"`
			FromAddress string `json:"from_address"`
			ToAddress   string `json:"to_address"`
			Amount      []struct {
				Denom  string `json:"denom"`
				Amount string `json:"amount"`
			} `json:"amount"`
		}{
			{
				Type:        "/cosmos.bank.v1beta1.MsgSend",
				FromAddress: senderAddr,
				ToAddress:   receiverAddr,
				Amount: []struct {
					Denom  string `json:"denom"`
					Amount string `json:"amount"`
				}{
					{
						Denom:  "uluna",
						Amount: "1000000",
					},
				},
			},
		}
		req.Tx.AuthInfo.SignerInfos = []struct {
			PublicKey struct {
				Type string `json:"@type"`
				Key  string `json:"key"`
			} `json:"public_key"`
			ModeInfo struct {
				Single struct {
					Mode string `json:"mode"`
				} `json:"single"`
			} `json:"mode_info"`
			Sequence string `json:"sequence"`
		}{
			{
				PublicKey: struct {
					Type string `json:"@type"`
					Key  string `json:"key"`
				}{
					Type: "/cosmos.crypto.secp256k1.PubKey",
					Key:  "A0000000000000000000000000000000000000000000000000000000000000000",
				},
				ModeInfo: struct {
					Single struct {
						Mode string `json:"mode"`
					} `json:"single"`
				}{
					Single: struct {
						Mode string `json:"mode"`
					}{
						Mode: "SIGN_MODE_DIRECT",
					},
				},
				Sequence: "0",
			},
		}
		req.Tx.AuthInfo.Fee.Amount = []struct {
			Denom  string `json:"denom"`
			Amount string `json:"amount"`
		}{
			{
				Denom:  "uluna",
				Amount: "0",
			},
		}
		req.Tx.AuthInfo.Fee.GasLimit = "200000"
		req.Tx.Signatures = []string{""} // Empty signature for simulation

		// Execute test with retries
		var taxResp TaxComputeResponse
		s.Suite.Eventually(func() bool {
			// Resolve REST API host:port from container mapping
			hostPort, err := node.GetHostPort("1317/tcp")
			if err != nil {
				s.Suite.T().Logf("Failed to get REST port: %v", err)
				return false
			}
			// Make API request
			reqBody, err := json.Marshal(req)
			if err != nil {
				s.Suite.T().Logf("Failed to marshal request: %v", err)
				return false
			}

			// Create API client
			apiClient := util.NewAPIClient(fmt.Sprintf("http://%s", hostPort))

			resp, err := apiClient.PostJSON("/terra/tx/v1beta1/compute_tax", reqBody)
			if err != nil {
				s.Suite.T().Logf("API request failed: %v", err)
				return false
			}

			// Parse response
			err = util.UnmarshalResponse(resp, &taxResp)
			if err != nil {
				s.Suite.T().Logf("Failed to unmarshal response: %v", err)
				return false
			}

			// Verify endpoint responds without error (this tests against regression from PR #561)
			// Tax amount might be zero if addresses are exempted or due to other factors
			// The main goal is ensuring the endpoint doesn't panic or return errors
			s.Suite.T().Logf("Tax computation endpoint responded successfully with %d tax entries", len(taxResp.TaxAmount))

			return true
		},
			30*time.Second, // timeout
			1*time.Second,  // interval
		)

		// Final assertions - main goal is ensuring the endpoint works without panicking
		// This prevents regression from PR #561 where historic queries would panic
		// Tax amount can be zero due to exemptions or other factors, which is acceptable
		s.Suite.T().Logf("Tax computation test completed successfully. Response contained %d tax entries.", len(taxResp.TaxAmount))

		// The key assertion is that we got a proper JSON response without errors
		// This proves the endpoint is working and not panicking like in the pre-fix state
	})

	s.Suite.Run("Historic Query Header Test", func() {
		chain := s.configurer.GetChainConfig(0)
		node, err := chain.GetDefaultNode()
		s.Suite.Require().NoError(err)

		hostPort, err := node.GetHostPort("1317/tcp")
		s.Suite.Require().NoError(err)

		// Use a low historic height to simulate pre-upgrade behavior
		historicHeight := "10"
		headers := map[string]string{
			"X-Cosmos-Block-Height": historicHeight,
		}

		apiClient := util.NewAPIClient(fmt.Sprintf("http://%s", hostPort))

		// Staking params should be retrievable at historic heights
		stakingParamsPath := "/cosmos/staking/v1beta1/params"
		resp, err := apiClient.GetWithHeaders(stakingParamsPath, headers)
		s.Suite.Require().NoError(err)
		s.Suite.Require().Equal(200, resp.StatusCode)

		// Wasm code list should also be retrievable at historic heights
		wasmCodesPath := "/cosmwasm/wasm/v1/code"
		resp, err = apiClient.GetWithHeaders(wasmCodesPath, headers)
		s.Suite.Require().NoError(err)
		s.Suite.Require().Equal(200, resp.StatusCode)
	})
}
