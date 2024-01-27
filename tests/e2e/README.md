# End-to-end Tests

## Structure

### `e2e` Package

The `e2e` package defines an integration testing suite used for full
end-to-end testing functionality. This package is decoupled from
depending on the Terra codebase. It initializes the chains for testing
via Docker files. As a result, the test suite may provide the desired
Terra version to Docker containers during the initialization.

The file e2e\_setup\_test.go defines the testing suite and contains the
core bootstrapping logic that creates a testing environment via Docker
containers. A testing network is created dynamically with 2 test
validators.

The file `e2e_test.go` contains the actual end-to-end integration tests
that utilize the testing suite.

Currently, there are 5 tests in `e2e_test.go`.

Additionally, there is an ability to disable certain components
of the e2e suite. This can be done by setting the environment
variables. See "Environment variables" section below for more details.

## How to run

To run all e2e-tests in the package, run: 

```sh
    make test-e2e
```

## `initialization` Package

The `initialization` package introduces the logic necessary for initializing a
chain by creating a genesis file and all required configuration files
such as the `app.toml`. This package directly depends on the Terra
codebase.

## `upgrade` Package

The `upgrade` package starts chain initialization. In addition, there is
a Dockerfile `init.Dockerfile`. When executed, its container
produces all files necessary for starting up a new chain. These
resulting files can be mounted on a volume and propagated to our
production Terra container to start the `terrad` service.

The decoupling between chain initialization and start-up allows to
minimize the differences between our test suite and the production
environment.

## `containers` Package

Introduces an abstraction necessary for creating and managing
Docker containers. Currently, validator containers are created
with a name of the corresponding validator struct that is initialized
in the `chain` package.

## Running From Current Branch

### To build chain initialization image

```sh
make docker-build-e2e-init-chain
```

### To build the debug Terra image

```sh
    make docker-build-e2e-debug
```

### Environment variables

Some tests take a long time to run. Sometimes, we would like to disable them
locally or in CI. The following are the environment variables to disable
certain components of e2e testing.

- `TERRA_E2E_SKIP_UPGRADE` - when true, skips the upgrade tests.
If TERRA_E2E_SKIP_UPGRADE is true, this must also be set to true because upgrade
tests require IBC logic.

- `TERRA_E2E_SKIP_IBC` - when true, skips the IBC tests tests.

- `TERRA_E2E_SKIP_STATE_SYNC` - when true, skips the state sync tests.

- `TERRA_E2E_SKIP_CLEANUP` - when true, avoids cleaning up the e2e Docker
containers.

- `TERRA_E2E_FORK_HEIGHT` - when the above "IS_FORK" env variable is set to true, this is the string
of the height in which the network should fork. This should match the ForkHeight set in constants.go

- `TERRA_E2E_UPGRADE_VERSION` - string of what version will be upgraded to (for example, "v10")

- `TERRA_E2E_DEBUG_LOG` - when true, prints debug logs from executing CLI commands
via Docker containers. Set to trus in CI by default.

#### VS Code Debug Configuration

This debug configuration helps to run e2e tests locally and skip the desired tests.

```json
{
    "name": "E2E IntegrationTestSuite",
    "type": "go",
    "request": "launch",
    "mode": "test",
    "program": "${workspaceFolder}/tests/e2e",
    "args": [
        "-test.timeout",
        "30m",
        "-test.run",
        "IntegrationTestSuite",
        "-test.v"
    ],
    "buildFlags": "-tags e2e",
    "env": {
        "TERRA_E2E_SKIP_IBC": "true",
        "TERRA_E2E_SKIP_UPGRADE": "true",
        "TERRA_E2E_SKIP_CLEANUP": "true",
        "TERRA_E2E_SKIP_STATE_SYNC": "true",
        "TERRA_E2E_UPGRADE_VERSION": "v5",
        "TERRA_E2E_DEBUG_LOG": "true",
        "TERRA_E2E_FORK_HEIGHT": ""
    }
}
```

### Common Problems

Please note that if the tests are stopped mid-way, the e2e framework might fail to start again due to duplicated containers. Make sure that
containers are removed before running the tests again: `docker containers rm -f $(docker containers ls -a -q)`.

Additionally, Docker networks do not get auto-removed. Therefore, you can manually remove them by running `docker network prune`.
