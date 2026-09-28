package dependencies

// Reduce drops edges implied by a longer path, so `api -> go-shared ->
// nested` is not also written as `api -> nested`. The closure monotrack walks
// at change-detection time is unchanged; only the config gets smaller.
//
// A cyclic graph is returned untouched: reduction there can disconnect a node,
// and the cycle is rejected by config validation anyway.
func Reduce(graph map[string][]string) map[string][]string {
	if hasCycle(graph) {
		return graph
	}

	out := make(map[string][]string, len(graph))
	for u, deps := range graph {
		kept := make([]string, 0, len(deps))
		for _, v := range deps {
			redundant := false
			for _, w := range deps {
				if w == v {
					continue
				}
				if _, ok := reachable(graph, w)[v]; ok {
					redundant = true
					break
				}
			}
			if !redundant {
				kept = append(kept, v)
			}
		}
		out[u] = kept
	}
	return out
}

// reachable returns the nodes reachable from start, excluding start itself
// unless a cycle leads back to it.
func reachable(graph map[string][]string, start string) map[string]struct{} {
	out := make(map[string]struct{})
	stack := append([]string(nil), graph[start]...)
	for len(stack) > 0 {
		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if _, seen := out[n]; seen {
			continue
		}
		out[n] = struct{}{}
		stack = append(stack, graph[n]...)
	}
	return out
}

func hasCycle(graph map[string][]string) bool {
	for n := range graph {
		if _, ok := reachable(graph, n)[n]; ok {
			return true
		}
	}
	return false
}
