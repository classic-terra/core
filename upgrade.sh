SOFTWARE_UPGRADE_NAME="v14rc4"
UPGRADE_HEIGHT=29667824

UPGRADE_INFO=$(jq -n '
  {
      "binaries": {
          "linux/amd64": "https://github.com/classic-terra/core/releases/download/v4.0.0-rc.4/terra_4.0.0-rc.4_Linux_x86_64.tar.gz?checksum=sha256:1a17a59f2a1e3c7672524490e08346a03493dcd07d381952208a3bf1dadbc58f"
      }
  }')

GOV_AUTHORITY=$(terrad q auth module-account gov --node https://rpc.luncblaze.com:443 --output json | jq -r '.account.value.address // .account.base_account.address // .account.address')

cat > /tmp/upgrade_proposal.json <<EOF
{
  "messages": [
    {
      "@type": "/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade",
      "authority": "$GOV_AUTHORITY",
      "plan": {
        "name": "$SOFTWARE_UPGRADE_NAME",
        "height": "$UPGRADE_HEIGHT",
        "info": $(echo "$UPGRADE_INFO" | jq -c '. | tostring')
      }
    }
  ],
  "deposit": "50000000uluna",
  "title": "Upgrade to $SOFTWARE_UPGRADE_NAME",
  "summary": "Upgrade to $SOFTWARE_UPGRADE_NAME",
  "metadata": ""
}
EOF

terrad tx gov submit-proposal /tmp/upgrade_proposal.json --from orbit-testnet --keyring-backend os --chain-id "rebel-2" --gas-prices 30uluna --gas auto --gas-adjustment 1.5 --node https://rpc.luncblaze.com:443
