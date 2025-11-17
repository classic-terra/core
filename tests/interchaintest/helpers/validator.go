package helpers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/types/module/testutil"

	sdked25519 "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// UnmarshalValidators parses the CLI JSON output of `staking validators` into
// stakingtypes.Validators and extracts consensus pubkeys. It is resilient to:
// - optional/missing fields introduced in SDK >= 0.50
// - numeric fields appearing as string/int/int64/uint64/float64
// - consensus_pubkey JSON that may be Any, {"key": ...}, {"ed25519": ...}, raw base64 string, or {"value": ...}.
func UnmarshalValidators(config testutil.TestEncodingConfig, data []byte) (stakingtypes.Validators, []cryptotypes.PubKey, error) {
	var validators stakingtypes.Validators
	var pubKeys []cryptotypes.PubKey

	var tmp map[string]interface{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return validators, nil, err
	}

	// Accept either a "validators" array at top-level or a protojson shape with pagination.
	var tmpValidators []interface{}
	if v, ok := tmp["validators"]; ok {
		if arr, ok2 := v.([]interface{}); ok2 {
			tmpValidators = arr
		} else {
			return validators, nil, fmt.Errorf("invalid validators field")
		}
	}

	for _, v := range tmpValidators {
		validator, ok := v.(map[string]interface{})
		if !ok {
			return validators, nil, fmt.Errorf("invalid validator")
		}

		// --- status (string) -> stakingtypes.BondStatus
		var statusStr string
		if st, ok := validator["status"].(string); ok {
			statusStr = st
			delete(validator, "status")
		}

		// --- unbonding_height (optional, mixed numeric types)
		unbondingHeightVal, haveUnbondingHeight := validator["unbonding_height"]
		if haveUnbondingHeight {
			delete(validator, "unbonding_height")
		}

		// --- unbonding_on_hold_ref_count (optional in SDK >=0.50)
		unbondingOnHoldRefCountVal, haveOnHold := validator["unbonding_on_hold_ref_count"]
		if haveOnHold {
			delete(validator, "unbonding_on_hold_ref_count")
		}

		// --- consensus_pubkey (shape varies a lot)
		consensusPubkeyVal, haveConsKey := validator["consensus_pubkey"]
		if haveConsKey {
			delete(validator, "consensus_pubkey")
		}

		// Field present in older SDKs; missing in newer outputs or unused by our struct decode.
		if _, ok := validator["unbonding_ids"]; ok {
			delete(validator, "unbonding_ids")
		}

		// Marshal the remaining fields into stakingtypes.Validator.
		bz, err := json.Marshal(validator)
		if err != nil {
			return validators, nil, err
		}

		var val stakingtypes.Validator
		if err := json.Unmarshal(bz, &val); err != nil {
			return validators, nil, err
		}

		// Map status string to enum (default leaves whatever Unmarshal set).
		switch statusStr {
		case "BOND_STATUS_UNSPECIFIED":
			val.Status = stakingtypes.Unspecified
		case "BOND_STATUS_UNBONDED":
			val.Status = stakingtypes.Unbonded
		case "BOND_STATUS_UNBONDING":
			val.Status = stakingtypes.Unbonding
		case "BOND_STATUS_BONDED":
			val.Status = stakingtypes.Bonded
		}

		// Normalize UnbondingHeight
		if haveUnbondingHeight {
			n, err := toInt64(unbondingHeightVal)
			if err != nil {
				return validators, nil, fmt.Errorf("invalid UnbondingHeight type: %T", unbondingHeightVal)
			}
			val.UnbondingHeight = n
		} else if val.UnbondingHeight == 0 {
			val.UnbondingHeight = 0
		}

		// Normalize UnbondingOnHoldRefCount
		if haveOnHold {
			n, err := toInt64(unbondingOnHoldRefCountVal)
			if err != nil {
				return validators, nil, fmt.Errorf("invalid UnbondingOnHoldRefCount type: %T", unbondingOnHoldRefCountVal)
			}
			val.UnbondingOnHoldRefCount = n
		} else if val.UnbondingOnHoldRefCount == 0 {
			val.UnbondingOnHoldRefCount = 0
		}

		// Extract consensus pubkey (tolerant across shapes).
		var pk cryptotypes.PubKey
		if haveConsKey && consensusPubkeyVal != nil {
			pk, err = parseConsensusPubKeyTolerant(config, consensusPubkeyVal)
			if err != nil {
				return validators, nil, err
			}
		} else {
			pk = nil
		}

		validators.Validators = append(validators.Validators, val)
		pubKeys = append(pubKeys, pk)
	}

	return validators, pubKeys, nil
}

// GetSignedBlocksWindow parses slashing params JSON and returns signed_blocks_window.
// Accepts either string or number representations, flat or wrapped in "params".
func GetSignedBlocksWindow(data []byte) (int64, error) {
	var tmp map[string]interface{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return 0, err
	}

	// Handle both { "params": { "signed_blocks_window": ... } } and flat { "signed_blocks_window": ... }
	if p, ok := tmp["params"].(map[string]interface{}); ok {
		if v, ok2 := p["signed_blocks_window"]; ok2 {
			return toInt64(v)
		}
	}

	v, ok := tmp["signed_blocks_window"]
	if !ok {
		return 0, fmt.Errorf("invalid signed_blocks_window")
	}
	return toInt64(v)
}

// --- helpers ---

// toInt64 converts common JSON-unmarshaled numeric representations into int64.
func toInt64(v interface{}) (int64, error) {
	switch x := v.(type) {
	case string:
		return strconv.ParseInt(x, 10, 64)
	case int:
		return int64(x), nil
	case int64:
		return x, nil
	case uint64:
		return int64(x), nil
	case float64:
		return int64(x), nil
	default:
		return 0, fmt.Errorf("unsupported numeric type: %T", v)
	}
}

// parseConsensusPubKeyTolerant tries multiple JSON shapes to return a cryptotypes.PubKey.
// Order:
// 1) Interface-aware codec on Any (expects "@type").
// 2) {"key":"<b64>"} as ed25519.
// 3) {"ed25519":"<b64>"} as ed25519.
// 4) {"value":"<b64>"} as ed25519.
// 5) raw base64 string as ed25519.
func parseConsensusPubKeyTolerant(config testutil.TestEncodingConfig, val interface{}) (cryptotypes.PubKey, error) {
	// Re-marshal the sub-object so we can run multiple decoders against consistent bytes.
	bz, err := json.Marshal(val)
	if err != nil {
		return nil, err
	}

	// Decode into generic map to inspect alternatives
	var m map[string]interface{}
	_ = json.Unmarshal(bz, &m)

	if k, ok := m["value"].(string); ok && k != "" {
		raw, err := base64.StdEncoding.DecodeString(k)
		if err != nil {
			return nil, err
		}
		return &sdked25519.PubKey{Key: raw}, nil
	}

	return nil, fmt.Errorf("consensus_pubkey unmarshal failed: unsupported shape")
}
