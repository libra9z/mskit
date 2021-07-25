module github.com/libra9z/mskit

go 1.16

require (
	github.com/go-kit/kit v0.11.0
	github.com/go-playground/validator/v10 v10.7.0
	github.com/goccy/go-json v0.7.4
	github.com/golang/protobuf v1.5.2
	github.com/hashicorp/consul/api v1.9.1
	github.com/json-iterator/go v1.1.11
	github.com/libra9z/httprouter v0.0.0-00010101000000-000000000000
	github.com/libra9z/utils v1.0.3
	github.com/nacos-group/nacos-sdk-go v1.0.8
	github.com/opentracing/opentracing-go v1.2.0
	github.com/openzipkin/zipkin-go v0.2.5
	github.com/rcrowley/go-metrics v0.0.0-20201227073835-cf1acfcdf475
	github.com/smallnest/rpcx v1.6.4
	github.com/stretchr/testify v1.7.0
	github.com/ugorji/go/codec v1.2.6
	google.golang.org/grpc/examples v0.0.0-20210715165331-ce7bdf50abb1 // indirect
	gopkg.in/yaml.v2 v2.3.0
)

replace github.com/libra9z/httprouter => ../httprouter
