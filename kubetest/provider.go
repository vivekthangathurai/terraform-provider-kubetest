package kubetest
import (
	"bytes"
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"log"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/mitchellh/go-homedir"
	apimachineryschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	restclient "k8s.io/client-go/rest"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	aggregator "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
)

func Provider() *schema.Provider {
	p := &schema.Provider{
	  ResourcesMap: map[string]*schema.Resource{
		"kubetest_event": resourceEvent(),
	  },
	}
	p.ConfigureContextFunc = func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		return providerConfigure(ctx, d, p.TerraformVersion)
	}
	return p
}


type KubeClientsets interface {
	MainClientset() (*kubernetes.Clientset, error)
	AggregatorClientset() (*aggregator.Clientset, error)
}

type kubeClientsets struct {
	config              *restclient.Config
	mainClientset       *kubernetes.Clientset
	aggregatorClientset *aggregator.Clientset
}

func (k kubeClientsets) MainClientset() (*kubernetes.Clientset, error) {
	if k.mainClientset != nil {
		return k.mainClientset, nil
	}
	if k.config != nil {
		kc, err := kubernetes.NewForConfig(k.config)
		if err != nil {
			return nil, fmt.Errorf("Failed to configure client: %s", err)
		}
		k.mainClientset = kc
	}
	return k.mainClientset, nil
}

func (k kubeClientsets) AggregatorClientset() (*aggregator.Clientset, error) {
	if k.aggregatorClientset != nil {
		return k.aggregatorClientset, nil
	}
	if k.config != nil {
		ac, err := aggregator.NewForConfig(k.config)
		if err != nil {
			return nil, fmt.Errorf("Failed to configure client: %s", err)
		}
		k.aggregatorClientset = ac
	}
	return k.aggregatorClientset, nil
}

func providerConfigure(ctx context.Context, d *schema.ResourceData, terraformVersion string) (interface{}, diag.Diagnostics) {

	// Config initialization
	cfg, err := initializeConfiguration(d)
	if err != nil {
		return nil, diag.FromErr(err)
	}
	if cfg == nil {
		// This is a TEMPORARY measure to work around https://github.com/hashicorp/terraform/issues/24055
		// IMPORTANT: this will NOT enable a workaround of issue: https://github.com/hashicorp/terraform/issues/4149
		// IMPORTANT: if the supplied configuration is incomplete or invalid
		///IMPORTANT: provider operations will fail or attempt to connect to localhost endpoints
		cfg = &restclient.Config{}
	}

	cfg.UserAgent = fmt.Sprintf("HashiCorp/1.0 Terraform/%s", terraformVersion)

	if logging.IsDebugOrHigher() {
		log.Printf("[DEBUG] Enabling HTTP requests/responses tracing")
		cfg.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
			return logging.NewTransport("Kubernetes", rt)
		}
	}

	m := kubeClientsets{
		config:              cfg,
		mainClientset:       nil,
		aggregatorClientset: nil,
	}
	return m, diag.Diagnostics{}
}

func initializeConfiguration(d *schema.ResourceData) (*restclient.Config, error) {
	overrides := &clientcmd.ConfigOverrides{}
	loader := &clientcmd.ClientConfigLoadingRules{}

	if d.Get("load_config_file").(bool) {
		log.Printf("[DEBUG] Trying to load configuration from file")
		if configPath, ok := d.GetOk("config_path"); ok && configPath.(string) != "" {
			path, err := homedir.Expand(configPath.(string))
			if err != nil {
				return nil, err
			}
			log.Printf("[DEBUG] Configuration file is: %s", path)
			loader.ExplicitPath = path

			ctxSuffix := "; default context"

			kubectx, ctxOk := d.GetOk("config_context")
			authInfo, authInfoOk := d.GetOk("config_context_auth_info")
			cluster, clusterOk := d.GetOk("config_context_cluster")
			if ctxOk || authInfoOk || clusterOk {
				ctxSuffix = "; overriden context"
				if ctxOk {
					overrides.CurrentContext = kubectx.(string)
					ctxSuffix += fmt.Sprintf("; config ctx: %s", overrides.CurrentContext)
					log.Printf("[DEBUG] Using custom current context: %q", overrides.CurrentContext)
				}

				overrides.Context = clientcmdapi.Context{}
				if authInfoOk {
					overrides.Context.AuthInfo = authInfo.(string)
					ctxSuffix += fmt.Sprintf("; auth_info: %s", overrides.Context.AuthInfo)
				}
				if clusterOk {
					overrides.Context.Cluster = cluster.(string)
					ctxSuffix += fmt.Sprintf("; cluster: %s", overrides.Context.Cluster)
				}
				log.Printf("[DEBUG] Using overidden context: %#v", overrides.Context)
			}
		}
	}

	// Overriding with static configuration
	if v, ok := d.GetOk("insecure"); ok {
		overrides.ClusterInfo.InsecureSkipTLSVerify = v.(bool)
	}
	if v, ok := d.GetOk("cluster_ca_certificate"); ok {
		overrides.ClusterInfo.CertificateAuthorityData = bytes.NewBufferString(v.(string)).Bytes()
	}
	if v, ok := d.GetOk("client_certificate"); ok {
		overrides.AuthInfo.ClientCertificateData = bytes.NewBufferString(v.(string)).Bytes()
	}
	if v, ok := d.GetOk("host"); ok {
		// Server has to be the complete address of the kubernetes cluster (scheme://hostname:port), not just the hostname,
		// because `overrides` are processed too late to be taken into account by `defaultServerUrlFor()`.
		// This basically replicates what defaultServerUrlFor() does with config but for overrides,
		// see https://github.com/kubernetes/client-go/blob/v12.0.0/rest/url_utils.go#L85-L87
		hasCA := len(overrides.ClusterInfo.CertificateAuthorityData) != 0
		hasCert := len(overrides.AuthInfo.ClientCertificateData) != 0
		defaultTLS := hasCA || hasCert || overrides.ClusterInfo.InsecureSkipTLSVerify
		host, _, err := restclient.DefaultServerURL(v.(string), "", apimachineryschema.GroupVersion{}, defaultTLS)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse host: %s", err)
		}

		overrides.ClusterInfo.Server = host.String()
	}
	if v, ok := d.GetOk("username"); ok {
		overrides.AuthInfo.Username = v.(string)
	}
	if v, ok := d.GetOk("password"); ok {
		overrides.AuthInfo.Password = v.(string)
	}
	if v, ok := d.GetOk("client_key"); ok {
		overrides.AuthInfo.ClientKeyData = bytes.NewBufferString(v.(string)).Bytes()
	}
	if v, ok := d.GetOk("token"); ok {
		overrides.AuthInfo.Token = v.(string)
	}

	if v, ok := d.GetOk("exec"); ok {
		exec := &clientcmdapi.ExecConfig{}
		if spec, ok := v.([]interface{})[0].(map[string]interface{}); ok {
			exec.APIVersion = spec["api_version"].(string)
			exec.Command = spec["command"].(string)
			exec.Args = expandStringSlice(spec["args"].([]interface{}))
			for kk, vv := range spec["env"].(map[string]interface{}) {
				exec.Env = append(exec.Env, clientcmdapi.ExecEnvVar{Name: kk, Value: vv.(string)})
			}
		} else {
			return nil, fmt.Errorf("Failed to parse exec")
		}
		overrides.AuthInfo.Exec = exec
	}

	cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides)
	cfg, err := cc.ClientConfig()
	if err != nil {
		log.Printf("[WARN] Invalid provider configuration was supplied. Provider operations likely to fail: %v", err)
		return nil, nil
	}

	log.Printf("[INFO] Successfully initialized config")
	return cfg, nil
}

var useadmissionregistrationv1beta1 *bool

func useAdmissionregistrationV1beta1(conn *kubernetes.Clientset) (bool, error) {
	if useadmissionregistrationv1beta1 != nil {
		return *useadmissionregistrationv1beta1, nil
	}

	d := conn.Discovery()

	group := "admissionregistration.k8s.io"

	v1, err := apimachineryschema.ParseGroupVersion(fmt.Sprintf("%s/v1", group))
	if err != nil {
		return false, err
	}

	err = discovery.ServerSupportsVersion(d, v1)
	if err == nil {
		log.Printf("[INFO] Using %s/v1", group)
		useadmissionregistrationv1beta1 = ptrToBool(false)
		return false, nil
	}

	v1beta1, err := apimachineryschema.ParseGroupVersion(fmt.Sprintf("%s/v1beta1", group))
	if err != nil {
		return false, err
	}

	err = discovery.ServerSupportsVersion(d, v1beta1)
	if err != nil {
		return false, err
	}

	log.Printf("[INFO] Using %s/v1beta1", group)
	useadmissionregistrationv1beta1 = ptrToBool(true)
	return true, nil
}

func ptrToBool(b bool) *bool {
	return &b
}

func expandStringSlice(s []interface{}) []string {
	result := make([]string, len(s), len(s))
	for k, v := range s {
		// Handle the Terraform parser bug which turns empty strings in lists to nil.
		if v == nil {
			result[k] = ""
		} else {
			result[k] = v.(string)
		}
	}
	return result
}
