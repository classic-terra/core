#!/usr/bin/make -f

########################################
### Simulations
e2e-help:
	@echo "e2e subcommands"
	@echo ""
	@echo "Usage:"
	@echo "  make e2e-[command]"
	@echo ""
	@echo "Available Commands:"
	@echo "  e2e-build                    		   Build e2e debug Docker image"
	@echo "  setup                                 Set up e2e environment"

e2e-setup: e2e-build
	@echo Finished e2e environment setup, ready to start the test
	
e2e-build: 
	@DOCKER_BUILDKIT=1 docker build -t core:local -f local.Dockerfile .
	