package formatter

import (
	"fmt"
	"sort"
	"strings"

	"k8s-configmap-analyzer/pkg/analyzer"
)

type groupKey struct {
	prefix string
	cms    string
}

// ToMermaid generates a consolidated Mermaid graph string with context in labels.
func ToMermaid(result *analyzer.AnalysisResult) string {
	var sb strings.Builder

	sb.WriteString("%% NOTE: If diagram is too large, increase maxEdges in your viewer's settings.\n")
	sb.WriteString("%% For example, call: mermaid.initialize({ 'maxEdges': 1000 });\n")
	sb.WriteString("graph TD\n")

	// Style Definitions
	sb.WriteString("    classDef deployment fill:#f9f,stroke:#333,stroke-width:2px;\n")
	sb.WriteString("    classDef configmap fill:#ccf,stroke:#333,stroke-width:2px;\n")
	sb.WriteString("    classDef external fill:#cfc,stroke:#333,stroke-width:2px;\n\n")

	// 1. Group by prefix and configmaps
	groups := make(map[groupKey][]string)
	for _, rel := range result.Relationships {
		prefix := "other"
		parts := strings.Split(rel.Deployment, "-")
		if len(parts) > 0 && parts[0] != "" {
			prefix = parts[0]
		}

		cms := make([]string, len(rel.ConfigMaps))
		copy(cms, rel.ConfigMaps)
		sort.Strings(cms)
		key := groupKey{prefix: prefix, cms: strings.Join(cms, ",")}
		groups[key] = append(groups[key], rel.Deployment)
	}

	renderedCMs := make(map[string]struct{})
	renderedExts := make(map[string]struct{})

	// 2. Render Subgraphs with Consolidated Nodes
	prefixes := make(map[string]struct{})
	for k := range groups {
		prefixes[k.prefix] = struct{}{}
	}
	var sortedPrefixes []string
	for p := range prefixes {
		sortedPrefixes = append(sortedPrefixes, p)
	}
	sort.Strings(sortedPrefixes)

	nodeToCMs := make(map[string][]string)

	for _, p := range sortedPrefixes {
		sb.WriteString(fmt.Sprintf("    subgraph sg_%s [\"%s Services\"]\n", p, p))
		sb.WriteString("        direction LR\n")
		
		var keysInPrefix []groupKey
		for k := range groups {
			if k.prefix == p {
				keysInPrefix = append(keysInPrefix, k)
			}
		}
		sort.Slice(keysInPrefix, func(i, j int) bool { return keysInPrefix[i].cms < keysInPrefix[j].cms })

		for _, k := range keysInPrefix {
			deps := groups[k]
			var nodeName, displayName string
			if len(deps) == 1 {
				nodeName = deps[0]
				displayName = deps[0]
			} else {
				common := commonPrefix(deps)
				if common == "" || common == p+"-" {
					common = p + "-cluster"
				} else {
					common = strings.TrimSuffix(common, "-")
				}
				nodeName = fmt.Sprintf("%s_cluster_%d", common, len(deps))
				displayName = fmt.Sprintf("%s (%d)", common, len(deps))
			}
			
			nodeID := sanitizeID("d_" + nodeName)
			sb.WriteString(fmt.Sprintf("        %s[\"%s\"]\n", nodeID, displayName))
			sb.WriteString(fmt.Sprintf("        class %s deployment\n", nodeID))
			
			if k.cms != "" {
				nodeToCMs[nodeID] = strings.Split(k.cms, ",")
			}
		}
		sb.WriteString("    end\n\n")
	}

	// 3. Render ConfigMap Subgraph
	sb.WriteString("    subgraph sg_configmaps [\"Config Maps\"]\n")
	sb.WriteString("        direction TB\n")
	for _, cms := range nodeToCMs {
		for _, cm := range cms {
			cmID := sanitizeID("cm_" + cm)
			if _, ok := renderedCMs[cmID]; !ok {
				sb.WriteString(fmt.Sprintf("        %s((\"%s\"))\n", cmID, cm))
				sb.WriteString(fmt.Sprintf("        class %s configmap\n", cmID))
				renderedCMs[cmID] = struct{}{}
			}
		}
	}
	sb.WriteString("    end\n\n")

	// 4. Render External Services Subgraph
	if len(result.ExternalDeps) > 0 {
		sb.WriteString("    subgraph sg_external [\"External Services\"]\n")
		sb.WriteString("        direction TB\n")
		for _, deps := range result.ExternalDeps {
			for _, dep := range deps {
				extID := sanitizeID("ext_" + dep.Value)
				if _, ok := renderedExts[extID]; !ok {
					// Add key context to the label
					displayName := fmt.Sprintf("%s (%s)", dep.Value, dep.Key)
					sb.WriteString(fmt.Sprintf("        %s{\"%s\"}\n", extID, displayName))
					sb.WriteString(fmt.Sprintf("        class %s external\n", extID))
					renderedExts[extID] = struct{}{}
				}
			}
		}
		sb.WriteString("    end\n\n")
	}

	// 5. Draw Edges
	for nodeID, cms := range nodeToCMs {
		for _, cm := range cms {
			cmID := sanitizeID("cm_" + cm)
			sb.WriteString(fmt.Sprintf("    %s --> %s\n", nodeID, cmID))
		}
	}

	for cmName, deps := range result.ExternalDeps {
		cmID := sanitizeID("cm_" + cmName)
		for _, dep := range deps {
			extID := sanitizeID("ext_" + dep.Value)
			sb.WriteString(fmt.Sprintf("    %s --> %s\n", cmID, extID))
		}
	}

	return sb.String()
}

func commonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	prefix := strs[0]
	for _, s := range strs[1:] {
		for i := 0; i < len(prefix); i++ {
			if i >= len(s) || prefix[i] != s[i] {
				prefix = prefix[:i]
				break
			}
		}
	}
	return prefix
}

func sanitizeID(id string) string {
	r := strings.NewReplacer(
		"-", "_",
		".", "_",
		":", "_",
		"/", "_",
		" ", "_",
		"@", "_",
		"(", "_",
		")", "_",
		"[", "_",
		"]", "_",
		"=", "_",
		"+", "_",
		",", "_",
		"&", "_",
		"$", "_",
		"!", "_",
		"?", "_",
	)
	return r.Replace(id)
}
