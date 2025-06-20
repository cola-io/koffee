package client

import (
	"fmt"
	"log/slog"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
)

// ClientBuilder is an interface for building Kubernetes clients.
type ClientBuilder interface {
	GetClient() (kubernetes.Interface, error)
	GetMetricsClient() (metricsclientset.Interface, error)
	GetDynamicClient() (dynamic.Interface, error)
	GetDiscoveryClient() (discovery.DiscoveryInterface, error)
	LoadRawConfig() (*clientcmdapi.Config, error)
	LoadRESTConfig() (*rest.Config, error)
	WriteToFile(config clientcmdapi.Config) error
}

type builder struct {
	kubeconfig string
}

// NewClientBuilder creates a new ClientBuilder with the specified kubeconfig file.
func NewClientBuilder(kubeconfig string) ClientBuilder {
	return &builder{
		kubeconfig: kubeconfig,
	}
}

// GetClient returns a Kubernetes client using the specified kubeconfig file.
func (b *builder) GetClient() (kubernetes.Interface, error) {
	cfg, err := b.loadConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(cfg)
}

// GetMetricsClient returns a metrics client using the specified kubeconfig file.
func (b *builder) GetMetricsClient() (metricsclientset.Interface, error) {
	cfg, err := b.loadConfig()
	if err != nil {
		return nil, err
	}
	return metricsclientset.NewForConfig(cfg)
}

// GetDynamicClient returns a dynamic Kubernetes client using the specified kubeconfig file.
func (b *builder) GetDynamicClient() (dynamic.Interface, error) {
	cfg, err := b.loadConfig()
	if err != nil {
		return nil, err
	}
	return dynamic.NewForConfig(cfg)
}

// GetDiscoveryClient returns a discovery client for Kubernetes API discovery using the specified kubeconfig file.
func (b *builder) GetDiscoveryClient() (discovery.DiscoveryInterface, error) {
	cfg, err := b.loadConfig()
	if err != nil {
		return nil, err
	}
	return discovery.NewDiscoveryClientForConfig(cfg)
}

// LoadApiConfig loads the Kubernetes raw configuration from the specified kubeconfig file or default locations.
func (b *builder) LoadRawConfig() (*clientcmdapi.Config, error) {
	if len(b.kubeconfig) > 0 {
		return clientcmd.LoadFromFile(b.kubeconfig)
	}
	return clientcmd.NewDefaultPathOptions().GetStartingConfig()
}

// LoadRESTConfig loads the Kubernetes configuration from the specified kubeconfig
func (b *builder) LoadRESTConfig() (*rest.Config, error) {
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
func (b *builder) loadConfig() (config *rest.Config, configErr error) {
	defer func() {
		if config != nil {
			config.QPS = float32(20)
			config.Burst = 30
			config.Timeout = 30 * time.Second
		}
	}()

	// If a flag is specified with the config location, use that
	if len(b.kubeconfig) > 0 {
		return loadConfigWithContext(&clientcmd.ClientConfigLoadingRules{ExplicitPath: b.kubeconfig})
	}

	// If the recommended kubeconfig env variable is not specified,
	// try the in-cluster config.
	kubeconfigPath := os.Getenv(clientcmd.RecommendedConfigPathEnvVar)
	if len(kubeconfigPath) == 0 {
		c, err := rest.InClusterConfig()
		if err == nil {
			return c, nil
		}

		defer func() {
			if configErr != nil {
				slog.Error("unable to load in-cluster config")
			}
		}()
	}

	// If the recommended kubeconfig env variable is set, or there
	// is no in-cluster config, try the default recommended locations.
	//
	// NOTE: For default config file locations, upstream only checks
	// $HOME for the user's home directory, but we can also try
	// os/user.HomeDir when $HOME is unset.
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if _, ok := os.LookupEnv("HOME"); !ok {
		u, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("could not get current user: %w", err)
		}
		loadingRules.Precedence = append(loadingRules.Precedence, filepath.Join(u.HomeDir, clientcmd.RecommendedHomeDir, clientcmd.RecommendedFileName))
	}
	return loadConfigWithContext(loadingRules)
}

func loadConfigWithContext(loader clientcmd.ClientConfigLoader) (*rest.Config, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, &clientcmd.ConfigOverrides{}).ClientConfig()
}
