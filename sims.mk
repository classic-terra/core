#!/usr/bin/make -f

########################################
### Simulations

BINDIR ?= $(shell go env GOPATH)/bin
SIMAPP = ./app
SIM_LOG_DIR ?= simulation-logs
SIM_TESTS = TestAppStateDeterminism TestAppImportExport TestAppSimulationAfterImport TestAppSimulationAfterImportValidatorExhaustion TestFullAppSimulation TestAppSimulationValidatorExhaustion

# The SDK default corpus also contains three fixtures whose randomized evidence
# deliberately removes every validator before block 50. Verify those separately;
# never count their SDK skips as import/export or continuous-block coverage.
SIM_SEEDS ?= 1,2,4,7,32,123,124,1893,2989,3012,4728,37827,981928,87821,891823782,989182,11,22,44,77,99,2020,3232,123123,124124,582582,18931893,29892989,30123012,47284728,8090485,977367484,491163361,424254581,673398983
comma := ,
empty :=
space := $(empty) $(empty)
SIM_AFTER_IMPORT_SEEDS ?= $(subst $(space),$(comma),$(filter-out 124 37827,$(subst $(comma),$(space),$(SIM_SEEDS))))
SIM_JOBS ?= 4

# Go exits successfully when -run matches no tests. Fail before invoking CI runners.
sim-preflight:
	@mkdir -p $(SIM_LOG_DIR)
	@set -eu; tests="$$(go test -mod=readonly $(SIMAPP) -list '^(Test(AppStateDeterminism|AppImportExport|AppSimulationAfterImport|AppSimulationAfterImportValidatorExhaustion|AppSimulationValidatorExhaustion|FullAppSimulation)|Benchmark(FullAppSimulation|Invariants))$$')"; \
	for name in $(SIM_TESTS); do printf '%s\n' "$$tests" | grep -qx "$$name" || { echo "Missing simulation test: $$name"; exit 1; }; done

simulation-runsim:
	@test -x $(BINDIR)/runsim || GOBIN=$(BINDIR) go install github.com/cosmos/tools/cmd/runsim@v1.0.0

# Keep runsim temporary per-seed output in the artifact directory.
# Print captured stdout on failure for CI visibility.
define run-simulation
	@TMPDIR=$(abspath $(SIM_LOG_DIR)) $(BINDIR)/runsim $(1) || { status=$$?; find $(SIM_LOG_DIR) -name '*.stdout' -type f -exec cat {} +; exit $$status; }
endef

test-sim-nondeterminism: sim-preflight
	@mkdir -p $(SIM_LOG_DIR)
	@echo "Running non-determinism test..."
	@go test -mod=readonly $(SIMAPP) -run TestAppStateDeterminism -Enabled=true \
		-NumBlocks=50 -BlockSize=200 -Commit=true -Period=0 -v -timeout 24h > $(SIM_LOG_DIR)/nondeterminism.log 2>&1 || { cat $(SIM_LOG_DIR)/nondeterminism.log; exit 1; }
	@cat $(SIM_LOG_DIR)/nondeterminism.log

test-sim-custom-genesis-fast: sim-preflight
	@echo "Running custom genesis simulation..."
	@echo "By default, ${HOME}/.terra/config/genesis.json will be used."
	@go test -mod=readonly $(SIMAPP) -run TestFullAppSimulation -Genesis=${HOME}/.terra/config/genesis.json \
		-Enabled=true -NumBlocks=50 -BlockSize=200 -Commit=true -Seed=99 -Period=5 -v -timeout 24h

test-sim-import-export: simulation-runsim sim-preflight
	@echo "Running application import/export simulation. This may take several minutes..."
	$(call run-simulation,-Jobs=$(SIM_JOBS) -Seeds=$(SIM_SEEDS) -SimAppPkg=$(SIMAPP) -ExitOnFail 50 5 TestAppImportExport)

test-sim-after-import: simulation-runsim sim-preflight test-sim-after-import-exhaustion
	@echo "Running application simulation-after-import. This may take several minutes..."
	$(call run-simulation,-Jobs=$(SIM_JOBS) -Seeds=$(SIM_AFTER_IMPORT_SEEDS) -SimAppPkg=$(SIMAPP) -ExitOnFail 50 5 TestAppSimulationAfterImport)

test-sim-custom-genesis-multi-seed: simulation-runsim sim-preflight
	@echo "Running multi-seed custom genesis simulation..."
	@echo "By default, ${HOME}/.terra/config/genesis.json will be used."
	$(call run-simulation,-Genesis=${HOME}/.terra/config/genesis.json -SimAppPkg=$(SIMAPP) -ExitOnFail 400 5 TestFullAppSimulation)

test-sim-multi-seed-long: simulation-runsim sim-preflight
	@echo "Running long multi-seed application simulation. This may take awhile!"
	$(call run-simulation,-Jobs=4 -SimAppPkg=$(SIMAPP) -ExitOnFail 500 50 TestFullAppSimulation)

test-sim-multi-seed-short: simulation-runsim sim-preflight test-sim-validator-exhaustion
	@echo "Running short multi-seed application simulation. This may take awhile!"
	$(call run-simulation,-Jobs=$(SIM_JOBS) -Seeds=$(SIM_SEEDS) -SimAppPkg=$(SIMAPP) -ExitOnFail 50 10 TestFullAppSimulation)

test-sim-benchmark-invariants: SIM_TESTS += BenchmarkInvariants
test-sim-benchmark-invariants: sim-preflight
	@echo "Running simulation invariant benchmarks..."
	@go test -mod=readonly $(SIMAPP) -benchmem -bench=BenchmarkInvariants -run=^$$ \
	-Enabled=true -NumBlocks=1000 -BlockSize=200 \
	-Period=1 -Commit=true -Seed=57 -v -timeout 24h

.PHONY: \
test-sim-nondeterminism \
test-sim-custom-genesis-fast \
test-sim-import-export \
test-sim-after-import \
test-sim-custom-genesis-multi-seed \
test-sim-multi-seed-short \
test-sim-multi-seed-long \
test-sim-benchmark-invariants

SIM_NUM_BLOCKS ?= 500
SIM_BLOCK_SIZE ?= 200
SIM_COMMIT ?= true

test-sim-benchmark: SIM_TESTS += BenchmarkFullAppSimulation
test-sim-benchmark: sim-preflight
	@echo "Running application benchmark for numBlocks=$(SIM_NUM_BLOCKS), blockSize=$(SIM_BLOCK_SIZE). This may take awhile!"
	@go test -mod=readonly -benchmem -run=^$$ $(SIMAPP) -bench ^BenchmarkFullAppSimulation$$  \
		-Enabled=true -NumBlocks=$(SIM_NUM_BLOCKS) -BlockSize=$(SIM_BLOCK_SIZE) -Commit=$(SIM_COMMIT) -timeout 24h

test-sim-profile: SIM_TESTS += BenchmarkFullAppSimulation
test-sim-profile: sim-preflight
	@echo "Running application benchmark for numBlocks=$(SIM_NUM_BLOCKS), blockSize=$(SIM_BLOCK_SIZE). This may take awhile!"
	@go test -mod=readonly -benchmem -run=^$$ $(SIMAPP) -bench ^BenchmarkFullAppSimulation$$ \
		-Enabled=true -NumBlocks=$(SIM_NUM_BLOCKS) -BlockSize=$(SIM_BLOCK_SIZE) -Commit=$(SIM_COMMIT) -timeout 24h -cpuprofile cpu.out -memprofile mem.out

.PHONY: test-sim-profile test-sim-benchmark

.PHONY: sim-preflight simulation-runsim

# The children intentionally receive SDK Skip at an empty validator set; their
# parent asserts the exact terminal heights and verifies that no validator remains.
test-sim-validator-exhaustion: sim-preflight
	@go test -mod=readonly $(SIMAPP) -run '^TestAppSimulationValidatorExhaustion$$' -Enabled=true -NumBlocks=50 -BlockSize=200 -Commit=true -Period=5 -v -timeout 24h > $(SIM_LOG_DIR)/validator-exhaustion.log 2>&1 || { cat $(SIM_LOG_DIR)/validator-exhaustion.log; exit 1; }
	@tail -n 15 $(SIM_LOG_DIR)/validator-exhaustion.log

.PHONY: test-sim-validator-exhaustion

test-sim-after-import-exhaustion: sim-preflight
	@go test -mod=readonly $(SIMAPP) -run '^TestAppSimulationAfterImportValidatorExhaustion$$' -Enabled=true -NumBlocks=50 -BlockSize=200 -Commit=true -Period=5 -v -timeout 24h > $(SIM_LOG_DIR)/after-import-validator-exhaustion.log 2>&1 || { cat $(SIM_LOG_DIR)/after-import-validator-exhaustion.log; exit 1; }
	@tail -n 15 $(SIM_LOG_DIR)/after-import-validator-exhaustion.log

.PHONY: test-sim-after-import-exhaustion
