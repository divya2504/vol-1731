module github.com/opencord/voltha-api-server

go 1.12

require (
	github.com/Shopify/sarama v1.21.0 // indirect
	github.com/bsm/sarama-cluster v2.1.15+incompatible // indirect
	github.com/golang/protobuf v1.4.2
	github.com/golang/snappy v0.0.1 // indirect
	github.com/google/uuid v1.1.1
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/opencord/voltha-go v2.2.0+incompatible
	github.com/opencord/voltha-protos v1.0.3
	github.com/stretchr/testify v1.4.0
	go.uber.org/atomic v1.3.2 // indirect
	go.uber.org/multierr v1.1.0 // indirect
	go.uber.org/zap v1.9.1 // indirect
	golang.org/x/net v0.0.0-20200707034311-ab3426394381
	google.golang.org/grpc v1.27.0
	k8s.io/api v0.20.0-alpha.2
	k8s.io/apimachinery v0.20.0-alpha.2
	k8s.io/client-go v0.20.0-alpha.2 // pseudo version corresponding to v12.0.0
)
