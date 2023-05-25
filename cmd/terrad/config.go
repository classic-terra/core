package main

import (
	"fmt"
//	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
)

const (
	defaultMemoryCacheSize    uint32 = 100 // in MiB
	defaultSmartQueryGasLimit uint64 = 3_000_000
	defaultContractDebugMode         = false
)


type WasmConfig struct {
	// SimulationGasLimit is the max gas to be used in a tx simulation call.
	// When not set the consensus max block gas is used instead
	SimulationGasLimit *uint64 `mapstructure:"simulation_gas_limit"`
	// SmartQueryGasLimit is the max gas to be used in a smart query contract call
	SmartQueryGasLimit uint64 `mapstructure:"query_gas_limit"`
	// MemoryCacheSize in MiB not bytes
	MemoryCacheSize uint32 `mapstructure:"memory_cache_size"`
	// ContractDebugMode log what contract print
	ContractDebugMode bool
}

// TerraAppConfig terra specify app config
type TerraAppConfig struct {
	serverconfig.Config
	Wasm wasmtypes.WasmConfig `mapstructure:"wasm"`
}

// DefaultWasmConfig returns the default settings for WasmConfig
func DefaultWasmConfig() WasmConfig {
	return WasmConfig{
		SmartQueryGasLimit: defaultSmartQueryGasLimit,
		MemoryCacheSize:    defaultMemoryCacheSize,
		ContractDebugMode:  defaultContractDebugMode,
	}
}

// ConfigTemplate toml snippet for app.toml
func WasmConfigTemplate(c WasmConfig) string {
	simGasLimit := `# simulation_gas_limit =`
	if c.SimulationGasLimit != nil {
		simGasLimit = fmt.Sprintf(`simulation_gas_limit = %d`, *c.SimulationGasLimit)
	}

	return fmt.Sprintf(`

###############################################################################
###                                  WASM                                   ###
###############################################################################

[wasm]
# Smart query gas limit is the max gas to be used in a smart query contract call
query_gas_limit = %d

# in-memory cache for Wasm contracts. Set to 0 to disable.
# The value is in MiB not bytes
memory_cache_size = %d

# Simulation gas limit is the max gas to be used in a tx simulation call.
# When not set the consensus max block gas is used instead
%s
`, c.SmartQueryGasLimit, c.MemoryCacheSize, simGasLimit)
}

// DefaultConfigTemplate toml snippet with default values for app.toml
func DefaultWasmConfigTemplate() string {
	return WasmConfigTemplate(DefaultWasmConfig())
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

	terraAppConfig := TerraAppConfig{
		Config:     *srvCfg,
		Wasm: wasmtypes.DefaultWasmConfig(),
	}

	terraAppTemplate := serverconfig.DefaultConfigTemplate + DefaultWasmConfigTemplate()

	return terraAppTemplate, terraAppConfig
}
