module hpa-tuner

go 1.15

require (
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.1
	github.com/prometheus/common v0.4.1
	go.uber.org/zap v1.10.0
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	k8s.io/api v0.17.2
	k8s.io/apimachinery v0.17.2
	k8s.io/client-go v0.17.2
	sigs.k8s.io/controller-runtime v0.5.0
)
