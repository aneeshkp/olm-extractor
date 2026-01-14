package extract_test

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/lburgazzoli/olm-extractor/pkg/certmanager"
	"github.com/lburgazzoli/olm-extractor/pkg/extract"

	. "github.com/onsi/gomega"
)

func TestTransformWatchNamespace(t *testing.T) {
	t.Run("transforms WATCH_NAMESPACE with olm.targetNamespaces fieldRef", func(t *testing.T) {
		g := NewWithT(t)

		deployment := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]any{
					"name": "test-operator",
				},
				"spec": map[string]any{
					"template": map[string]any{
						"spec": map[string]any{
							"containers": []any{
								map[string]any{
									"name": "manager",
									"env": []any{
										map[string]any{
											"name": "WATCH_NAMESPACE",
											"valueFrom": map[string]any{
												"fieldRef": map[string]any{
													"fieldPath": "metadata.annotations['olm.targetNamespaces']",
												},
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

		// Call the unexported function via the public ApplyTransformations
		result, err := extract.ApplyTransformations(
			[]*unstructured.Unstructured{deployment},
			"test-namespace",
			"my-watch-namespace",
			nil,
			nil,
			certmanager.Config{Enabled: false},
		)

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(1))

		// Verify the transformation
		containers, found, err := unstructured.NestedSlice(result[0].Object, "spec", "template", "spec", "containers")
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(found).To(BeTrue())
		g.Expect(containers).To(HaveLen(1))

		container := containers[0].(map[string]any)
		env, found, err := unstructured.NestedSlice(container, "env")
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(found).To(BeTrue())

		envVar := env[0].(map[string]any)
		g.Expect(envVar["name"]).To(Equal("WATCH_NAMESPACE"))
		g.Expect(envVar["value"]).To(Equal("my-watch-namespace"))
		g.Expect(envVar).ToNot(HaveKey("valueFrom"))
	})

	t.Run("overwrites WATCH_NAMESPACE with static value", func(t *testing.T) {
		g := NewWithT(t)

		deployment := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]any{
					"name": "test-operator",
				},
				"spec": map[string]any{
					"template": map[string]any{
						"spec": map[string]any{
							"containers": []any{
								map[string]any{
									"name": "manager",
									"env": []any{
										map[string]any{
											"name":  "WATCH_NAMESPACE",
											"value": "existing-value",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		result, err := extract.ApplyTransformations(
			[]*unstructured.Unstructured{deployment},
			"test-namespace",
			"my-watch-namespace",
			nil,
			nil,
			certmanager.Config{Enabled: false},
		)

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(1))

		// Verify the static value is overwritten with the new value
		containers, _, _ := unstructured.NestedSlice(result[0].Object, "spec", "template", "spec", "containers")
		container := containers[0].(map[string]any)
		env, _, _ := unstructured.NestedSlice(container, "env")
		envVar := env[0].(map[string]any)

		g.Expect(envVar["value"]).To(Equal("my-watch-namespace"))
	})

	t.Run("transforms WATCH_NAMESPACE in initContainers", func(t *testing.T) {
		g := NewWithT(t)

		deployment := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]any{
					"name": "test-operator",
				},
				"spec": map[string]any{
					"template": map[string]any{
						"spec": map[string]any{
							"initContainers": []any{
								map[string]any{
									"name": "init",
									"env": []any{
										map[string]any{
											"name": "WATCH_NAMESPACE",
											"valueFrom": map[string]any{
												"fieldRef": map[string]any{
													"fieldPath": "metadata.annotations['olm.targetNamespaces']",
												},
											},
										},
									},
								},
							},
							"containers": []any{
								map[string]any{
									"name": "manager",
								},
							},
						},
					},
				},
			},
		}

		result, err := extract.ApplyTransformations(
			[]*unstructured.Unstructured{deployment},
			"test-namespace",
			"init-watch-namespace",
			nil,
			nil,
			certmanager.Config{Enabled: false},
		)

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(1))

		// Verify the transformation in initContainers
		initContainers, found, err := unstructured.NestedSlice(result[0].Object, "spec", "template", "spec", "initContainers")
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(found).To(BeTrue())

		container := initContainers[0].(map[string]any)
		env, _, _ := unstructured.NestedSlice(container, "env")
		envVar := env[0].(map[string]any)

		g.Expect(envVar["name"]).To(Equal("WATCH_NAMESPACE"))
		g.Expect(envVar["value"]).To(Equal("init-watch-namespace"))
		g.Expect(envVar).ToNot(HaveKey("valueFrom"))
	})

	t.Run("preserves other fieldRef patterns", func(t *testing.T) {
		g := NewWithT(t)

		deployment := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]any{
					"name": "test-operator",
				},
				"spec": map[string]any{
					"template": map[string]any{
						"spec": map[string]any{
							"containers": []any{
								map[string]any{
									"name": "manager",
									"env": []any{
										map[string]any{
											"name": "POD_NAME",
											"valueFrom": map[string]any{
												"fieldRef": map[string]any{
													"fieldPath": "metadata.name",
												},
											},
										},
										map[string]any{
											"name": "POD_NAMESPACE",
											"valueFrom": map[string]any{
												"fieldRef": map[string]any{
													"fieldPath": "metadata.namespace",
												},
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

		result, err := extract.ApplyTransformations(
			[]*unstructured.Unstructured{deployment},
			"test-namespace",
			"my-watch-namespace",
			nil,
			nil,
			certmanager.Config{Enabled: false},
		)

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(1))

		// Verify other fieldRef patterns are preserved
		containers, _, _ := unstructured.NestedSlice(result[0].Object, "spec", "template", "spec", "containers")
		container := containers[0].(map[string]any)
		env, _, _ := unstructured.NestedSlice(container, "env")

		podNameEnv := env[0].(map[string]any)
		g.Expect(podNameEnv["name"]).To(Equal("POD_NAME"))
		g.Expect(podNameEnv).To(HaveKey("valueFrom"))

		podNamespaceEnv := env[1].(map[string]any)
		g.Expect(podNamespaceEnv["name"]).To(Equal("POD_NAMESPACE"))
		g.Expect(podNamespaceEnv).To(HaveKey("valueFrom"))
	})

	t.Run("sets value when WATCH_NAMESPACE has no value field", func(t *testing.T) {
		g := NewWithT(t)

		deployment := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]any{
					"name": "test-operator",
				},
				"spec": map[string]any{
					"template": map[string]any{
						"spec": map[string]any{
							"containers": []any{
								map[string]any{
									"name": "manager",
									"env": []any{
										map[string]any{
											"name": "WATCH_NAMESPACE",
											// No value or valueFrom field at all
										},
									},
								},
							},
						},
					},
				},
			},
		}

		result, err := extract.ApplyTransformations(
			[]*unstructured.Unstructured{deployment},
			"test-namespace",
			"", // Empty watch namespace
			nil,
			nil,
			certmanager.Config{Enabled: false},
		)

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(1))

		// Verify empty string is explicitly set
		containers, _, _ := unstructured.NestedSlice(result[0].Object, "spec", "template", "spec", "containers")
		container := containers[0].(map[string]any)
		env, _, _ := unstructured.NestedSlice(container, "env")
		envVar := env[0].(map[string]any)

		g.Expect(envVar["name"]).To(Equal("WATCH_NAMESPACE"))
		g.Expect(envVar["value"]).To(Equal(""))
		g.Expect(envVar).ToNot(HaveKey("valueFrom"))
	})

	t.Run("handles empty watch namespace", func(t *testing.T) {
		g := NewWithT(t)

		deployment := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]any{
					"name": "test-operator",
				},
				"spec": map[string]any{
					"template": map[string]any{
						"spec": map[string]any{
							"containers": []any{
								map[string]any{
									"name": "manager",
									"env": []any{
										map[string]any{
											"name": "WATCH_NAMESPACE",
											"valueFrom": map[string]any{
												"fieldRef": map[string]any{
													"fieldPath": "metadata.annotations['olm.targetNamespaces']",
												},
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

		result, err := extract.ApplyTransformations(
			[]*unstructured.Unstructured{deployment},
			"test-namespace",
			"", // Empty watch namespace for cluster-wide
			nil,
			nil,
			certmanager.Config{Enabled: false},
		)

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(1))

		// Verify empty string is set
		containers, _, _ := unstructured.NestedSlice(result[0].Object, "spec", "template", "spec", "containers")
		container := containers[0].(map[string]any)
		env, _, _ := unstructured.NestedSlice(container, "env")
		envVar := env[0].(map[string]any)

		g.Expect(envVar["value"]).To(Equal(""))
		g.Expect(envVar).ToNot(HaveKey("valueFrom"))
	})

	t.Run("does not modify non-deployment resources", func(t *testing.T) {
		g := NewWithT(t)

		service := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "v1",
				"kind":       "Service",
				"metadata": map[string]any{
					"name": "test-service",
				},
			},
		}

		result, err := extract.ApplyTransformations(
			[]*unstructured.Unstructured{service},
			"test-namespace",
			"my-watch-namespace",
			nil,
			nil,
			certmanager.Config{Enabled: false},
		)

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(1))
		g.Expect(result[0].GetKind()).To(Equal("Service"))
	})

	t.Run("handles multiple containers", func(t *testing.T) {
		g := NewWithT(t)

		deployment := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"metadata": map[string]any{
					"name": "test-operator",
				},
				"spec": map[string]any{
					"template": map[string]any{
						"spec": map[string]any{
							"containers": []any{
								map[string]any{
									"name": "manager",
									"env": []any{
										map[string]any{
											"name": "WATCH_NAMESPACE",
											"valueFrom": map[string]any{
												"fieldRef": map[string]any{
													"fieldPath": "metadata.annotations['olm.targetNamespaces']",
												},
											},
										},
									},
								},
								map[string]any{
									"name": "sidecar",
									"env": []any{
										map[string]any{
											"name": "WATCH_NAMESPACE",
											"valueFrom": map[string]any{
												"fieldRef": map[string]any{
													"fieldPath": "metadata.annotations['olm.targetNamespaces']",
												},
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

		result, err := extract.ApplyTransformations(
			[]*unstructured.Unstructured{deployment},
			"test-namespace",
			"multi-watch-namespace",
			nil,
			nil,
			certmanager.Config{Enabled: false},
		)

		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(result).To(HaveLen(1))

		// Verify both containers are transformed
		containers, _, _ := unstructured.NestedSlice(result[0].Object, "spec", "template", "spec", "containers")
		g.Expect(containers).To(HaveLen(2))

		for i := range 2 {
			container := containers[i].(map[string]any)
			env, _, _ := unstructured.NestedSlice(container, "env")
			envVar := env[0].(map[string]any)

			g.Expect(envVar["name"]).To(Equal("WATCH_NAMESPACE"))
			g.Expect(envVar["value"]).To(Equal("multi-watch-namespace"))
			g.Expect(envVar).ToNot(HaveKey("valueFrom"))
		}
	})
}
