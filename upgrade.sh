SOFTWARE_UPGRADE_NAME="v13_1"
UPGRADE_HEIGHT=27963601

UPGRADE_INFO=$(jq -n '
  {
      "binaries": {
          "linux/amd64": "https://github.com/classic-terra/core/releases/download/v3.6.1-rc.0/terra_3.6.1-rc.0_Linux_x86_64.tar.gz?checksum=sha256:543dd9d8a65c96ab8d98d3cf49f99035a8bfd474d30d738b68d1032144c9a243",
      }
  }')



terrad tx gov submit-legacy-proposal software-upgrade "$SOFTWARE_UPGRADE_NAME" --upgrade-height $UPGRADE_HEIGHT --upgrade-info "$UPGRADE_INFO" --title "Upgrade to v13_1" --description "Upgrade to v13_1"  --from orbit-testnet --keyring-backend os --chain-id "rebel-2" --gas-prices 30uluna --gas 3000000 --node https://rpc.luncblaze.com:443