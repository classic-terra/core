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

			if tc.expectTransform {
				// Check single tx_response
				if txResponse, ok := response["tx_response"].(map[string]interface{}); ok {
					logs, ok := txResponse["logs"].([]interface{})
					require.True(t, ok, "logs should be present")
					if tc.expectLogsLen > 0 {
						require.Equal(t, tc.expectLogsLen, len(logs))
					}
				}

				// Check multiple tx_responses (GetTxsEvent)
				if txResponses, ok := response["tx_responses"].([]interface{}); ok {
					require.Equal(t, 2, len(txResponses), "should have 2 tx_responses")

					// Verify first transaction (TX1)
					tx1, ok := txResponses[0].(map[string]interface{})
					require.True(t, ok, "first tx_response should be a map")
					require.Equal(t, "TX1", tx1["txhash"], "first tx hash should match")
					logs1, ok := tx1["logs"].([]interface{})
					require.True(t, ok, "first tx should have logs")
					require.Equal(t, 1, len(logs1), "first tx should have 1 log entry")
					// Verify log content
					log1, ok := logs1[0].(map[string]interface{})
					require.True(t, ok, "log should be a map")
					events1, ok := log1["events"].([]interface{})
					require.True(t, ok, "log should have events")
					require.Equal(t, 1, len(events1), "log should have 1 event")
					event1, ok := events1[0].(map[string]interface{})
					require.True(t, ok, "event should be a map")
					require.Equal(t, "message", event1["type"], "event type should be message")
					attrs1, ok := event1["attributes"].([]interface{})
					require.True(t, ok, "event should have attributes")
					require.Equal(t, 1, len(attrs1), "event should have 1 attribute")
					attr1, ok := attrs1[0].(map[string]interface{})
					require.True(t, ok, "attribute should be a map")
					require.Equal(t, "action", attr1["key"], "attribute key should be action")
					require.Equal(t, "send", attr1["value"], "attribute value should be send")

					// Verify second transaction (TX2)
					tx2, ok := txResponses[1].(map[string]interface{})
					require.True(t, ok, "second tx_response should be a map")
					require.Equal(t, "TX2", tx2["txhash"], "second tx hash should match")
					logs2, ok := tx2["logs"].([]interface{})
					require.True(t, ok, "second tx should have logs")
					require.Equal(t, 1, len(logs2), "second tx should have 1 log entry")
					// Verify log content
					log2, ok := logs2[0].(map[string]interface{})
					require.True(t, ok, "log should be a map")
					events2, ok := log2["events"].([]interface{})
					require.True(t, ok, "log should have events")
					require.Equal(t, 1, len(events2), "log should have 1 event")
					event2, ok := events2[0].(map[string]interface{})
					require.True(t, ok, "event should be a map")
					require.Equal(t, "message", event2["type"], "event type should be message")
					attrs2, ok := event2["attributes"].([]interface{})
					require.True(t, ok, "event should have attributes")
					require.Equal(t, 1, len(attrs2), "event should have 1 attribute")
					attr2, ok := attrs2[0].(map[string]interface{})
					require.True(t, ok, "attribute should be a map")
					require.Equal(t, "action", attr2["key"], "attribute key should be action")
					require.Equal(t, "delegate", attr2["value"], "attribute value should be delegate")
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

// TestRealOracleMultiMessageTxLogsReconstruction tests logs reconstruction with a real
// Terra Classic oracle multi-message transaction from mainnet
// TX: A7075AA154D7D4DCBAB70465EAF85168CED8C76E700CC30142DBA46F1D80E0F2
// This transaction contains TWO messages:
// 1. MsgAggregateExchangeRateVote (msg_index: 0)
// 2. MsgAggregateExchangeRatePrevote (msg_index: 1)
func TestRealOracleMultiMessageTxLogsReconstruction(t *testing.T) {
	// Real transaction response from Terra Classic mainnet
	// This is a multi-message tx with oracle vote and prevote messages
	realTxResponse := `{
		"tx": {
			"@type": "/cosmos.tx.v1beta1.Tx",
			"body": {
				"messages": [
					{
						"@type": "/terra.oracle.v1beta1.MsgAggregateExchangeRateVote",
						"feeder": "terra19p69dm52exmhtyklgcpd2jrfwtv0awlp0ve63z",
						"validator": "terravaloper1j5pj3n3m9nxmv9dgl4wnv2yq53k2jf2283j5zm",
						"exchange_rates": "0.000066097941407884uaud,0.000061456692115652ucad,0.00003538288127528uchf,0.00030892650733513ucny,0.000283966498350392udkk,0.00003800111245669ueur,0.000032909024645851ugbp,0.000345525060444445uhkd,0.746825090789225464uidr,0.003995990890723461uinr,0.007034943470680445ujpy,0.065282105991835183ukrw,0.0umnt,0.00017969339697793umyr,0.000446742366801887unok,0.002633155589658828uphp,0.000032429987267087usdr,0.000407127883304437usek,0.00005702359065234usgd,0.001393924629462205uthb,0.0utwd,0.000044286727883801uusd",
						"salt": "2c52"
					},
					{
						"@type": "/terra.oracle.v1beta1.MsgAggregateExchangeRatePrevote",
						"feeder": "terra19p69dm52exmhtyklgcpd2jrfwtv0awlp0ve63z",
						"validator": "terravaloper1j5pj3n3m9nxmv9dgl4wnv2yq53k2jf2283j5zm",
						"hash": "0a15e0773b0819d71bcd74e2cf427f92597f85a7"
					}
				],
				"memo": "@classic-terra/oracle-feeder@3.1.5",
				"timeout_height": "0",
				"extension_options": [],
				"non_critical_extension_options": []
			},
			"auth_info": {
				"signer_infos": [
					{
						"public_key": {
							"@type": "/cosmos.crypto.secp256k1.PubKey",
							"key": "Ag64r5OyRGIi3qMB/OlximK99iOJQ5WLMkP+ZDz4cd19"
						},
						"mode_info": {
							"single": {
								"mode": "SIGN_MODE_DIRECT"
							}
						},
						"sequence": "165214"
					}
				],
				"fee": {
					"amount": [],
					"gas_limit": "300000",
					"payer": "",
					"granter": ""
				}
			},
			"signatures": ["gBcq2FcjuBAfXz8Ha5w7c6HCdwuh1eDzKqi4ef7fMN1A77wE9vdJWTclbV0u3xUo3LPwjXR0v/hawghBa6PMZQ=="]
		},
		"tx_response": {
			"height": "26864566",
			"txhash": "A7075AA154D7D4DCBAB70465EAF85168CED8C76E700CC30142DBA46F1D80E0F2",
			"codespace": "",
			"code": 0,
			"data": "123C0A3A2F74657272612E6F7261636C652E763162657461312E4D736741676772656761746545786368616E676552617465566F7465526573706F6E7365123F0A3D2F74657272612E6F7261636C652E763162657461312E4D736741676772656761746545786368616E676552617465507265766F7465526573706F6E7365",
			"raw_log": "[{\"msg_index\":0,\"events\":[{\"type\":\"message\",\"attributes\":[{\"key\":\"action\",\"value\":\"/terra.oracle.v1beta1.MsgAggregateExchangeRateVote\"},{\"key\":\"sender\",\"value\":\"terra19p69dm52exmhtyklgcpd2jrfwtv0awlp0ve63z\"}]},{\"type\":\"aggregate_vote\",\"attributes\":[{\"key\":\"voter\",\"value\":\"terravaloper1j5pj3n3m9nxmv9dgl4wnv2yq53k2jf2283j5zm\"},{\"key\":\"exchange_rates\",\"value\":\"0.000066097941407884uaud,0.000061456692115652ucad,0.00003538288127528uchf,0.00030892650733513ucny,0.000283966498350392udkk,0.00003800111245669ueur,0.000032909024645851ugbp,0.000345525060444445uhkd,0.746825090789225464uidr,0.003995990890723461uinr,0.007034943470680445ujpy,0.065282105991835183ukrw,0.0umnt,0.00017969339697793umyr,0.000446742366801887unok,0.002633155589658828uphp,0.000032429987267087usdr,0.000407127883304437usek,0.00005702359065234usgd,0.001393924629462205uthb,0.0utwd,0.000044286727883801uusd\"}]},{\"type\":\"message\",\"attributes\":[{\"key\":\"module\",\"value\":\"oracle\"},{\"key\":\"sender\",\"value\":\"terra19p69dm52exmhtyklgcpd2jrfwtv0awlp0ve63z\"}]}]},{\"msg_index\":1,\"events\":[{\"type\":\"message\",\"attributes\":[{\"key\":\"action\",\"value\":\"/terra.oracle.v1beta1.MsgAggregateExchangeRatePrevote\"},{\"key\":\"sender\",\"value\":\"terra19p69dm52exmhtyklgcpd2jrfwtv0awlp0ve63z\"}]},{\"type\":\"aggregate_prevote\",\"attributes\":[{\"key\":\"voter\",\"value\":\"terravaloper1j5pj3n3m9nxmv9dgl4wnv2yq53k2jf2283j5zm\"}]},{\"type\":\"message\",\"attributes\":[{\"key\":\"module\",\"value\":\"oracle\"},{\"key\":\"sender\",\"value\":\"terra19p69dm52exmhtyklgcpd2jrfwtv0awlp0ve63z\"}]}]}]",
			"logs": [],
			"info": "",
			"gas_wanted": "300000",
			"gas_used": "110287",
			"timestamp": "2026-01-14T08:09:39Z",
			"events": [
				{
					"type": "tx",
					"attributes": [
						{"key": "fee", "value": "", "msg_index": 0},
						{"key": "fee_payer", "value": "terra19p69dm52exmhtyklgcpd2jrfwtv0awlp0ve63z", "msg_index": 0}
					]
				},
				{
					"type": "tx",
					"attributes": [
						{"key": "acc_seq", "value": "terra19p69dm52exmhtyklgcpd2jrfwtv0awlp0ve63z/165214", "msg_index": 0}
					]
				},
				{
					"type": "tx",
					"attributes": [
						{"key": "signature", "value": "gBcq2FcjuBAfXz8Ha5w7c6HCdwuh1eDzKqi4ef7fMN1A77wE9vdJWTclbV0u3xUo3LPwjXR0v/hawghBa6PMZQ==", "msg_index": 0}
					]
				},
				{
					"type": "message",
					"attributes": [
						{"key": "action", "value": "/terra.oracle.v1beta1.MsgAggregateExchangeRateVote", "msg_index": 0},
						{"key": "sender", "value": "terra19p69dm52exmhtyklgcpd2jrfwtv0awlp0ve63z", "msg_index": 0}
					]
				},
				{
					"type": "aggregate_vote",
					"attributes": [
						{"key": "voter", "value": "terravaloper1j5pj3n3m9nxmv9dgl4wnv2yq53k2jf2283j5zm", "msg_index": 0},
						{"key": "exchange_rates", "value": "0.000066097941407884uaud,0.000061456692115652ucad,0.00003538288127528uchf,0.00030892650733513ucny,0.000283966498350392udkk,0.00003800111245669ueur,0.000032909024645851ugbp,0.000345525060444445uhkd,0.746825090789225464uidr,0.003995990890723461uinr,0.007034943470680445ujpy,0.065282105991835183ukrw,0.0umnt,0.00017969339697793umyr,0.000446742366801887unok,0.002633155589658828uphp,0.000032429987267087usdr,0.000407127883304437usek,0.00005702359065234usgd,0.001393924629462205uthb,0.0utwd,0.000044286727883801uusd", "msg_index": 0}
					]
				},
				{
					"type": "message",
					"attributes": [
						{"key": "module", "value": "oracle", "msg_index": 0},
						{"key": "sender", "value": "terra19p69dm52exmhtyklgcpd2jrfwtv0awlp0ve63z", "msg_index": 0}
					]
				},
				{
					"type": "message",
					"attributes": [
						{"key": "action", "value": "/terra.oracle.v1beta1.MsgAggregateExchangeRatePrevote", "msg_index": 1},
						{"key": "sender", "value": "terra19p69dm52exmhtyklgcpd2jrfwtv0awlp0ve63z", "msg_index": 1}
					]
				},
				{
					"type": "aggregate_prevote",
					"attributes": [
						{"key": "voter", "value": "terravaloper1j5pj3n3m9nxmv9dgl4wnv2yq53k2jf2283j5zm", "msg_index": 1}
					]
				},
				{
					"type": "message",
					"attributes": [
						{"key": "module", "value": "oracle", "msg_index": 1},
						{"key": "sender", "value": "terra19p69dm52exmhtyklgcpd2jrfwtv0awlp0ve63z", "msg_index": 1}
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
	req := httptest.NewRequest(http.MethodGet, "/cosmos/tx/v1beta1/txs/A7075AA154D7D4DCBAB70465EAF85168CED8C76E700CC30142DBA46F1D80E0F2", nil)
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
	require.Equal(t, "A7075AA154D7D4DCBAB70465EAF85168CED8C76E700CC30142DBA46F1D80E0F2", txResponse["txhash"])
	require.Equal(t, "26864566", txResponse["height"])
	require.Equal(t, float64(0), txResponse["code"])

	// Verify logs were reconstructed for BOTH messages
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

	// Should have exactly 2 log entries for the 2 messages
	require.Equal(t, 2, len(logs), "should have exactly 2 log entries for multi-message tx")

	// Verify first log (msg_index: 0 - MsgAggregateExchangeRateVote)
	log0 := logs[0]
	msgIndex0, _ := log0["msg_index"].(float64)
	require.Equal(t, 0, int(msgIndex0), "first log should have msg_index 0")
	require.Equal(t, "", log0["log"])

	var logEvents0 []map[string]interface{}
	switch e := log0["events"].(type) {
	case []map[string]interface{}:
		logEvents0 = e
	case []interface{}:
		logEvents0 = make([]map[string]interface{}, len(e))
		for i, v := range e {
			logEvents0[i] = v.(map[string]interface{})
		}
	}

	// Verify event types for msg_index 0
	eventTypes0 := make(map[string]bool)
	for _, event := range logEvents0 {
		if eventType, ok := event["type"].(string); ok {
			eventTypes0[eventType] = true
		}
	}
	require.True(t, eventTypes0["message"], "msg_index 0 should have 'message' event type")
	require.True(t, eventTypes0["aggregate_vote"], "msg_index 0 should have 'aggregate_vote' event type")
	require.True(t, eventTypes0["tx"], "msg_index 0 should have 'tx' event type")

	// Verify second log (msg_index: 1 - MsgAggregateExchangeRatePrevote)
	log1 := logs[1]
	msgIndex1, _ := log1["msg_index"].(float64)
	require.Equal(t, 1, int(msgIndex1), "second log should have msg_index 1")
	require.Equal(t, "", log1["log"])

	var logEvents1 []map[string]interface{}
	switch e := log1["events"].(type) {
	case []map[string]interface{}:
		logEvents1 = e
	case []interface{}:
		logEvents1 = make([]map[string]interface{}, len(e))
		for i, v := range e {
			logEvents1[i] = v.(map[string]interface{})
		}
	}

	// Verify event types for msg_index 1
	eventTypes1 := make(map[string]bool)
	for _, event := range logEvents1 {
		if eventType, ok := event["type"].(string); ok {
			eventTypes1[eventType] = true
		}
	}
	require.True(t, eventTypes1["message"], "msg_index 1 should have 'message' event type")
	require.True(t, eventTypes1["aggregate_prevote"], "msg_index 1 should have 'aggregate_prevote' event type")

	// Verify aggregate_vote event in msg_index 0 has correct attributes
	var aggregateVoteEvent map[string]interface{}
	for _, event := range logEvents0 {
		if event["type"] == "aggregate_vote" {
			aggregateVoteEvent = event
			break
		}
	}
	require.NotNil(t, aggregateVoteEvent, "aggregate_vote event should exist in msg_index 0")

	var voteAttrs []map[string]interface{}
	switch a := aggregateVoteEvent["attributes"].(type) {
	case []map[string]interface{}:
		voteAttrs = a
	case []interface{}:
		voteAttrs = make([]map[string]interface{}, len(a))
		for i, v := range a {
			voteAttrs[i] = v.(map[string]interface{})
		}
	}

	voteAttrMap := make(map[string]string)
	for _, attr := range voteAttrs {
		if key, ok := attr["key"].(string); ok {
			if value, ok := attr["value"].(string); ok {
				voteAttrMap[key] = value
			}
		}
	}

	require.Equal(t, "terravaloper1j5pj3n3m9nxmv9dgl4wnv2yq53k2jf2283j5zm", voteAttrMap["voter"])
	require.Contains(t, voteAttrMap["exchange_rates"], "uaud")
	require.Contains(t, voteAttrMap["exchange_rates"], "uusd")

	// Verify aggregate_prevote event in msg_index 1
	var aggregatePrevoteEvent map[string]interface{}
	for _, event := range logEvents1 {
		if event["type"] == "aggregate_prevote" {
			aggregatePrevoteEvent = event
			break
		}
	}
	require.NotNil(t, aggregatePrevoteEvent, "aggregate_prevote event should exist in msg_index 1")

	var prevoteAttrs []map[string]interface{}
	switch a := aggregatePrevoteEvent["attributes"].(type) {
	case []map[string]interface{}:
		prevoteAttrs = a
	case []interface{}:
		prevoteAttrs = make([]map[string]interface{}, len(a))
		for i, v := range a {
			prevoteAttrs[i] = v.(map[string]interface{})
		}
	}

	prevoteAttrMap := make(map[string]string)
	for _, attr := range prevoteAttrs {
		if key, ok := attr["key"].(string); ok {
			if value, ok := attr["value"].(string); ok {
				prevoteAttrMap[key] = value
			}
		}
	}

	require.Equal(t, "terravaloper1j5pj3n3m9nxmv9dgl4wnv2yq53k2jf2283j5zm", prevoteAttrMap["voter"])
}

func TestTxLogsMiddleware_FailedTransaction(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		response   string
	}{
		{
			name:       "400 Bad Request",
			statusCode: http.StatusBadRequest,
			response:   `{"error": "invalid request"}`,
		},
		{
			name:       "404 Not Found",
			statusCode: http.StatusNotFound,
			response:   `{"error": "tx not found"}`,
		},
		{
			name:       "500 Internal Server Error",
			statusCode: http.StatusInternalServerError,
			response:   `{"error": "internal error"}`,
		},
		{
			name:       "503 Service Unavailable",
			statusCode: http.StatusServiceUnavailable,
			response:   `{"error": "service unavailable"}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.statusCode)
				w.Write([]byte(tc.response))
			})

			handler := TxLogsMiddleware(mockHandler)
			req := httptest.NewRequest(http.MethodGet, "/cosmos/tx/v1beta1/txs/ABCD1234", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			// Verify status code is preserved
			require.Equal(t, tc.statusCode, rec.Code)

			// Verify response is unchanged
			require.Equal(t, tc.response, rec.Body.String())

			// Verify Content-Type header is preserved
			require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
		})
	}
}

func TestTxLogsMiddleware_MalformedJSON(t *testing.T) {
	tests := []struct {
		name     string
		response string
	}{
		{
			name:     "invalid JSON syntax",
			response: `{"tx_response": {"height": "100", invalid}`,
		},
		{
			name:     "empty response",
			response: ``,
		},
		{
			name:     "truncated JSON",
			response: `{"tx_response":`,
		},
		{
			name:     "non-JSON content",
			response: `plain text error`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tc.response))
			})

			handler := TxLogsMiddleware(mockHandler)
			req := httptest.NewRequest(http.MethodGet, "/cosmos/tx/v1beta1/txs/ABCD1234", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			// Verify original response is returned unchanged
			require.Equal(t, tc.response, rec.Body.String())
			require.Equal(t, http.StatusOK, rec.Code)
		})
	}
}

func TestTxLogsMiddleware_SparseMsgIndex(t *testing.T) {
	tests := []struct {
		name               string
		response           string
		expectedLogs       int
		expectedMsgIndices []int
	}{
		{
			name: "sparse msg_index (0 and 5)",
			response: `{
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
								{"key": "action", "value": "delegate", "msg_index": 5}
							]
						}
					]
				}
			}`,
			expectedLogs:       2,
			expectedMsgIndices: []int{0, 5},
		},
		{
			name: "sparse msg_index with large gap (0, 3, 10)",
			response: `{
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
								{"key": "action", "value": "vote", "msg_index": 3}
							]
						},
						{
							"type": "message",
							"attributes": [
								{"key": "action", "value": "delegate", "msg_index": 10}
							]
						}
					]
				}
			}`,
			expectedLogs:       3,
			expectedMsgIndices: []int{0, 3, 10},
		},
		{
			name: "only high msg_index values (7 and 9)",
			response: `{
				"tx": {},
				"tx_response": {
					"height": "100",
					"txhash": "ABCD1234",
					"logs": [],
					"events": [
						{
							"type": "message",
							"attributes": [
								{"key": "action", "value": "send", "msg_index": 7}
							]
						},
						{
							"type": "message",
							"attributes": [
								{"key": "action", "value": "delegate", "msg_index": 9}
							]
						}
					]
				}
			}`,
			expectedLogs:       2,
			expectedMsgIndices: []int{7, 9},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(tc.response))
			})

			handler := TxLogsMiddleware(mockHandler)
			req := httptest.NewRequest(http.MethodGet, "/cosmos/tx/v1beta1/txs/ABCD1234", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			// Parse response
			var response map[string]interface{}
			err := json.Unmarshal(rec.Body.Bytes(), &response)
			require.NoError(t, err)

			txResponse, ok := response["tx_response"].(map[string]interface{})
			require.True(t, ok, "tx_response should exist")

			// Verify logs count
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

			require.Equal(t, tc.expectedLogs, len(logs), "should have expected number of log entries")

			// Verify msg_index values match expected (and are in order)
			actualMsgIndices := make([]int, len(logs))
			for i, log := range logs {
				msgIndexFloat, ok := log["msg_index"].(float64)
				require.True(t, ok, "msg_index should be present and numeric")
				actualMsgIndices[i] = int(msgIndexFloat)
			}

			require.Equal(t, tc.expectedMsgIndices, actualMsgIndices, "msg_index values should match expected and be in order")
		})
	}
}
