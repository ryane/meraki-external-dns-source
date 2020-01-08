module github.com/ryane/meraki-external-dns-source

go 1.13

require (
	github.com/go-logr/logr v0.1.0
	github.com/golang/groupcache v0.0.0-20190129154638-5b532d6fd5ef // indirect
	github.com/imdario/mergo v0.3.7 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/kubernetes-incubator/external-dns v0.5.12
	github.com/onsi/ginkgo v1.10.1
	github.com/onsi/gomega v1.7.0
	github.com/prometheus/client_golang v0.9.3-0.20190127221311-3c4408c8b829 // indirect
	github.com/prometheus/client_model v0.0.0-20190129233127-fd36f4220a90 // indirect
	github.com/prometheus/procfs v0.0.0-20190403104016-ea9eea638872 // indirect
	github.com/sirupsen/logrus v1.4.2
	golang.org/x/sys v0.0.0-20191010194322-b09406accb47 // indirect

	k8s.io/api v0.17.0
	k8s.io/apimachinery v0.17.0
	k8s.io/client-go v0.17.0
	sigs.k8s.io/controller-runtime v0.4.0
)

replace k8s.io/code-generator v0.0.0-20190409092313-b1289fc74931 => k8s.io/code-generator v0.0.0-20181128191024-b1289fc74931
