// Package images provides functionality to extract related images from OLM bundles.
package images

import (
	"errors"
	"fmt"
	"strings"

	"github.com/operator-framework/api/pkg/manifests"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	// DefaultEnvPattern is the default environment variable prefix for related images.
	DefaultEnvPattern = "RELATED_IMAGE"

	// deploymentStrategy is the expected install strategy name.
	deploymentStrategy = "deployment"
)

// Config contains options for image extraction.
type Config struct {
	EnvPattern            string `mapstructure:"env-pattern"`
	IncludeOperatorImages bool   `mapstructure:"include-operator-images"`
}

// Result contains the extracted images from a bundle.
type Result struct {
	operatorImages sets.Set[string]
	relatedImages  sets.Set[string]
}

// newResult creates a new Result with initialized sets.
func newResult() *Result {
	return &Result{
		operatorImages: sets.New[string](),
		relatedImages:  sets.New[string](),
	}
}

// OperatorImages returns operator images as a sorted slice.
func (r *Result) OperatorImages() []string {
	return sets.List(r.operatorImages)
}

// RelatedImages returns related images as a sorted slice.
func (r *Result) RelatedImages() []string {
	return sets.List(r.relatedImages)
}

// AllImages returns all images (operator + related) as a single deduplicated sorted slice.
func (r *Result) AllImages() []string {
	return sets.List(r.operatorImages.Union(r.relatedImages))
}

// Extract extracts related images from an OLM bundle.
func Extract(b *manifests.Bundle, cfg Config) (*Result, error) {
	if b == nil {
		return nil, errors.New("bundle is nil")
	}

	csv := b.CSV
	if csv == nil {
		return nil, errors.New("bundle has no ClusterServiceVersion")
	}

	strategy := csv.Spec.InstallStrategy
	if strategy.StrategyName != deploymentStrategy {
		return nil, fmt.Errorf("unsupported install strategy: %s", strategy.StrategyName)
	}

	result := newResult()

	// Iterate over all deployment specs in the install strategy
	for _, depSpec := range strategy.StrategySpec.DeploymentSpecs {
		result.extractFromContainers(depSpec.Spec.Template.Spec.Containers, cfg)
		result.extractFromContainers(depSpec.Spec.Template.Spec.InitContainers, cfg)
	}

	return result, nil
}

// extractFromContainers extracts images from a slice of containers.
func (r *Result) extractFromContainers(containers []corev1.Container, cfg Config) {
	for _, container := range containers {
		// Collect operator container images if requested
		if cfg.IncludeOperatorImages && container.Image != "" {
			r.operatorImages.Insert(container.Image)
		}

		// Collect related images from environment variables
		for _, env := range container.Env {
			if strings.HasPrefix(env.Name, cfg.EnvPattern) && env.Value != "" {
				r.relatedImages.Insert(env.Value)
			}
		}
	}
}
