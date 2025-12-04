package ui

import "fmt"

// getMapValue safely gets a value from a map
func getMapValue(m map[string]interface{}, key string) (interface{}, bool) {
	if m == nil {
		return nil, false
	}
	val, ok := m[key]
	return val, ok
}

// valuesEqual compares two values for equality
func valuesEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Compare maps recursively
	aMap, aIsMap := a.(map[string]interface{})
	bMap, bIsMap := b.(map[string]interface{})
	if aIsMap && bIsMap {
		if len(aMap) != len(bMap) {
			return false
		}
		for k, av := range aMap {
			bv, ok := bMap[k]
			if !ok || !valuesEqual(av, bv) {
				return false
			}
		}
		return true
	}

	// Compare slices
	aSlice, aIsSlice := a.([]interface{})
	bSlice, bIsSlice := b.([]interface{})
	if aIsSlice && bIsSlice {
		if len(aSlice) != len(bSlice) {
			return false
		}
		for i := range aSlice {
			if !valuesEqual(aSlice[i], bSlice[i]) {
				return false
			}
		}
		return true
	}

	// Use fmt for simple comparison
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// sortStrings sorts a slice of strings in place
func sortStrings(s []string) {
	for i := 0; i < len(s)-1; i++ {
		for j := i + 1; j < len(s); j++ {
			if s[i] > s[j] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}

// truncateMiddle truncates a string in the middle if it exceeds maxLen,
// keeping the beginning and end visible with "..." in the middle.
// Returns the original string if it fits within maxLen.
func truncateMiddle(s string, maxLen int) string {
	if len(s) <= maxLen || maxLen < 5 {
		return s
	}

	// Reserve 3 chars for "..."
	// Split remaining space between start and end, favoring the end slightly
	remaining := maxLen - 3
	endLen := (remaining + 1) / 2
	startLen := remaining - endLen

	return s[:startLen] + "***" + s[len(s)-endLen:]
}
