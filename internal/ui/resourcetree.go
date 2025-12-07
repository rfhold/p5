package ui

import (
	"sort"

	"github.com/rfhold/p5/internal/pulumi"
)

// organizeItemsAsTree sorts items into tree order (parent followed by children)
// and sets Depth and IsLast for each item
func organizeItemsAsTree(items []ResourceItem) []ResourceItem {
	if len(items) == 0 {
		return items
	}

	// Build parent -> children map
	childrenOf := make(map[string][]int) // parent URN -> indices of children
	rootIndices := make([]int, 0)

	for i, item := range items {
		if item.Parent == "" {
			rootIndices = append(rootIndices, i)
		} else {
			childrenOf[item.Parent] = append(childrenOf[item.Parent], i)
		}
	}

	// Sort roots and children by URN for deterministic ordering
	sort.Slice(rootIndices, func(i, j int) bool {
		return items[rootIndices[i]].URN < items[rootIndices[j]].URN
	})
	for parent := range childrenOf {
		children := childrenOf[parent]
		sort.Slice(children, func(i, j int) bool {
			return items[children[i]].URN < items[children[j]].URN
		})
	}

	// Flatten tree into result slice with depth/isLast info
	result := make([]ResourceItem, 0, len(items))

	var addItem func(idx int, depth int, isLast bool)
	addItem = func(idx int, depth int, isLast bool) {
		item := items[idx]
		item.Depth = depth
		item.IsLast = isLast
		result = append(result, item)

		children := childrenOf[item.URN]
		for i, childIdx := range children {
			isLastChild := i == len(children)-1
			addItem(childIdx, depth+1, isLastChild)
		}
	}

	for i, rootIdx := range rootIndices {
		isLastRoot := i == len(rootIndices)-1
		addItem(rootIdx, 0, isLastRoot)
	}

	return result
}

// ensureParentExists adds a placeholder parent item if it doesn't exist
// This recursively ensures all ancestors exist
func (r *ResourceList) ensureParentExists(parentURN string) {
	if parentURN == "" {
		return
	}

	// Check if parent already exists
	for _, item := range r.items {
		if item.URN == parentURN {
			return // Parent exists
		}
	}

	// Parent doesn't exist - create a placeholder with OpSame
	// Extract type and name from URN
	parentType := extractResourceType(parentURN)
	parentName := extractResourceName(parentURN)

	// Add the parent placeholder
	// Note: We don't know the grandparent URN from the URN alone,
	// but when the parent's event arrives (if ever), it will update with correct parent
	r.items = append(r.items, ResourceItem{
		URN:    parentURN,
		Type:   parentType,
		Name:   parentName,
		Op:     OpSame,
		Status: StatusNone,
		Parent: "", // Will be updated if parent's event arrives later
	})
}

// extractResourceType gets the resource type from a URN
// URN format: urn:pulumi:stack::project::type::name
func extractResourceType(urn string) string {
	parts := splitURN(urn)
	if len(parts) >= 4 {
		return parts[3]
	}
	return urn
}

// extractResourceName is a local wrapper that calls the shared implementation.
// URN format: urn:pulumi:stack::project::type::name
func extractResourceName(urn string) string {
	return pulumi.ExtractResourceName(urn)
}

// splitURN splits a URN by :: delimiter
func splitURN(urn string) []string {
	var parts []string
	current := ""
	for i := 0; i < len(urn); i++ {
		if i < len(urn)-1 && urn[i] == ':' && urn[i+1] == ':' {
			parts = append(parts, current)
			current = ""
			i++ // Skip the second ':'
		} else {
			current += string(urn[i])
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// isReplaceOp returns true for all replace-related operations
func isReplaceOp(op ResourceOp) bool {
	return op == OpReplace || op == OpCreateReplace || op == OpDeleteReplace
}

// rebuildVisibleIndex applies filters to build the visible index
func (r *ResourceList) rebuildVisibleIndex() {
	r.visibleIdx = make([]int, 0, len(r.items))

	if r.showAllOps {
		// Show everything
		for i := range r.items {
			r.visibleIdx = append(r.visibleIdx, i)
		}
	} else {
		// Build set of URNs that have changes (not OpSame)
		// and URNs that are ancestors of changed items
		visibleURNs := make(map[string]bool)

		// First pass: mark all items with changes
		for _, item := range r.items {
			if item.Op != OpSame {
				visibleURNs[item.URN] = true
			}
		}

		// Second pass: mark all ancestors of changed items
		for _, item := range r.items {
			if item.Op != OpSame && item.Parent != "" {
				r.markAncestorsVisible(item.Parent, visibleURNs)
			}
		}

		// Third pass: add visible items in order
		for i, item := range r.items {
			if visibleURNs[item.URN] {
				r.visibleIdx = append(r.visibleIdx, i)
			}
		}
	}

	// Clamp cursor
	if r.cursor >= len(r.visibleIdx) {
		r.cursor = len(r.visibleIdx) - 1
		if r.cursor < 0 {
			r.cursor = 0
		}
	}
	r.ensureCursorVisible()
}

// markAncestorsVisible recursively marks all ancestors as visible
func (r *ResourceList) markAncestorsVisible(parentURN string, visibleURNs map[string]bool) {
	if parentURN == "" {
		return
	}
	if visibleURNs[parentURN] {
		return // Already marked
	}
	visibleURNs[parentURN] = true

	// Find the parent item and recurse to its parent
	for _, item := range r.items {
		if item.URN == parentURN {
			if item.Parent != "" {
				r.markAncestorsVisible(item.Parent, visibleURNs)
			}
			return
		}
	}
}

// buildAncestorIsLast traces back through the parent chain to determine
// which ancestors were the last child of their parent (for tree line drawing)
func (r *ResourceList) buildAncestorIsLast(itemIdx int) []bool {
	item := r.items[itemIdx]
	if item.Depth == 0 {
		return nil
	}

	result := make([]bool, item.Depth-1)

	// Build a URN -> item index map for quick lookup
	urnToIdx := make(map[string]int)
	for i, it := range r.items {
		urnToIdx[it.URN] = i
	}

	// Trace back through parent chain
	currentURN := item.Parent
	for level := item.Depth - 2; level >= 0; level-- {
		if parentIdx, ok := urnToIdx[currentURN]; ok {
			parent := r.items[parentIdx]
			result[level] = parent.IsLast
			currentURN = parent.Parent
		} else {
			break
		}
	}

	return result
}
