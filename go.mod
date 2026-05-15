module github.com/temporalio/omes

go 1.25.0

require (
	github.com/gogo/protobuf v1.3.2
	github.com/golang/protobuf v1.5.4
	github.com/google/uuid v1.6.0
	github.com/parquet-go/parquet-go v0.25.1
	github.com/prometheus/client_golang v1.16.0
	github.com/shirou/gopsutil/v4 v4.25.10
	github.com/spf13/cobra v1.7.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.11.1
	github.com/temporalio/features v0.0.0-20260427223549-86e4c0deedd7
	github.com/temporalio/omes/workers/go/harness/api v0.0.0-00010101000000-000000000000
	go.temporal.io/api v1.62.11
	go.temporal.io/sdk v1.43.0
	go.uber.org/zap v1.27.0
	golang.org/x/mod v0.31.0
	golang.org/x/sync v0.19.0
	google.golang.org/grpc v1.79.3
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/prometheus/client_model v0.4.0 // indirect
	github.com/prometheus/common v0.44.0
	github.com/prometheus/procfs v0.11.1 // indirect
)

require (
	github.com/andybalholm/brotli v1.1.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/ebitengine/purego v0.9.0 // indirect
	github.com/facebookgo/clock v0.0.0-20150410010913-600d898af40a // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.3.2 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.22.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/nexus-rpc/sdk-go v0.6.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.21 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/robfig/cron v1.2.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/tklauser/go-sysconf v0.3.15 // indirect
	github.com/tklauser/numcpus v0.10.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	golang.org/x/time v0.6.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260120221211-b8f7ae30c516 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260120221211-b8f7ae30c516 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// This is dumb, but necesary because Go (for some commands) can't figure out the transitive
// local-replace inside of the features module itself, so we have to help it.
replace (
	github.com/temporalio/features/features => github.com/temporalio/features/features v0.0.0-20260324215619-e5868d9ba03f
	github.com/temporalio/features/harness/go => github.com/temporalio/features/harness/go v0.0.0-20260324215619-e5868d9ba03f
	github.com/temporalio/omes/workers/go/harness/api => ./workers/go/harness/api
)
