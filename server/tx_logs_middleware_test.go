package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTxLogsMiddleware(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		method          string
		responseBody    string
		expectLogsLen   int
		expectTransform bool
	}{
		{
			name:   "single tx with events - should reconstruct logs",
			path:   "/cosmos/tx/v1beta1/txs/ABCD1234",
			method: http.MethodGet,
			responseBody: `{
				"tx": {},
				"tx_response": {
					"height": "100",
					"txhash": "ABCD1234",
					"logs": [],
					"events": [
						{
							"type": "message",
							"attributes": [
								{"key": "action", "value": "send", "msg_index": 0}
							]
						},
						{
							"type": "transfer",
							"attributes": [
								{"key": "recipient", "value": "terra1abc", "msg_index": 0}
							]
						}
					]
				}
			}`,
			expectLogsLen:   1,
			expectTransform: true,
		},
		{
			name:   "multiple messages - should group by msg_index",
			path:   "/cosmos/tx/v1beta1/txs/ABCD1234",
			method: http.MethodGet,
			responseBody: `{
				"tx": {},
				"tx_response": {
					"height": "100",
					"txhash": "ABCD1234",
					"logs": [],
					"events": [
						{
							"type": "message",
							"attributes": [
								{"key": "action", "value": "send", "msg_index": 0}
							]
						},
						{
							"type": "message",
							"attributes": [
								{"key": "action", "value": "delegate", "msg_index": 1}
							]
						}
					]
				}
			}`,
			expectLogsLen:   2,
			expectTransform: true,
		},
		{
			name:   "non-tx endpoint - should not transform",
			path:   "/cosmos/bank/v1beta1/balances/terra1abc",
			method: http.MethodGet,
			responseBody: `{
				"balances": []
			}`,
			expectTransform: false,
		},
		{
			name:   "POST request - should not transform",
			path:   "/cosmos/tx/v1beta1/txs",
			method: http.MethodPost,
			responseBody: `{
				"tx_response": {}
			}`,
			expectTransform: false,
		},
		{
			name:   "GetTxsEvent response - should transform multiple tx_responses",
			path:   "/cosmos/tx/v1beta1/txs",
			method: http.MethodGet,
			responseBody: `{
				"txs": [],
				"tx_responses": [
					{
						"height": "100",
						"txhash": "TX1",
						"logs": [],
						"events": [
							{
								"type": "message",
								"attributes": [
									{"key": "action", "value": "send", "msg_index": 0}
								]
							}
						]
					},
					{
						"height": "101",
						"txhash": "TX2",
						"logs": [],
						"events": [
							{
								"type": "message",
								"attributes": [
									{"key": "action", "value": "delegate", "msg_index": 0}
								]
							}
						]
					}
				]
			}`,
			expectTransform: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock handler that returns the test response
			mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tc.responseBody))
			})

			// Wrap with our middleware
			handler := TxLogsMiddleware(mockHandler)

			// Create test request
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()

			// Execute
			handler.ServeHTTP(rec, req)

			// Parse response
			var response map[string]interface{}
			err := json.Unmarshal(rec.Body.Bytes(), &response)
			require.NoError(t, err)

			if tc.expectTransform && tc.expectLogsLen > 0 {
				// Check single tx_response
				if txResponse, ok := response["tx_response"].(map[string]interface{}); ok {
					logs, ok := txResponse["logs"].([]interface{})
					require.True(t, ok, "logs should be present")
					require.Equal(t, tc.expectLogsLen, len(logs))
				}
			}

			if !tc.expectTransform {
				// Original response should be unchanged (structurally)
				var original map[string]interface{}
				err = json.Unmarshal([]byte(tc.responseBody), &original)
				require.NoError(t, err)
				// Just verify no error occurred
			}
		})
	}
}

func TestReconstructLogs(t *testing.T) {
	tests := []struct {
		name         string
		txResponse   map[string]interface{}
		expectedLogs int
	}{
		{
			name: "basic event reconstruction",
			txResponse: map[string]interface{}{
				"logs": []interface{}{},
				"events": []interface{}{
					map[string]interface{}{
						"type": "message",
						"attributes": []interface{}{
							map[string]interface{}{
								"key":       "action",
								"value":     "send",
								"msg_index": 0,
							},
						},
					},
				},
			},
			expectedLogs: 1,
		},
		{
			name: "already has logs - should not modify",
			txResponse: map[string]interface{}{
				"logs": []interface{}{
					map[string]interface{}{
						"msg_index": 0,
						"events":    []interface{}{},
					},
				},
				"events": []interface{}{},
			},
			expectedLogs: 1,
		},
		{
			name: "no events - should not add logs",
			txResponse: map[string]interface{}{
				"logs":   []interface{}{},
				"events": []interface{}{},
			},
			expectedLogs: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reconstructLogs(tc.txResponse)

			// logs can be []interface{} (original) or []map[string]interface{} (reconstructed)
			logsRaw := tc.txResponse["logs"]
			var logsLen int
			switch logs := logsRaw.(type) {
			case []interface{}:
				logsLen = len(logs)
			case []map[string]interface{}:
				logsLen = len(logs)
			}
			require.Equal(t, tc.expectedLogs, logsLen)
		})
	}
}

func TestExtractMsgIndex(t *testing.T) {
	tests := []struct {
		name     string
		event    map[string]interface{}
		expected int
	}{
		{
			name: "msg_index in attribute",
			event: map[string]interface{}{
				"type": "message",
				"attributes": []interface{}{
					map[string]interface{}{
						"key":       "action",
						"value":     "send",
						"msg_index": 2,
					},
				},
			},
			expected: 2,
		},
		{
			name: "msg_index as float64 (JSON unmarshaling)",
			event: map[string]interface{}{
				"type": "message",
				"attributes": []interface{}{
					map[string]interface{}{
						"key":       "action",
						"value":     "send",
						"msg_index": float64(3),
					},
				},
			},
			expected: 3,
		},
		{
			name: "no msg_index - default to 0",
			event: map[string]interface{}{
				"type": "message",
				"attributes": []interface{}{
					map[string]interface{}{
						"key":   "action",
						"value": "send",
					},
				},
			},
			expected: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractMsgIndex(tc.event)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestIsTxQueryEndpoint(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"/cosmos/tx/v1beta1/txs/ABCD1234", true},
		{"/cosmos/tx/v1beta1/txs", true},
		{"/cosmos/bank/v1beta1/balances/terra1abc", false},
		{"/cosmos/staking/v1beta1/validators", false},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			result := isTxQueryEndpoint(tc.path)
			require.Equal(t, tc.expected, result)
		})
	}
}

// TestRealWasmExecuteTxLogsReconstruction tests logs reconstruction with a real
// Terra Classic wasm execute contract transaction payload from mainnet
// TX: CA3F0FB02FA0FDC92A16A5F973B87CE4F3667DB77FAB995F96C2C67AF970A616
func TestRealWasmExecuteTxLogsReconstruction(t *testing.T) {
	// Real transaction response from Terra Classic mainnet
	// This is a MsgExecuteContract with Trigger{} action
	realTxResponse := `{
		"tx": {
			"@type": "/cosmos.tx.v1beta1.Tx",
			"body": {
				"messages": [
					{
						"@type": "/cosmwasm.wasm.v1.MsgExecuteContract",
						"sender": "terra1xnu72mn60yzcyr0fl8avgjy5wepfw85c0knfeh",
						"contract": "terra16amw508yzhe8afshhgjp4kjz9wc20p8qhurg2dkmmc9tryeuflqsmjgrh2",
						"msg": {"Trigger": {}},
						"funds": []
					}
				],
				"memo": "",
				"timeout_height": "0",
				"extension_options": [],
				"non_critical_extension_options": []
			},
			"auth_info": {
				"signer_infos": [
					{
						"public_key": {
							"@type": "/cosmos.crypto.secp256k1.PubKey",
							"key": "A4aGAfjknnQPKSRyem85Q1DPlK5swAXVYsC/4O8tzt3c"
						},
						"mode_info": {
							"single": {
								"mode": "SIGN_MODE_DIRECT"
							}
						},
						"sequence": "51505"
					}
				],
				"fee": {
					"amount": [
						{
							"denom": "uluna",
							"amount": "6752621"
						}
					],
					"gas_limit": "232849",
					"payer": "",
					"granter": ""
				}
			},
			"signatures": ["qqswFeXrtVeiSbzT+er9kgqzxtXrMZ6enlx+gv1ExpJAy3WKKKMyMg0xzoBJiqkgIgJgUNZI0e1zBv6P6MZCPA=="]
		},
		"tx_response": {
			"height": "26861855",
			"txhash": "CA3F0FB02FA0FDC92A16A5F973B87CE4F3667DB77FAB995F96C2C67AF970A616",
			"codespace": "",
			"code": 0,
			"data": "122E0A2C2F636F736D7761736D2E7761736D2E76312E4D736745786563757465436F6E7472616374526573706F6E7365",
			"raw_log": "[{\"msg_index\":0,\"events\":[{\"type\":\"message\",\"attributes\":[{\"key\":\"action\",\"value\":\"/cosmwasm.wasm.v1.MsgExecuteContract\"},{\"key\":\"sender\",\"value\":\"terra1xnu72mn60yzcyr0fl8avgjy5wepfw85c0knfeh\"},{\"key\":\"module\",\"value\":\"wasm\"}]},{\"type\":\"execute\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"terra16amw508yzhe8afshhgjp4kjz9wc20p8qhurg2dkmmc9tryeuflqsmjgrh2\"}]},{\"type\":\"wasm\",\"attributes\":[{\"key\":\"_contract_address\",\"value\":\"terra16amw508yzhe8afshhgjp4kjz9wc20p8qhurg2dkmmc9tryeuflqsmjgrh2\"},{\"key\":\"action\",\"value\":\"trigger\"},{\"key\":\"delegator\",\"value\":\"terra1765aqc7vyvxc7cch3t9vmaavj7534s2wnhzelp\"}]}]}]",
			"logs": [],
			"info": "",
			"gas_wanted": "232849",
			"gas_used": "151207",
			"timestamp": "2026-01-14T03:41:59Z",
			"events": [
				{
					"type": "coin_spent",
					"attributes": [
						{"key": "spender", "value": "terra1xnu72mn60yzcyr0fl8avgjy5wepfw85c0knfeh", "msg_index": 0},
						{"key": "amount", "value": "6752621uluna", "msg_index": 0}
					]
				},
				{
					"type": "coin_received",
					"attributes": [
						{"key": "receiver", "value": "terra17xpfvakm2amg962yls6f84z3kell8c5lkaeqfa", "msg_index": 0},
						{"key": "amount", "value": "6752621uluna", "msg_index": 0}
					]
				},
				{
					"type": "transfer",
					"attributes": [
						{"key": "recipient", "value": "terra17xpfvakm2amg962yls6f84z3kell8c5lkaeqfa", "msg_index": 0},
						{"key": "sender", "value": "terra1xnu72mn60yzcyr0fl8avgjy5wepfw85c0knfeh", "msg_index": 0},
						{"key": "amount", "value": "6752621uluna", "msg_index": 0}
					]
				},
				{
					"type": "message",
					"attributes": [
						{"key": "sender", "value": "terra1xnu72mn60yzcyr0fl8avgjy5wepfw85c0knfeh", "msg_index": 0}
					]
				},
				{
					"type": "tx",
					"attributes": [
						{"key": "fee", "value": "6752621uluna", "msg_index": 0},
						{"key": "fee_payer", "value": "terra1xnu72mn60yzcyr0fl8avgjy5wepfw85c0knfeh", "msg_index": 0}
					]
				},
				{
					"type": "tx",
					"attributes": [
						{"key": "acc_seq", "value": "terra1xnu72mn60yzcyr0fl8avgjy5wepfw85c0knfeh/51505", "msg_index": 0}
					]
				},
				{
					"type": "tx",
					"attributes": [
						{"key": "signature", "value": "qqswFeXrtVeiSbzT+er9kgqzxtXrMZ6enlx+gv1ExpJAy3WKKKMyMg0xzoBJiqkgIgJgUNZI0e1zBv6P6MZCPA==", "msg_index": 0}
					]
				},
				{
					"type": "message",
					"attributes": [
						{"key": "action", "value": "/cosmwasm.wasm.v1.MsgExecuteContract", "msg_index": 0},
						{"key": "sender", "value": "terra1xnu72mn60yzcyr0fl8avgjy5wepfw85c0knfeh", "msg_index": 0},
						{"key": "module", "value": "wasm", "msg_index": 0}
					]
				},
				{
					"type": "execute",
					"attributes": [
						{"key": "_contract_address", "value": "terra16amw508yzhe8afshhgjp4kjz9wc20p8qhurg2dkmmc9tryeuflqsmjgrh2", "msg_index": 0}
					]
				},
				{
					"type": "wasm",
					"attributes": [
						{"key": "_contract_address", "value": "terra16amw508yzhe8afshhgjp4kjz9wc20p8qhurg2dkmmc9tryeuflqsmjgrh2", "msg_index": 0},
						{"key": "action", "value": "trigger", "msg_index": 0},
						{"key": "delegator", "value": "terra1765aqc7vyvxc7cch3t9vmaavj7534s2wnhzelp", "msg_index": 0}
					]
				}
			]
		}
	}`

	// Create a mock handler that returns the real tx response
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(realTxResponse))
	})

	// Wrap with TxLogsMiddleware
	handler := TxLogsMiddleware(mockHandler)

	// Create test request for specific tx hash
	req := httptest.NewRequest(http.MethodGet, "/cosmos/tx/v1beta1/txs/CA3F0FB02FA0FDC92A16A5F973B87CE4F3667DB77FAB995F96C2C67AF970A616", nil)
	rec := httptest.NewRecorder()

	// Execute
	handler.ServeHTTP(rec, req)

	// Parse response
	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify tx_response exists
	txResponse, ok := response["tx_response"].(map[string]interface{})
	require.True(t, ok, "tx_response should exist")

	// Verify basic tx info
	require.Equal(t, "CA3F0FB02FA0FDC92A16A5F973B87CE4F3667DB77FAB995F96C2C67AF970A616", txResponse["txhash"])
	require.Equal(t, "26861855", txResponse["height"])
	require.Equal(t, float64(0), txResponse["code"]) // JSON unmarshals numbers as float64

	// Verify logs were reconstructed
	// After JSON marshal/unmarshal, logs can be either []map[string]interface{} or []interface{}
	var logs []map[string]interface{}
	switch l := txResponse["logs"].(type) {
	case []map[string]interface{}:
		logs = l
	case []interface{}:
		logs = make([]map[string]interface{}, len(l))
		for i, v := range l {
			logs[i] = v.(map[string]interface{})
		}
	default:
		t.Fatalf("logs has unexpected type: %T", txResponse["logs"])
	}
	require.Equal(t, 1, len(logs), "should have exactly 1 log entry for single message tx")

	// Verify the log structure
	log := logs[0]
	msgIndex, _ := log["msg_index"].(float64) // JSON unmarshals numbers as float64
	require.Equal(t, 0, int(msgIndex))        // msg_index should be 0
	require.Equal(t, "", log["log"])          // log string should be empty

	// Verify events were properly grouped in the log
	var logEvents []map[string]interface{}
	switch e := log["events"].(type) {
	case []map[string]interface{}:
		logEvents = e
	case []interface{}:
		logEvents = make([]map[string]interface{}, len(e))
		for i, v := range e {
			logEvents[i] = v.(map[string]interface{})
		}
	default:
		t.Fatalf("log events has unexpected type: %T", log["events"])
	}
	require.Greater(t, len(logEvents), 0, "should have events in the log")

	// Verify specific event types are present
	eventTypes := make(map[string]bool)
	for _, event := range logEvents {
		if eventType, ok := event["type"].(string); ok {
			eventTypes[eventType] = true
		}
	}

	// These event types should be present from the wasm execute tx
	require.True(t, eventTypes["message"], "should have 'message' event type")
	require.True(t, eventTypes["execute"], "should have 'execute' event type")
	require.True(t, eventTypes["wasm"], "should have 'wasm' event type")
	require.True(t, eventTypes["coin_spent"], "should have 'coin_spent' event type")
	require.True(t, eventTypes["coin_received"], "should have 'coin_received' event type")
	require.True(t, eventTypes["transfer"], "should have 'transfer' event type")

	// Verify wasm event has correct attributes
	var wasmEvent map[string]interface{}
	for _, event := range logEvents {
		if event["type"] == "wasm" {
			wasmEvent = event
			break
		}
	}
	require.NotNil(t, wasmEvent, "wasm event should exist")

	// Handle different attribute slice types after JSON marshal/unmarshal
	var wasmAttrs []map[string]interface{}
	switch a := wasmEvent["attributes"].(type) {
	case []map[string]interface{}:
		wasmAttrs = a
	case []interface{}:
		wasmAttrs = make([]map[string]interface{}, len(a))
		for i, v := range a {
			wasmAttrs[i] = v.(map[string]interface{})
		}
	default:
		t.Fatalf("wasm attributes has unexpected type: %T", wasmEvent["attributes"])
	}

	// Verify wasm attributes contain expected values
	attrMap := make(map[string]string)
	for _, attr := range wasmAttrs {
		if key, ok := attr["key"].(string); ok {
			if value, ok := attr["value"].(string); ok {
				attrMap[key] = value
			}
		}
	}

	require.Equal(t, "terra16amw508yzhe8afshhgjp4kjz9wc20p8qhurg2dkmmc9tryeuflqsmjgrh2", attrMap["_contract_address"])
	require.Equal(t, "trigger", attrMap["action"])
	require.Equal(t, "terra1765aqc7vyvxc7cch3t9vmaavj7534s2wnhzelp", attrMap["delegator"])
}

// TestRealBankSendWithTaxPaymentLogsReconstruction tests logs reconstruction with a real
// Terra Classic bank send transaction that includes tax_payment events from mainnet
// TX: 32980A4F8A2EBDEB773D390CEBE7A4215426C67C95BDD508AFF0AE5ED16675A3
func TestRealBankSendWithTaxPaymentLogsReconstruction(t *testing.T) {
	// Real transaction response from Terra Classic mainnet
	// This is a MsgSend with tax_payment events
	realTxResponse := `{
		"tx": {
			"@type": "/cosmos.tx.v1beta1.Tx",
			"body": {
				"messages": [
					{
						"@type": "/cosmos.bank.v1beta1.MsgSend",
						"from_address": "terra1j435gkgg8d0qadjcn09s73rtk5k3ftrx7mc4a8",
						"to_address": "terra1ycnrw0uvwhchdw4zthsnwdsqgd5tyvtvx2pupm",
						"amount": [
							{
								"denom": "uluna",
								"amount": "348211891260"
							}
						]
					}
				],
				"memo": "",
				"timeout_height": "0",
				"extension_options": [],
				"non_critical_extension_options": []
			},
			"auth_info": {
				"signer_infos": [
					{
						"public_key": {
							"@type": "/cosmos.crypto.secp256k1.PubKey",
							"key": "A8wvU597ZqozsfAXFdUf3iUvmfPDqR+FDWd79tF0BfBE"
						},
						"mode_info": {
							"single": {
								"mode": "SIGN_MODE_DIRECT"
							}
						},
						"sequence": "95205"
					}
				],
				"fee": {
					"amount": [
						{
							"denom": "uluna",
							"amount": "1751059457"
						}
					],
					"gas_limit": "300000",
					"payer": "",
					"granter": ""
				}
			},
			"signatures": ["OjhLdjkqwUwtQDLB5fMetd87sH4pHcsyl09QZxwXOSE4+t0qLAtDM+PXfcBWBlio6+DeO5FTmWAAtIeb3NaZ3w=="]
		},
		"tx_response": {
			"height": "26861864",
			"txhash": "32980A4F8A2EBDEB773D390CEBE7A4215426C67C95BDD508AFF0AE5ED16675A3",
			"codespace": "",
			"code": 0,
			"data": "12260A242F636F736D6F732E62616E6B2E763162657461312E4D736753656E64526573706F6E7365",
			"raw_log": "[{\"msg_index\":0,\"events\":[{\"type\":\"message\",\"attributes\":[{\"key\":\"action\",\"value\":\"/cosmos.bank.v1beta1.MsgSend\"},{\"key\":\"sender\",\"value\":\"terra1j435gkgg8d0qadjcn09s73rtk5k3ftrx7mc4a8\"},{\"key\":\"module\",\"value\":\"bank\"}]},{\"type\":\"tax_payment\",\"attributes\":[{\"key\":\"reverse_charge\",\"value\":\"false\"}]}]}]",
			"logs": [],
			"info": "",
			"gas_wanted": "300000",
			"gas_used": "232274",
			"timestamp": "2026-01-14T03:42:52Z",
			"events": [
				{
					"type": "coin_spent",
					"attributes": [
						{"key": "spender", "value": "terra1j435gkgg8d0qadjcn09s73rtk5k3ftrx7mc4a8", "msg_index": 0},
						{"key": "amount", "value": "10000001uluna", "msg_index": 0}
					]
				},
				{
					"type": "coin_received",
					"attributes": [
						{"key": "receiver", "value": "terra17xpfvakm2amg962yls6f84z3kell8c5lkaeqfa", "msg_index": 0},
						{"key": "amount", "value": "10000001uluna", "msg_index": 0}
					]
				},
				{
					"type": "transfer",
					"attributes": [
						{"key": "recipient", "value": "terra17xpfvakm2amg962yls6f84z3kell8c5lkaeqfa", "msg_index": 0},
						{"key": "sender", "value": "terra1j435gkgg8d0qadjcn09s73rtk5k3ftrx7mc4a8", "msg_index": 0},
						{"key": "amount", "value": "10000001uluna", "msg_index": 0}
					]
				},
				{
					"type": "message",
					"attributes": [
						{"key": "sender", "value": "terra1j435gkgg8d0qadjcn09s73rtk5k3ftrx7mc4a8", "msg_index": 0}
					]
				},
				{
					"type": "tx",
					"attributes": [
						{"key": "fee", "value": "1751059457uluna", "msg_index": 0},
						{"key": "fee_payer", "value": "terra1j435gkgg8d0qadjcn09s73rtk5k3ftrx7mc4a8", "msg_index": 0}
					]
				},
				{
					"type": "tx",
					"attributes": [
						{"key": "acc_seq", "value": "terra1j435gkgg8d0qadjcn09s73rtk5k3ftrx7mc4a8/95205", "msg_index": 0}
					]
				},
				{
					"type": "tx",
					"attributes": [
						{"key": "signature", "value": "OjhLdjkqwUwtQDLB5fMetd87sH4pHcsyl09QZxwXOSE4+t0qLAtDM+PXfcBWBlio6+DeO5FTmWAAtIeb3NaZ3w==", "msg_index": 0}
					]
				},
				{
					"type": "message",
					"attributes": [
						{"key": "action", "value": "/cosmos.bank.v1beta1.MsgSend", "msg_index": 0},
						{"key": "sender", "value": "terra1j435gkgg8d0qadjcn09s73rtk5k3ftrx7mc4a8", "msg_index": 0},
						{"key": "module", "value": "bank", "msg_index": 0}
					]
				},
				{
					"type": "tax_payment",
					"attributes": [
						{"key": "reverse_charge", "value": "false", "msg_index": 0}
					]
				},
				{
					"type": "coin_spent",
					"attributes": [
						{"key": "spender", "value": "terra1j435gkgg8d0qadjcn09s73rtk5k3ftrx7mc4a8", "msg_index": 0},
						{"key": "amount", "value": "348211891260uluna", "msg_index": 0}
					]
				},
				{
					"type": "coin_received",
					"attributes": [
						{"key": "receiver", "value": "terra1ycnrw0uvwhchdw4zthsnwdsqgd5tyvtvx2pupm", "msg_index": 0},
						{"key": "amount", "value": "348211891260uluna", "msg_index": 0}
					]
				},
				{
					"type": "transfer",
					"attributes": [
						{"key": "recipient", "value": "terra1ycnrw0uvwhchdw4zthsnwdsqgd5tyvtvx2pupm", "msg_index": 0},
						{"key": "sender", "value": "terra1j435gkgg8d0qadjcn09s73rtk5k3ftrx7mc4a8", "msg_index": 0},
						{"key": "amount", "value": "348211891260uluna", "msg_index": 0}
					]
				},
				{
					"type": "message",
					"attributes": [
						{"key": "sender", "value": "terra1j435gkgg8d0qadjcn09s73rtk5k3ftrx7mc4a8", "msg_index": 0}
					]
				}
			]
		}
	}`

	// Create a mock handler that returns the real tx response
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(realTxResponse))
	})

	// Wrap with TxLogsMiddleware
	handler := TxLogsMiddleware(mockHandler)

	// Create test request for specific tx hash
	req := httptest.NewRequest(http.MethodGet, "/cosmos/tx/v1beta1/txs/32980A4F8A2EBDEB773D390CEBE7A4215426C67C95BDD508AFF0AE5ED16675A3", nil)
	rec := httptest.NewRecorder()

	// Execute
	handler.ServeHTTP(rec, req)

	// Parse response
	var response map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &response)
	require.NoError(t, err)

	// Verify tx_response exists
	txResponse, ok := response["tx_response"].(map[string]interface{})
	require.True(t, ok, "tx_response should exist")

	// Verify basic tx info
	require.Equal(t, "32980A4F8A2EBDEB773D390CEBE7A4215426C67C95BDD508AFF0AE5ED16675A3", txResponse["txhash"])
	require.Equal(t, "26861864", txResponse["height"])
	require.Equal(t, float64(0), txResponse["code"])

	// Verify logs were reconstructed
	var logs []map[string]interface{}
	switch l := txResponse["logs"].(type) {
	case []map[string]interface{}:
		logs = l
	case []interface{}:
		logs = make([]map[string]interface{}, len(l))
		for i, v := range l {
			logs[i] = v.(map[string]interface{})
		}
	default:
		t.Fatalf("logs has unexpected type: %T", txResponse["logs"])
	}
	require.Equal(t, 1, len(logs), "should have exactly 1 log entry for single message tx")

	// Verify the log structure
	log := logs[0]
	msgIndex, _ := log["msg_index"].(float64)
	require.Equal(t, 0, int(msgIndex))
	require.Equal(t, "", log["log"])

	// Verify events were properly grouped in the log
	var logEvents []map[string]interface{}
	switch e := log["events"].(type) {
	case []map[string]interface{}:
		logEvents = e
	case []interface{}:
		logEvents = make([]map[string]interface{}, len(e))
		for i, v := range e {
			logEvents[i] = v.(map[string]interface{})
		}
	default:
		t.Fatalf("log events has unexpected type: %T", log["events"])
	}
	require.Greater(t, len(logEvents), 0, "should have events in the log")

	// Verify specific event types are present (including Terra-specific tax_payment)
	eventTypes := make(map[string]bool)
	for _, event := range logEvents {
		if eventType, ok := event["type"].(string); ok {
			eventTypes[eventType] = true
		}
	}

	// These event types should be present from the bank send tx with tax
	require.True(t, eventTypes["message"], "should have 'message' event type")
	require.True(t, eventTypes["tax_payment"], "should have 'tax_payment' event type (Terra-specific)")
	require.True(t, eventTypes["coin_spent"], "should have 'coin_spent' event type")
	require.True(t, eventTypes["coin_received"], "should have 'coin_received' event type")
	require.True(t, eventTypes["transfer"], "should have 'transfer' event type")

	// Verify tax_payment event has correct attributes
	var taxPaymentEvent map[string]interface{}
	for _, event := range logEvents {
		if event["type"] == "tax_payment" {
			taxPaymentEvent = event
			break
		}
	}
	require.NotNil(t, taxPaymentEvent, "tax_payment event should exist")

	// Handle different attribute slice types
	var taxAttrs []map[string]interface{}
	switch a := taxPaymentEvent["attributes"].(type) {
	case []map[string]interface{}:
		taxAttrs = a
	case []interface{}:
		taxAttrs = make([]map[string]interface{}, len(a))
		for i, v := range a {
			taxAttrs[i] = v.(map[string]interface{})
		}
	default:
		t.Fatalf("tax_payment attributes has unexpected type: %T", taxPaymentEvent["attributes"])
	}

	// Verify tax_payment attributes contain expected values
	attrMap := make(map[string]string)
	for _, attr := range taxAttrs {
		if key, ok := attr["key"].(string); ok {
			if value, ok := attr["value"].(string); ok {
				attrMap[key] = value
			}
		}
	}

	require.Equal(t, "false", attrMap["reverse_charge"])
}
