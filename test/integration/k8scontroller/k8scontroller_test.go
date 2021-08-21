package k8scontroller_test

import (
	"context"
	"testing"
	"time"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	slothv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	"github.com/slok/sloth/test/integration/k8scontroller"
	"github.com/slok/sloth/test/integration/testutils"
)

// sanitizePrometheusRule will remove all the dynamic fields on a monitoringv1.PrometheusRule object
// these fileds are normally set by Kubernetes.
func sanitizePrometheusRule(pr *monitoringv1.PrometheusRule) *monitoringv1.PrometheusRule {
	pr = pr.DeepCopy()

	pr.ManagedFields = nil
	pr.UID = ""
	pr.ResourceVersion = ""
	pr.Generation = 0
	pr.CreationTimestamp = metav1.Time{}
	pr.SelfLink = ""

	for i := range pr.OwnerReferences {
		pr.OwnerReferences[i].UID = ""
	}

	return pr
}

func TestKubernetesControllerPromOperatorGenerate(t *testing.T) {
	// Tests config.
	config := k8scontroller.NewConfig(t)
	version, err := testutils.SlothVersion(context.TODO(), config.Binary)
	require.NoError(t, err)

	// KubeClis.
	kubeClis, err := k8scontroller.NewKubernetesClients(context.TODO(), config)
	require.NoError(t, err)

	// Tests.
	tests := map[string]struct {
		exec func(ctx context.Context, t *testing.T, ns string, kubeClis *k8scontroller.KubeClients)
	}{
		"Having SLOs as a CRD should generate Prometheus operator CRD.": {
			exec: func(ctx context.Context, t *testing.T, ns string, kubeClis *k8scontroller.KubeClients) {
				// Prepare our SLO on Kubernetes.
				SLOs := getBasePrometheusServiceLevel()
				_, err = kubeClis.Sloth.SlothV1().PrometheusServiceLevels(ns).Create(ctx, SLOs, metav1.CreateOptions{})
				require.NoError(t, err)

				// Wait to be sure the controller had time for handling.
				time.Sleep(250 * time.Millisecond)

				// Check.
				expRule := getBasePromOpPrometheusRule(version)
				expRule.Namespace = ns

				gotRule, err := kubeClis.Monitoring.MonitoringV1().PrometheusRules(ns).Get(ctx, expRule.Name, metav1.GetOptions{})
				gotRule = sanitizePrometheusRule(gotRule) // Remove variations.
				require.NoError(t, err)

				assert.Equal(t, expRule, gotRule)
			},
		},

		"Having SLOs with plugins as a CRD should generate Prometheus operator CRD.": {
			exec: func(ctx context.Context, t *testing.T, ns string, kubeClis *k8scontroller.KubeClients) {
				// Prepare our SLO on Kubernetes with plugin based SLO.
				SLOs := getPluginPrometheusServiceLevel()
				_, err = kubeClis.Sloth.SlothV1().PrometheusServiceLevels(ns).Create(ctx, SLOs, metav1.CreateOptions{})
				require.NoError(t, err)

				// Wait to be sure the controller had time for handling.
				time.Sleep(250 * time.Millisecond)

				// Check.
				expRule := getPluginPromOpPrometheusRule(version)
				expRule.Namespace = ns

				gotRule, err := kubeClis.Monitoring.MonitoringV1().PrometheusRules(ns).Get(ctx, expRule.Name, metav1.GetOptions{})
				gotRule = sanitizePrometheusRule(gotRule) // Remove variations.
				require.NoError(t, err)

				assert.Equal(t, expRule, gotRule)
			},
		},

		"Having SLOs as a CRD should set the status as correct on the CRD.": {
			exec: func(ctx context.Context, t *testing.T, ns string, kubeClis *k8scontroller.KubeClients) {
				// Prepare our SLO on Kubernetes.
				SLOs := getBasePrometheusServiceLevel()
				newSLOs, err := kubeClis.Sloth.SlothV1().PrometheusServiceLevels(ns).Create(ctx, SLOs, metav1.CreateOptions{})
				require.NoError(t, err)

				// Wait to be sure the controller had time for handling.
				time.Sleep(250 * time.Millisecond)

				// Check.
				gotSLOs, err := kubeClis.Sloth.SlothV1().PrometheusServiceLevels(ns).Get(ctx, SLOs.Name, metav1.GetOptions{})
				require.NoError(t, err)

				expStatus := slothv1.PrometheusServiceLevelStatus{
					ProcessedSLOs:            2,
					PromOpRulesGeneratedSLOs: 2,
					PromOpRulesGenerated:     true,
					ObservedGeneration:       newSLOs.Generation,
				}
				gotSLOs.Status.LastPromOpRulesSuccessfulGenerated = nil // Remove variations.

				assert.Equal(t, expStatus, gotSLOs.Status)
			},
		},

		"Having wrong SLOs as a CRD should set the status failed on the CRD.": {
			exec: func(ctx context.Context, t *testing.T, ns string, kubeClis *k8scontroller.KubeClients) {
				// Prepare our wrong SLO on Kubernetes.
				SLOs := getBasePrometheusServiceLevel()
				SLOs.Spec.SLOs[0].Objective = 101 // Make the SLO invalid.
				newSLOs, err := kubeClis.Sloth.SlothV1().PrometheusServiceLevels(ns).Create(ctx, SLOs, metav1.CreateOptions{})
				require.NoError(t, err)

				// Wait to be sure the controller had time for handling.
				time.Sleep(250 * time.Millisecond)

				// Check.
				gotSLOs, err := kubeClis.Sloth.SlothV1().PrometheusServiceLevels(ns).Get(ctx, SLOs.Name, metav1.GetOptions{})
				require.NoError(t, err)

				expStatus := slothv1.PrometheusServiceLevelStatus{
					ProcessedSLOs:            2,
					PromOpRulesGeneratedSLOs: 0,
					PromOpRulesGenerated:     false,
					ObservedGeneration:       newSLOs.Generation,
				}
				gotSLOs.Status.LastPromOpRulesSuccessfulGenerated = nil // Remove variations.

				assert.Equal(t, expStatus, gotSLOs.Status)
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			// Create a context with cancel so we can stop everything at the end of the test.
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Create NS and delete on test end.
			ns, deleteNS, err := k8scontroller.NewKubernetesNamespace(ctx, kubeClis.Std)
			require.NoError(err)
			defer func() {
				err := deleteNS(ctx)
				require.NoError(err)
			}()

			// Run controller in background.
			go func() {
				_, _, _ = k8scontroller.RunSlothController(ctx, config, ns, "")
			}()

			// Execute test.
			test.exec(ctx, t, ns, kubeClis)
		})
	}
}
