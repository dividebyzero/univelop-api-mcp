package tools

import "strconv"

// headerInt parses an integer response header.
func headerInt(h map[string][]string, name string) int {
	v, ok := h[name]
	if !ok || len(v) == 0 {
		return 0
	}
	n, err := strconv.Atoi(v[0])
	if err != nil {
		return 0
	}
	return n
}