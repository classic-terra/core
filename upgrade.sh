SOFTWARE_UPGRADE_NAME="v14rc4"
UPGRADE_HEIGHT=29667824

UPGRADE_INFO=$(jq -n '
  {
      "binaries": {
          "linux/amd64": "https://github.com/classic-terra/core/releases/download/v4.0.0-rc.4/terra_4.0.0-rc.4_Linux_x86_64.tar.gz?checksum=sha256:1a17a59f2a1e3c7672524490e08346a03493dcd07d381952208a3bf1dadbc58f",
      }
  }')



terrad tx gov submit-legacy-proposal software-upgrade "$SOFTWARE_UPGRADE_NAME" --upgrade-height $UPGRADE_HEIGHT --upgrade-info "$UPGRADE_INFO" --title "Upgrade to v14rc4" --description "Upgrade to v14rc4"  --from orbit-testnet --keyring-backend os --chain-id "rebel-2" --gas-prices 30uluna --gas 3000000 --node https://rpc.luncblaze.com:443