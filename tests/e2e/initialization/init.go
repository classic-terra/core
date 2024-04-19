package initialization

import (
	"errors"
	"fmt"
	"path/filepath"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/classic-terra/core/v3/tests/e2e/util"
	coreutil "github.com/classic-terra/core/v3/types/util"
)

func init() {
	SetAddressPrefixes()
}

// SetAddressPrefixes builds the Config with Bech32 addressPrefix and publKeyPrefix for accounts, validators, and consensus nodes and verifies that addreeses have correct format.
func SetAddressPrefixes() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(coreutil.Bech32PrefixAccAddr, coreutil.Bech32PrefixAccPub)
	config.SetBech32PrefixForValidator(coreutil.Bech32PrefixValAddr, coreutil.Bech32PrefixValPub)
	config.SetBech32PrefixForConsensusNode(coreutil.Bech32PrefixConsAddr, coreutil.Bech32PrefixConsPub)

	// This is copied from the cosmos sdk v0.43.0-beta1
	// source: https://github.com/cosmos/cosmos-sdk/blob/v0.43.0-beta1/types/address.go#L141
	config.SetAddressVerifier(func(bytes []byte) error {
		if len(bytes) == 0 {
			return sdkerrors.Wrap(sdkerrors.ErrUnknownAddress, "addresses cannot be empty")
		}

		if len(bytes) > address.MaxAddrLen {
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownAddress, "address max length is %d, got %d", address.MaxAddrLen, len(bytes))
		}

		// TODO: Do we want to allow addresses of lengths other than 20 and 32 bytes?
		if len(bytes) != 20 && len(bytes) != 32 {
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownAddress, "address length must be 20 or 32 bytes, got %d", len(bytes))
		}

		return nil
	})
}

func InitChain(id, dataDir string, nodeConfigs []*NodeConfig, forkHeight int) (*Chain, error) {
	chain, err := new(id, dataDir)
	if err != nil {
		return nil, err
	}

	for _, nodeConfig := range nodeConfigs {
		newNode, err := newNode(chain, nodeConfig)
		if err != nil {
			return nil, err
		}
		chain.nodes = append(chain.nodes, newNode)
	}

	if err := initGenesis(chain, forkHeight); err != nil {
		return nil, err
	}

	var peers []string
	for _, peer := range chain.nodes {
		peerID := fmt.Sprintf("%s@%s:26656", peer.getNodeKey().ID(), peer.moniker)
		peer.peerID = peerID
		peers = append(peers, peerID)
	}

	for _, node := range chain.nodes {
		if node.isValidator {
			if err := node.initNodeConfigs(peers); err != nil {
				return nil, err
			}
		}
	}
	return chain.export(), nil
}

func InitSingleNode(chainID, dataDir string, existingGenesisDir string, nodeConfig *NodeConfig, trustHeight int64, trustHash string, stateSyncRPCServers []string, persistentPeers []string) (*Node, error) {
	if nodeConfig.IsValidator {
		return nil, errors.New("creating individual validator nodes after starting up chain is not currently supported")
	}

	chain, err := new(chainID, dataDir)
	if err != nil {
		return nil, err
	}

	newNode, err := newNode(chain, nodeConfig)
	if err != nil {
		return nil, err
	}

	_, err = util.CopyFile(
		existingGenesisDir,
		filepath.Join(newNode.configDir(), "config", "genesis.json"),
	)
	if err != nil {
		return nil, err
	}

	if err := newNode.initNodeConfigs(persistentPeers); err != nil {
		return nil, err
	}

	if err := newNode.initStateSyncConfig(trustHeight, trustHash, stateSyncRPCServers); err != nil {
		return nil, err
	}

	return newNode.export(), nil
}
