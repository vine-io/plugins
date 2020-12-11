module github.com/lack-io/plugins

go 1.15

require (
	github.com/coreos/etcd v0.0.0-00010101000000-000000000000
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/gogo/protobuf v1.3.1
	github.com/json-iterator/go v1.1.10
	github.com/lack-io/vine v0.2.1
	github.com/mitchellh/hashstructure v1.1.0
	google.golang.org/grpc v1.34.0
)

replace (
	github.com/coreos/etcd => github.com/coreos/etcd v3.3.18+incompatible
	google.golang.org/grpc => google.golang.org/grpc v1.26.0
)
