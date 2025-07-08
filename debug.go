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
	fmt.Printf("%sPath: %q, HasHandler: %v, IsParam: %v, ParamName: %q, Wildcard: %v\n",
		indent, n.path, n.handler != nil, n.isParam, n.paramName, n.wildcard)
	for _, child := range n.children {
		r.debugNode(child, indent+"  ")
	}
}

func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}
