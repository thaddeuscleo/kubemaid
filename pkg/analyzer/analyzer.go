package analyzer

import (
	"context"
	"regexp"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Relationship represents a Deployment to ConfigMap and ConfigMap to External connection.
type Relationship struct {
	Deployment string
	ConfigMaps []string
}

// ExternalDep represents an external target with context from the ConfigMap key.
type ExternalDep struct {
	Value string
	Key   string // e.g., DB_HOST
}

// AnalysisResult holds the full picture of the cluster's connections.
type AnalysisResult struct {
	Relationships []Relationship
	ExternalDeps  map[string][]ExternalDep // ConfigMap name -> list of external targets
}

var (
	// Regex patterns for external dependencies
	schemeRegex = regexp.MustCompile(`[a-zA-Z0-9\+]+://[a-zA-Z0-9\.\-_/]+(:[0-9]+)?`)
	ipPortRegex = regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}:[0-9]{2,5}\b`)
	genericAddrRegex = regexp.MustCompile(`\b[a-zA-Z0-9\.\-_]+:[0-9]{2,5}\b`)
)

// AnalyzeDeployments lists deployments and extracts ConfigMap references.
func AnalyzeDeployments(clientset *kubernetes.Clientset, namespace string) (*AnalysisResult, error) {
	// List deployments
	deployments, err := clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// List all ConfigMaps to scan their content
	configMaps, err := clientset.CoreV1().ConfigMaps(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	cmDataMap := make(map[string]map[string]string)
	for _, cm := range configMaps.Items {
		cmDataMap[cm.Name] = cm.Data
	}

	var relationships []Relationship
	referencedCMs := make(map[string]struct{})

	for _, d := range deployments.Items {
		configMapSet := make(map[string]struct{})

		// Analyze containers
		for _, container := range d.Spec.Template.Spec.Containers {
			// Check env
			for _, env := range container.Env {
				if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil {
					configMapSet[env.ValueFrom.ConfigMapKeyRef.Name] = struct{}{}
				}
			}

			// Check envFrom
			for _, envFrom := range container.EnvFrom {
				if envFrom.ConfigMapRef != nil {
					configMapSet[envFrom.ConfigMapRef.Name] = struct{}{}
				}
			}
		}

		// Analyze volumes
		for _, volume := range d.Spec.Template.Spec.Volumes {
			if volume.ConfigMap != nil {
				configMapSet[volume.ConfigMap.Name] = struct{}{}
			}
			if volume.Projected != nil {
				for _, source := range volume.Projected.Sources {
					if source.ConfigMap != nil {
						configMapSet[source.ConfigMap.Name] = struct{}{}
					}
				}
			}
		}

		if len(configMapSet) > 0 {
			var cms []string
			for cm := range configMapSet {
				cms = append(cms, cm)
				referencedCMs[cm] = struct{}{}
			}
			relationships = append(relationships, Relationship{
				Deployment: d.Name,
				ConfigMaps: cms,
			})
		}
	}

	// Scan referenced ConfigMaps for external dependencies
	externalDeps := make(map[string][]ExternalDep)
	for cmName := range referencedCMs {
		if data, ok := cmDataMap[cmName]; ok {
			depMap := make(map[string]string) // value -> first found key
			for key, val := range data {
				// Find all schemes
				matches := schemeRegex.FindAllString(val, -1)
				for _, m := range matches {
					if _, ok := depMap[m]; !ok { depMap[m] = key }
				}
				// Find IP:Ports
				matches = ipPortRegex.FindAllString(val, -1)
				for _, m := range matches {
					if _, ok := depMap[m]; !ok { depMap[m] = key }
				}
				// Find Host:Ports
				matches = genericAddrRegex.FindAllString(val, -1)
				for _, m := range matches {
					if _, ok := depMap[m]; !ok { depMap[m] = key }
				}
			}
			if len(depMap) > 0 {
				var deps []ExternalDep
				for val, key := range depMap {
					deps = append(deps, ExternalDep{Value: val, Key: key})
				}
				externalDeps[cmName] = deps
			}
		}
	}

	return &AnalysisResult{
		Relationships: relationships,
		ExternalDeps:  externalDeps,
	}, nil
}
