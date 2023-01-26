package main

import (
	wasmconfig "github.com/terra-money/core/x/wasm/config"

	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
)

// TerraAppConfig terra specify app config
type TerraAppConfig struct {
	serverconfig.Config

	WASMConfig wasmconfig.Config `mapstructure:"wasm"`
}

// initAppConfig helps to override default appConfig template and configs.
// return "", nil if no custom configuration is required for the application.
func initAppConfig() (string, interface{}) {
	// Optionally allow the chain developer to overwrite the SDK's default
	// server config.
	srvCfg := serverconfig.DefaultConfig()

	// The SDK's default minimum gas price is set to "" (empty value) inside
	// app.toml. If left empty by validators, the node will halt on startup.
	// However, the chain developer can set a default app.toml value for their
	// validators here.
	//
	// In summary:
	// - if you leave srvCfg.MinGasPrices = "", all validators MUST tweak their
	//   own app.toml config,
	// - if you set srvCfg.MinGasPrices non-empty, validators CAN tweak their
	//   own app.toml to override, or use this default value.
	//
	// In simapp, we set the min gas prices to 0.
	srvCfg.MinGasPrices = "0uluna"

	// This ensures that Terra upgrades will use IAVL fast node.
	// There's a second order effect: archive nodes will take a veritable long-ass time to upgrade.
	// Reference this history of this file for more information: https://github.com/evmos/evmos/blob/1ca54a4e1c0812933960a9c943a7ab6c4901210d/cmd/evmosd/root.go

	srvCfg.IAVLDisableFastNode = false

	terraAppConfig := TerraAppConfig{
		Config:     *srvCfg,
		WASMConfig: *wasmconfig.DefaultConfig(),
	}

	terraAppTemplate := serverconfig.DefaultConfigTemplate + wasmconfig.DefaultConfigTemplate

	return terraAppTemplate, terraAppConfig
}
