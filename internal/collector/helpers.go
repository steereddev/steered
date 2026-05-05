package collector

// isSystemNamespace returns true for kube system namespaces
func isSystemNamespace(ns string) bool {
	system := map[string]bool{
		"kube-system":     true,
		"kube-public":     true,
		"kube-node-lease": true,
	}
	return system[ns]
}
