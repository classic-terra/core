package helpers

import (
	"encoding/json"
	"fmt"
	"strconv"

	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func UnmarshalValidators(config simappparams.EncodingConfig, data []byte) (stakingtypes.Validators, []cryptotypes.PubKey, error) {
	var validators stakingtypes.Validators
	var pubKeys []cryptotypes.PubKey

	var tmp map[string]interface{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return nil, nil, err
	}

	tmpValidators, ok := tmp["validators"].([]interface{})
	if !ok {
		return nil, nil, fmt.Errorf("invalid validators field")
	}

	for _, v := range tmpValidators {
		validator, ok := v.(map[string]interface{})
		if !ok {
			return nil, nil, fmt.Errorf("invalid validator")
		}

		status, ok := validator["status"].(string)
		if !ok {
			return nil, nil, fmt.Errorf("invalid BondStatus")
		}
		delete(validator, "status")

		unbondingHeight, ok := validator["unbonding_height"].(string)
		if !ok {
			return nil, nil, fmt.Errorf("invalid UnbondingHeight")
		}
		delete(validator, "unbonding_height")

		concensusPubkey, ok := validator["consensus_pubkey"].(map[string]interface{})
		if !ok {
			return nil, nil, fmt.Errorf("invalid consensus_pubkey")
		}
		delete(validator, "consensus_pubkey")

		// Encode the validator without the BondStatus field
		bz, err := json.Marshal(validator)
		if err != nil {
			return nil, nil, err
		}

		var val stakingtypes.Validator
		err = json.Unmarshal(bz, &val)
		if err != nil {
			return nil, nil, err
		}

		// Find the status field and convert it to BondStatus
		switch status {
		case "BOND_STATUS_UNSPECIFIED":
			val.Status = stakingtypes.Unspecified
		case "BOND_STATUS_UNBONDED":
			val.Status = stakingtypes.Unbonded
		case "BOND_STATUS_UNBONDING":
			val.Status = stakingtypes.Unbonding
		case "BOND_STATUS_BONDED":
			val.Status = stakingtypes.Bonded
		}

		// Convert UnbondingHeight to int64
		unbondingHeightInt, err := strconv.ParseInt(unbondingHeight, 10, 64)
		if err != nil {
			return nil, nil, err
		}
		val.UnbondingHeight = unbondingHeightInt

		// Convert consensus_pubkey to PubKey
		concensusPubkeyBz, err := json.Marshal(concensusPubkey)
		if err != nil {
			return nil, nil, err
		}
		var pk cryptotypes.PubKey
		err = config.Codec.UnmarshalInterfaceJSON(concensusPubkeyBz, &pk)
		if err != nil {
			return nil, nil, err
		}
		validators = append(validators, val)
		pubKeys = append(pubKeys, pk)
	}

	return validators, pubKeys, nil
}

func GetSignedBlocksWindow(data []byte) (int64, error) {
	var tmp map[string]interface{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return 0, err
	}

	signedBlocksWindow, ok := tmp["signed_blocks_window"].(string)
	if !ok {
		return 0, fmt.Errorf("invalid signed_blocks_window")
	}

	return strconv.ParseInt(signedBlocksWindow, 10, 64)
}
