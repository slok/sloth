package k8scontroller

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	monitoringclientset "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned"
	corev1 "k8s.io/api/core/v1"
	kubeerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	slothclientset "github.com/slok/sloth/pkg/kubernetes/gen/clientset/versioned"
	"github.com/slok/sloth/test/integration/testutils"
)

type Config struct {
	Binary      string
	KubeConfig  string
	KubeContext string
}

func (c *Config) defaults() error {
	if c.Binary == "" {
		c.Binary = "sloth"
	}

	_, err := exec.LookPath(c.Binary)
	if err != nil {
		return fmt.Errorf("sloth binary missing in %q: %w", c.Binary, err)
	}

	if c.KubeConfig == "" {
		return fmt.Errorf("kubeconfig path is required")
	}

	return nil
}

// NewIntegrationConfig prepares the configuration for integration tests, if the configuration is not ready
// it will skip the test.
func NewConfig(t *testing.T) Config {
	const (
		envSlothBin         = "SLOTH_INTEGRATION_BINARY"
		envSlothKubeContext = "SLOTH_INTEGRATION_KUBE_CONTEXT"
		envSlothKubeConfig  = "SLOTH_INTEGRATION_KUBE_CONFIG"
	)

	c := Config{
		Binary:      os.Getenv(envSlothBin),
		KubeConfig:  os.Getenv(envSlothKubeConfig),
		KubeContext: os.Getenv(envSlothKubeContext),
	}

	err := c.defaults()
	if err != nil {
		t.Skipf("Skipping due to invalid config: %s", err)
	}

	return c
}

func RunSlothController(ctx context.Context, config Config, ns string, cmdArgs string) (stdout, stderr []byte, err error) {
	env := []string{
		fmt.Sprintf("SLOTH_KUBE_CONFIG=%s", config.KubeConfig),
		fmt.Sprintf("SLOTH_KUBE_CONTEXT=%s", config.KubeContext),
		fmt.Sprintf("SLOTH_KUBE_NAMESPACE=%s", ns),
		fmt.Sprintf("SLOTH_KUBE_LOCAL=%t", true),
	}

	return testutils.RunSloth(ctx, env, config.Binary, fmt.Sprintf("kubernetes-controller %s", cmdArgs), true)
}

type KubeClients struct {
	Std        kubernetes.Interface
	Sloth      slothclientset.Interface
	Monitoring monitoringclientset.Interface
}

// NewKubernetesClients returns Kubernetes clients.
func NewKubernetesClients(ctx context.Context, config Config) (*KubeClients, error) {
	kcfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{
			ExplicitPath: config.KubeConfig,
		},
		&clientcmd.ConfigOverrides{
			CurrentContext: config.KubeContext,
			Timeout:        "3s",
		},
	).ClientConfig()

	if err != nil {
		return nil, fmt.Errorf("could not load Kubernetes configuration: %w", err)
	}

	kcfg.Burst = 100
	kcfg.QPS = 100

	stdCli, err := kubernetes.NewForConfig(kcfg)
	if err != nil {
		return nil, fmt.Errorf("could not create Kubernetes client: %w", err)
	}

	slothcli, err := slothclientset.NewForConfig(kcfg)
	if err != nil {
		return nil, fmt.Errorf("could not create Kubernetes sloth client: %w", err)
	}

	monitoringCli, err := monitoringclientset.NewForConfig(kcfg)
	if err != nil {
		return nil, fmt.Errorf("could not create Kubernetes monitoring (prometheus-operator) client: %w", err)
	}

	return &KubeClients{
		Std:        stdCli,
		Sloth:      slothcli,
		Monitoring: monitoringCli,
	}, nil
}

func NewKubernetesNamespace(ctx context.Context, cli kubernetes.Interface) (nsName string, deleteNS func(ctx context.Context) error, err error) {
	// Create NS.
	nsName = fmt.Sprintf("sloth-test-%d", time.Now().UnixNano())
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: nsName}}
	_, err = cli.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		return "", nil, fmt.Errorf("could not create test namespace: %w", err)
	}

	// Generate the delete NS func.
	cancelFunc := func(ctx context.Context) error {
		err := cli.CoreV1().Namespaces().Delete(ctx, nsName, metav1.DeleteOptions{})
		if err != nil && !kubeerrors.IsNotFound(err) {
			return err
		}

		// Wait.
		ticker := time.NewTicker(200 * time.Millisecond)
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		for {
			select {
			case <-ticker.C:
			case <-ctx.Done():
				return fmt.Errorf("context cancelled while waiting for namespace cleanup")
			}

			// Check if deleted.
			_, err := cli.CoreV1().Namespaces().Get(ctx, nsName, metav1.GetOptions{})
			if err != nil && kubeerrors.IsNotFound(err) {
				break
			}
		}

		return nil
	}

	return nsName, cancelFunc, nil
}
