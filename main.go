package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"k8s-configmap-analyzer/pkg/analyzer"
	"k8s-configmap-analyzer/pkg/formatter"
	"k8s-configmap-analyzer/pkg/k8s"
)

func main() {
	var namespace string
	flag.StringVar(&namespace, "namespace", "default", "Kubernetes namespace to analyze")

	clientset, err := k8s.NewClientset()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Kubernetes client: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Analyzing namespace: %s...\n", namespace)

	result, err := analyzer.AnalyzeDeployments(clientset, namespace)
	if err != nil {
		log.Fatalf("Error analyzing deployments: %v", err)
	}

	if len(result.Relationships) == 0 {
		fmt.Println("No Deployment-to-ConfigMap relationships found.")
		return
	}

	mermaidOutput := formatter.ToMermaid(result)
	fmt.Println(mermaidOutput)
}
