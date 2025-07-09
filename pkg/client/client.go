package client

import (
	"os"
	"os/user"
	"path/filepath"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
)

type Option func(*builder)

func WithQPS(q float32) Option {
	return func(b *builder) {
		b.qps = q
	}
}

func WithBurst(b int) Option {
	return func(bu *builder) {
		bu.burst = b
	}
}

// ClientBuilder is an interface for building Kubernetes clients.
type ClientBuilder interface {
	GetClient() (kubernetes.Interface, error)
	GetMetricsClient() (metricsclientset.Interface, error)
	GetDynamicClient() (dynamic.Interface, error)
	GetDiscoveryClient() (discovery.DiscoveryInterface, error)
	ToRawKubeConfigLoader() clientcmd.ClientConfig
	WriteToFile(config clientcmdapi.Config) error
}

type builder struct {
	kubeconfig string
	qps        float32
	burst      int
}

// NewClientBuilder creates a new ClientBuilder with the specified kubeconfig file.
func NewClientBuilder(kubeconfig string, opts ...Option) ClientBuilder {
	b := &builder{kubeconfig: kubeconfig, qps: 20.0, burst: 30}
	for _, o := range opts {
		o(b)
	}
	return b
}

// GetClient returns a Kubernetes client using the specified kubeconfig file.
func (b *builder) GetClient() (kubernetes.Interface, error) {
	cfg, err := b.loadRESTConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(cfg)
}

// GetMetricsClient returns a metrics client using the specified kubeconfig file.
func (b *builder) GetMetricsClient() (metricsclientset.Interface, error) {
	cfg, err := b.loadRESTConfig()
	if err != nil {
		return nil, err
	}
	return metricsclientset.NewForConfig(cfg)
}

// GetDynamicClient returns a dynamic Kubernetes client using the specified kubeconfig file.
func (b *builder) GetDynamicClient() (dynamic.Interface, error) {
	cfg, err := b.loadRESTConfig()
	if err != nil {
		return nil, err
	}
	return dynamic.NewForConfig(cfg)
}

// GetDiscoveryClient returns a discovery client for Kubernetes API discovery using the specified kubeconfig file.
func (b *builder) GetDiscoveryClient() (discovery.DiscoveryInterface, error) {
	cfg, err := b.loadRESTConfig()
	if err != nil {
		return nil, err
	}
	return discovery.NewDiscoveryClientForConfig(cfg)
}

// LoadRESTConfig loads the Kubernetes configuration from the specified kubeconfig
func (b *builder) loadRESTConfig() (*rest.Config, error) {
	cfg, err := b.loadConfig().ClientConfig()
	if err != nil {
		return nil, err
	}
	cfg.QPS = b.qps
	cfg.Burst = b.burst
	return cfg, nil
}

// ToRawKubeConfigLoader loads client config to access kubernetes apiserver
func (b *builder) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return b.loadConfig()
}

// WriteToFile writes the provided Kubernetes raw configuration to the kubeconfig file.
func (b *builder) WriteToFile(config clientcmdapi.Config) error {
	if len(b.kubeconfig) > 0 {
		return clientcmd.WriteToFile(config, b.kubeconfig)
	}
	return clientcmd.ModifyConfig(clientcmd.NewDefaultPathOptions(), config, false)
}

// copy from sigs.k8s.io/controller-runtime/pkg/client/config/config.go
// loadConfig loads a Kubernetes client configuration from the specified kubeconfig file.
// If kubeconfig is empty, it will attempt to load the in-cluster config first,
// and if that fails, it will look for the kubeconfig in the default locations.
func (b *builder) loadConfig() clientcmd.ClientConfig {
	// If a flag is specified with the config location, use that
	if len(b.kubeconfig) > 0 {
		return loadClientConfig(&clientcmd.ClientConfigLoadingRules{ExplicitPath: b.kubeconfig})
	}

	// If the recommended kubeconfig env variable is set, or there
	// is no in-cluster config, try the default recommended locations.
	//
	// For default config file locations, upstream only checks
	// $HOME for the user's home directory, but we can also try
	// os/user.HomeDir when $HOME is unset.
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if _, ok := os.LookupEnv("HOME"); !ok {
		u, err := user.Current()
		if err != nil {
			return nil
		}
		loadingRules.Precedence = append(loadingRules.Precedence, filepath.Join(u.HomeDir, clientcmd.RecommendedHomeDir, clientcmd.RecommendedFileName))
	}
	return loadClientConfig(loadingRules)
}

func loadClientConfig(rules clientcmd.ClientConfigLoader) clientcmd.ClientConfig {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, &clientcmd.ConfigOverrides{})
}
