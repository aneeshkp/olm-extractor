package images_test

import (
	"testing"

	"github.com/operator-framework/api/pkg/manifests"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/lburgazzoli/olm-extractor/pkg/images"

	. "github.com/onsi/gomega"
)

func TestExtract(t *testing.T) {
	t.Run("extracts related images from env vars", func(t *testing.T) {
		g := NewWithT(t)

		bundle := createTestBundle(
			[]corev1.Container{
				{
					Name:  "controller",
					Image: "quay.io/operator/controller:v1.0.0",
					Env: []corev1.EnvVar{
						{Name: "RELATED_IMAGE_PROMETHEUS", Value: "quay.io/prometheus/prometheus:v2.40.0"},
						{Name: "RELATED_IMAGE_ALERTMANAGER", Value: "quay.io/prometheus/alertmanager:v0.25.0"},
						{Name: "OTHER_VAR", Value: "not-an-image"},
					},
				},
			},
			nil,
		)

		cfg := images.Config{
			EnvPattern:            images.DefaultEnvPattern,
			IncludeOperatorImages: false,
		}
		result, err := images.Extract(bundle, cfg)

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(result.RelatedImages()).To(ConsistOf(
			"quay.io/prometheus/alertmanager:v0.25.0",
			"quay.io/prometheus/prometheus:v2.40.0",
		))
		g.Expect(result.OperatorImages()).To(BeEmpty())
	})

	t.Run("includes operator images when requested", func(t *testing.T) {
		g := NewWithT(t)

		bundle := createTestBundle(
			[]corev1.Container{
				{
					Name:  "controller",
					Image: "quay.io/operator/controller:v1.0.0",
					Env: []corev1.EnvVar{
						{Name: "RELATED_IMAGE_COMPONENT", Value: "quay.io/example/component:v1.0.0"},
					},
				},
			},
			nil,
		)

		cfg := images.Config{
			EnvPattern:            images.DefaultEnvPattern,
			IncludeOperatorImages: true,
		}
		result, err := images.Extract(bundle, cfg)

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(result.OperatorImages()).To(ConsistOf("quay.io/operator/controller:v1.0.0"))
		g.Expect(result.RelatedImages()).To(ConsistOf("quay.io/example/component:v1.0.0"))
	})

	t.Run("extracts from init containers", func(t *testing.T) {
		g := NewWithT(t)

		bundle := createTestBundle(
			[]corev1.Container{
				{
					Name:  "controller",
					Image: "quay.io/operator/controller:v1.0.0",
				},
			},
			[]corev1.Container{
				{
					Name:  "init",
					Image: "quay.io/operator/init:v1.0.0",
					Env: []corev1.EnvVar{
						{Name: "RELATED_IMAGE_SETUP", Value: "quay.io/example/setup:v1.0.0"},
					},
				},
			},
		)

		cfg := images.Config{
			EnvPattern:            images.DefaultEnvPattern,
			IncludeOperatorImages: true,
		}
		result, err := images.Extract(bundle, cfg)

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(result.OperatorImages()).To(ConsistOf(
			"quay.io/operator/controller:v1.0.0",
			"quay.io/operator/init:v1.0.0",
		))
		g.Expect(result.RelatedImages()).To(ConsistOf("quay.io/example/setup:v1.0.0"))
	})

	t.Run("deduplicates images", func(t *testing.T) {
		g := NewWithT(t)

		bundle := createTestBundle(
			[]corev1.Container{
				{
					Name:  "controller1",
					Image: "quay.io/operator/controller:v1.0.0",
					Env: []corev1.EnvVar{
						{Name: "RELATED_IMAGE_A", Value: "quay.io/example/image:v1.0.0"},
					},
				},
				{
					Name:  "controller2",
					Image: "quay.io/operator/controller:v1.0.0", // duplicate
					Env: []corev1.EnvVar{
						{Name: "RELATED_IMAGE_B", Value: "quay.io/example/image:v1.0.0"}, // duplicate
					},
				},
			},
			nil,
		)

		cfg := images.Config{
			EnvPattern:            images.DefaultEnvPattern,
			IncludeOperatorImages: true,
		}
		result, err := images.Extract(bundle, cfg)

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(result.OperatorImages()).To(HaveLen(1))
		g.Expect(result.RelatedImages()).To(HaveLen(1))
	})

	t.Run("uses custom env pattern", func(t *testing.T) {
		g := NewWithT(t)

		bundle := createTestBundle(
			[]corev1.Container{
				{
					Name:  "controller",
					Image: "quay.io/operator/controller:v1.0.0",
					Env: []corev1.EnvVar{
						{Name: "RELATED_IMAGE_DEFAULT", Value: "quay.io/default/image:v1.0.0"},
						{Name: "MY_IMAGE_CUSTOM", Value: "quay.io/custom/image:v1.0.0"},
					},
				},
			},
			nil,
		)

		cfg := images.Config{
			EnvPattern:            "MY_IMAGE",
			IncludeOperatorImages: false,
		}
		result, err := images.Extract(bundle, cfg)

		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(result.RelatedImages()).To(ConsistOf("quay.io/custom/image:v1.0.0"))
	})

	t.Run("returns error for nil bundle", func(t *testing.T) {
		g := NewWithT(t)

		cfg := images.Config{EnvPattern: images.DefaultEnvPattern}
		_, err := images.Extract(nil, cfg)

		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("bundle is nil"))
	})

	t.Run("returns error for missing CSV", func(t *testing.T) {
		g := NewWithT(t)

		bundle := &manifests.Bundle{}
		cfg := images.Config{EnvPattern: images.DefaultEnvPattern}
		_, err := images.Extract(bundle, cfg)

		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("no ClusterServiceVersion"))
	})
}

func TestResult_AllImages(t *testing.T) {
	t.Run("returns combined deduplicated list", func(t *testing.T) {
		g := NewWithT(t)

		bundle := createTestBundle(
			[]corev1.Container{
				{
					Name:  "controller",
					Image: "quay.io/a:v1",
					Env: []corev1.EnvVar{
						{Name: "RELATED_IMAGE_B", Value: "quay.io/b:v1"},
						{Name: "RELATED_IMAGE_C", Value: "quay.io/c:v1"},
					},
				},
			},
			nil,
		)

		cfg := images.Config{
			EnvPattern:            images.DefaultEnvPattern,
			IncludeOperatorImages: true,
		}
		result, err := images.Extract(bundle, cfg)

		g.Expect(err).NotTo(HaveOccurred())

		all := result.AllImages()
		g.Expect(all).To(HaveLen(3))
		g.Expect(all).To(ContainElements("quay.io/a:v1", "quay.io/b:v1", "quay.io/c:v1"))
	})
}

// createTestBundle creates a minimal bundle for testing.
func createTestBundle(
	containers []corev1.Container,
	initContainers []corev1.Container,
) *manifests.Bundle {
	return &manifests.Bundle{
		CSV: &v1alpha1.ClusterServiceVersion{
			Spec: v1alpha1.ClusterServiceVersionSpec{
				InstallStrategy: v1alpha1.NamedInstallStrategy{
					StrategyName: "deployment",
					StrategySpec: v1alpha1.StrategyDetailsDeployment{
						DeploymentSpecs: []v1alpha1.StrategyDeploymentSpec{
							{
								Name: "controller-manager",
								Spec: appsv1.DeploymentSpec{
									Template: corev1.PodTemplateSpec{
										Spec: corev1.PodSpec{
											Containers:     containers,
											InitContainers: initContainers,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
