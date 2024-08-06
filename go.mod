module github.com/temporalio/omes

go 1.21

toolchain go1.22.5

require (
	github.com/gogo/protobuf v1.3.2
	github.com/golang/protobuf v1.5.4
	github.com/google/uuid v1.6.0
	github.com/prometheus/client_golang v1.16.0
	github.com/spf13/cobra v1.7.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.9.0
	github.com/temporalio/features v0.0.0-20240806202554-bdfe567c9d89
	go.temporal.io/api v1.36.0
	go.temporal.io/sdk v1.28.1
	go.uber.org/zap v1.27.0
	golang.org/x/mod v0.20.0
	golang.org/x/sync v0.8.0
	golang.org/x/sys v0.23.0
	google.golang.org/protobuf v1.34.2
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/prometheus/client_model v0.4.0 // indirect
	github.com/prometheus/common v0.44.0 // indirect
	github.com/prometheus/procfs v0.11.1 // indirect
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/facebookgo/clock v0.0.0-20150410010913-600d898af40a // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware v1.4.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.21.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/nexus-rpc/sdk-go v0.0.9 // indirect
	github.com/pborman/uuid v1.2.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/robfig/cron v1.2.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/exp v0.0.0-20240719175910-8a7402abbf56 // indirect
	golang.org/x/net v0.28.0 // indirect
	golang.org/x/text v0.17.0 // indirect
	golang.org/x/time v0.6.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240805194559-2c9e96a0b5d4 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240805194559-2c9e96a0b5d4 // indirect
	google.golang.org/grpc v1.65.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// This is dumb, but necesary because Go (for some commands) can't figure out the transitive
// local-replace inside of the features module itself, so we have to help it.
replace (
	github.com/temporalio/features/features => github.com/temporalio/features/features v0.0.0-20240806202554-bdfe567c9d89
	github.com/temporalio/features/harness/go => github.com/temporalio/features/harness/go v0.0.0-20240806202554-bdfe567c9d89
)
