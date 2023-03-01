go 1.18

module github.com/classic-terra/core

require (
	github.com/cosmos/cosmos-sdk v0.45.12
	github.com/CosmWasm/wasmd v0.30.0
	github.com/CosmWasm/wasmvm v1.1.1
	github.com/cosmos/gogoproto v1.4.4
	github.com/cosmos/ibc-go/v4 v4.3.0
	github.com/gogo/protobuf v1.3.3
	github.com/golang/protobuf v1.5.2
	github.com/google/gofuzz v1.2.0
	github.com/gorilla/mux v1.8.0
	github.com/grpc-ecosystem/grpc-gateway v1.16.0
	github.com/pkg/errors v0.9.1
	github.com/rakyll/statik v0.1.7
	github.com/spf13/cast v1.5.0
	github.com/spf13/cobra v1.6.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.8.1
	github.com/tendermint/tendermint v0.34.24
	github.com/tendermint/tm-db v0.6.7
	google.golang.org/genproto v0.0.0-20221118155620-16455021b5e6
	google.golang.org/grpc v1.52.3
	gopkg.in/yaml.v2 v2.4.0
)



replace (
	github.com/99designs/keyring => github.com/cosmos/keyring v1.1.7-0.20210622111912-ef00f8ac3d76
	github.com/aws/aws-sdk-go v1.25.48 => github.com/aws/aws-sdk-go v1.33.0
	github.com/aws/aws-sdk-go v1.27.0 => github.com/aws/aws-sdk-go v1.33.0
	github.com/confio/ics23/go => github.com/cosmos/cosmos-sdk/ics23/go v0.8.0
	github.com/cosmos/cosmos-sdk => github.com/terra-rebels/cosmos-sdk v0.45.10-rebels.0.20221119081420-65efd50239be
	github.com/cosmos/ledger-cosmos-go => github.com/terra-money/ledger-terra-go v0.11.2
	github.com/docker/docker v1.4.2-0.20180625184442-8e610b2b55bf => github.com/docker/docker v1.6.1
	github.com/ethereum/go-ethereum v1.9.25 => github.com/ethereum/go-ethereum v1.10.9
	github.com/gin-gonic/gin v1.4.0 => github.com/gin-gonic/gin v1.7.7
	github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1
	github.com/libp2p/go-buffer-pool v0.0.2 => github.com/libp2p/go-buffer-pool v0.1.0
	github.com/microcosm-cc/bluemonday v1.0.2 => github.com/microcosm-cc/bluemonday v1.0.16
	github.com/miekg/dns v1.0.14 => github.com/miekg/dns v1.1.25
	github.com/tendermint/tendermint => github.com/terra-money/tendermint v0.34.14-terra.3
	github.com/tidwall/gjson v1.6.7 => github.com/tidwall/gjson v1.9.3
	google.golang.org/grpc => google.golang.org/grpc v1.33.2
)
