package formatter

import (
	"strings"
	"testing"

	"k8s-configmap-analyzer/pkg/analyzer"
)

func TestToMermaidConsolidated(t *testing.T) {
	result := &analyzer.AnalysisResult{
		Relationships: []analyzer.Relationship{
			{
				Deployment: "mock-worker-1",
				ConfigMaps: []string{"shared-cm"},
			},
			{
				Deployment: "mock-worker-2",
				ConfigMaps: []string{"shared-cm"},
			},
			{
				Deployment: "auth-serve",
				ConfigMaps: []string{"auth-cm"},
			},
		},
		ExternalDeps: map[string][]analyzer.ExternalDep{
			"shared-cm": {
				{Value: "1.2.3.4:5432", Key: "DB_HOST"},
			},
		},
	}

	output := ToMermaid(result)
	t.Logf("Generated output:\n%s", output)

	// Check for consolidated node with quotes
	if !strings.Contains(output, "[\"mock-worker (2)\"]") {
		t.Errorf("Expected output to contain consolidated node '[\"mock-worker (2)\"]'")
	}

	// Check for single deployment node with quotes
	if !strings.Contains(output, "d_auth_serve[\"auth-serve\"]") {
		t.Errorf("Expected output to contain single node 'd_auth_serve[\"auth-serve\"]'")
	}

	// Check for external dependency node with key context
	if !strings.Contains(output, "{\"1.2.3.4:5432 (DB_HOST)\"}") {
		t.Errorf("Expected output to contain external dependency label with key context")
	}

	// Check for subgraphs
	if !strings.Contains(output, "subgraph sg_mock [\"mock Services\"]") {
		t.Errorf("Expected output to contain mock subgraph")
	}
}
