module github.com/vivekthangathurai/demo-terraform-provider

go 1.14

require (
	github.com/gophercloud/gophercloud v0.14.0 // indirect
	github.com/hashicorp/nomad v0.12.9
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.3.0
	github.com/hashicorp/terraform-provider-kubernetes v1.13.3 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	k8s.io/api v0.19.1
	k8s.io/apimachinery v0.19.1
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/klog v1.0.0 // indirect
	k8s.io/kube-aggregator v0.19.1
)

replace (
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.19.1
	k8s.io/client-go => k8s.io/client-go v0.19.1
)
