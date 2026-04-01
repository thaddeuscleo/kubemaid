package analyzer

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestExtractConfigMaps(t *testing.T) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "mock-dep"},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "mock-container",
							Env: []corev1.EnvVar{
								{
									Name: "MOCK_ENV_VAR",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{Name: "mock-cm-1"},
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

	configMapSet := make(map[string]struct{})
	for _, container := range deployment.Spec.Template.Spec.Containers {
		for _, env := range container.Env {
			if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil {
				configMapSet[env.ValueFrom.ConfigMapKeyRef.Name] = struct{}{}
			}
		}
	}

	if _, ok := configMapSet["mock-cm-1"]; !ok {
		t.Errorf("Expected mock-cm-1 to be identified")
	}
}

func TestScanExternalDepsWithKey(t *testing.T) {
	cmData := map[string]string{
		"DB_HOST":                 "1.2.3.4:5432",
		"ELASTICSEARCH_ADDRESSES": "https://service.example.com:9243",
		"OTEL_COLLECTOR_ENDPOINT": "dns:///mock-agent.mock-ns.svc.cluster.local:4317",
	}

	depMap := make(map[string]string)
	for key, val := range cmData {
		matches := schemeRegex.FindAllString(val, -1)
		for _, m := range matches {
			if _, ok := depMap[m]; !ok { depMap[m] = key }
		}
		matches = ipPortRegex.FindAllString(val, -1)
		for _, m := range matches {
			if _, ok := depMap[m]; !ok { depMap[m] = key }
		}
		matches = genericAddrRegex.FindAllString(val, -1)
		for _, m := range matches {
			if _, ok := depMap[m]; !ok { depMap[m] = key }
		}
	}

	expected := map[string]string{
		"1.2.3.4:5432": "DB_HOST",
		"https://service.example.com:9243": "ELASTICSEARCH_ADDRESSES",
		"dns:///mock-agent.mock-ns.svc.cluster.local:4317": "OTEL_COLLECTOR_ENDPOINT",
	}

	for val, expKey := range expected {
		if key, ok := depMap[val]; !ok || key != expKey {
			t.Errorf("Expected external dependency %s with key %s, got key %s", val, expKey, key)
		}
	}
}
