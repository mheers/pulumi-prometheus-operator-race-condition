package main

import (
	"github.com/aws/smithy-go/ptr"
	"github.com/mheers/pulumi-helper/helm"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	"github.com/pulumi/pulumi-kubernetes/provider/v4/pkg/provider"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	helmv3 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	pulumiV1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	pulumiYaml "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/yaml"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Prometheus struct {
	ctx                *pulumi.Context
	helmChart          *helmv3.Chart
	helmChartSrc       helm.HelmChartSrc
	namespace          *corev1.Namespace
	prometheusInstance *monitoringv1.Prometheus
}

func NewPrometheus(ctx *pulumi.Context) (*Prometheus, error) {
	pm := &Prometheus{
		ctx: ctx,
		helmChartSrc: helm.HelmChartSrc{
			HelmChartOpts: provider.HelmChartOpts{
				Chart: "kube-prometheus-stack",
				HelmFetchOpts: provider.HelmFetchOpts{
					Repo:    "https://prometheus-community.github.io/helm-charts",
					Version: "51.2.0",
				},
			},
		},
	}

	return pm, nil
}

func (pm *Prometheus) Install() error {
	if err := pm.installNamespace(); err != nil {
		return err
	}
	if err := pm.installPrometheusOperator(); err != nil {
		return err
	}
	if err := pm.installPrometheusInstance(); err != nil {
		return err
	}

	return nil
}

func (pm *Prometheus) installNamespace() error {
	namespaceName := "prometheus"

	namespace, err := corev1.NewNamespace(pm.ctx, namespaceName, &corev1.NamespaceArgs{
		Metadata: &pulumiV1.ObjectMetaArgs{
			Name: pulumi.String(namespaceName),
		},
	},
	)
	if err != nil {
		return err
	}

	pm.namespace = namespace

	return nil
}

func (pm *Prometheus) installPrometheusOperator() error {
	prometheus, err := helmv3.NewChart(pm.ctx, "prometheus", helmv3.ChartArgs{
		Chart:     pulumi.String(pm.helmChartSrc.Chart),
		Namespace: pulumi.String("prometheus"),
		FetchArgs: helmv3.FetchArgs{
			Repo: pulumi.String(pm.helmChartSrc.HelmFetchOpts.Repo),
		},
		// see https://artifacthub.io/packages/helm/prometheus-community/prometheus?modal=values for values
		Values: pulumi.Map{
			"nameOverride:": pulumi.String("prometheus-operator"),
			"coreDns": pulumi.Map{
				"enabled": pulumi.Bool(false),
			},
			"kubeEtcd": pulumi.Map{
				"enabled": pulumi.Bool(false),
			},
			"kubeControllerManager": pulumi.Map{
				"enabled": pulumi.Bool(false),
			},
			"kubeScheduler": pulumi.Map{
				"enabled": pulumi.Bool(false),
			},
			"kubeProxy": pulumi.Map{
				"enabled": pulumi.Bool(false),
			},
			"kubeApiServer": pulumi.Map{
				"tlsConfig": pulumi.Map{
					"insecureSkipVerify": pulumi.Bool(true),
				},
			},
			"alertmanager": pulumi.Map{
				"enabled": pulumi.Bool(false),
			},
			"grafana": pulumi.Map{
				"enabled": pulumi.Bool(false),
			},
			"prometheus": pulumi.Map{
				"enabled": pulumi.Bool(false),
			},
			// see https://artifacthub.io/packages/helm/prometheus-community/kube-state-metrics?modal=values for values
			"kube-state-metrics": pulumi.Map{
				//Resources from: https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-state-metrics
				"resources": pulumi.Map{
					"limits": pulumi.Map{
						"memory": pulumi.String("250Mi"),
					},
					"requests": pulumi.Map{
						"memory": pulumi.String("250Mi"),
						"cpu":    pulumi.String("100m"),
					},
				},
			},
			"nodeExporter": pulumi.Map{
				"enabled": pulumi.Bool(false),
			},
			"prometheus-node-exporter": pulumi.Map{
				"resources": pulumi.Map{
					"limits": pulumi.Map{
						"memory": pulumi.String("64Mi"),
					},
					"requests": pulumi.Map{
						"memory": pulumi.String("64Mi"),
						"cpu":    pulumi.String("10m"),
					},
				},
			},
			"prometheusOperator": pulumi.Map{
				"logLevel": pulumi.String("all"),
				"tls": pulumi.Map{
					"enabled": pulumi.Bool(false),
				},
				"admissionWebhooks": pulumi.Map{
					"enabled":       pulumi.Bool(false),
					"failurePolicy": pulumi.String("Ignore"),
					"patch": pulumi.Map{
						"enabled": pulumi.Bool(false),
					},
				},

				"prometheusConfigReloader": pulumi.Map{},
			},
		},
	},
		pulumi.Parent(pm.namespace),
	)
	if err != nil {
		return err
	}
	pm.helmChart = prometheus
	return nil
}

func (pm *Prometheus) installPrometheusInstance() error {

	prometheus := monitoringv1.Prometheus{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Prometheus",
			APIVersion: "monitoring.coreos.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prometheus",
			Namespace: "prometheus",
		},
		Spec: monitoringv1.PrometheusSpec{
			CommonPrometheusFields: monitoringv1.CommonPrometheusFields{
				Replicas: ptr.Int32(1),
				Storage: &monitoringv1.StorageSpec{
					VolumeClaimTemplate: monitoringv1.EmbeddedPersistentVolumeClaim{
						Spec: v1.PersistentVolumeClaimSpec{
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{
									v1.ResourceStorage: resource.MustParse("5Gi"),
								},
							},
						},
					},
				},
			},
		},
	}

	pm.prometheusInstance = &prometheus
	_, err := pm.createPulumiResource(pm.prometheusInstance)
	if err != nil {
		return err
	}

	return nil
}

func (pm *Prometheus) createPulumiResource(customResource interface{}) (pulumi.Resource, error) {
	scheduleYamlMap, err := GetYamlMap(customResource)
	if err != nil {
		return nil, err
	}

	objs := []map[string]interface{}{scheduleYamlMap}

	prometheus, err := pulumiYaml.ParseYamlObjects(
		pm.ctx,
		objs,
		nil,
		"prometheus",
		pulumi.Parent(pm.namespace),
	)
	if err != nil {
		return nil, err
	}

	// ugly, should be easier in go 1.22
	keys := make([]string, 1)
	for k := range prometheus {
		keys[0] = k
		break // we only need the first key
	}
	return prometheus[keys[0]], nil
}
