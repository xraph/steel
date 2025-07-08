package forgerouter

import (
	"fmt"
)

// DebugRoutes debug method to inspect the routing tree
func (r *FastRouter) DebugRoutes() {
	fmt.Println("=== FastRouter Debug Info ===")
	for method, tree := range r.trees {
		fmt.Printf("Method: %s\n", method)
		if tree != nil {
			r.debugNode(tree, "  ")
		}
		fmt.Println()
	}
}

func (r *FastRouter) debugNode(n *node, indent string) {
	// fmt.Printf("%sPath: %q, HasHandler: %v, IsParam: %v, ParamName: %q, Wildcard: %v\n",
	// 	indent, n.path, n.handler != nil, n.isParam, n.paramName, n.wildcard)
	// for _, child := range n.children {
	// 	r.debugNode(child, indent+"  ")
	// }
}

// float64Ptr returns a pointer to the given float64 value
func float64Ptr(v float64) *float64 {
	return &v
}

// intPtr returns a pointer to the given int value
func intPtr(v int) *int {
	return &v
}

// stringPtr returns a pointer to the given string value
func stringPtr(v string) *string {
	return &v
}

// boolPtr returns a pointer to the given bool value
func boolPtr(v bool) *bool {
	return &v
}
