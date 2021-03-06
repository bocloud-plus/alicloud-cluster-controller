module cloudplus.io/alicloud-cluster-controller

go 1.13

require (
	github.com/aliyun/alibaba-cloud-sdk-go v1.61.478
	github.com/go-logr/logr v0.1.0
	github.com/go-logr/zapr v0.1.0
	github.com/google/martian v2.1.0+incompatible
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	github.com/pkg/errors v0.9.1
	go.etcd.io/etcd v0.0.0-20191023171146-3cf2f69b5738
	go.uber.org/zap v1.10.0
	k8s.io/apimachinery v0.17.9
	k8s.io/client-go v0.17.9
	sigs.k8s.io/cluster-api v0.3.9
	sigs.k8s.io/controller-runtime v0.5.10
)
