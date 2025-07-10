package steel

import (
	"strings"
)

type node struct {
	path      string
	handler   HandlerFunc
	children  []*node
	paramName string
	wildcard  bool
	isParam   bool
	methods   map[string]HandlerFunc
}

// Enhanced node findHandler with proper path matching
func (n *node) findHandler(path string, params *Params) HandlerFunc {
	// Root path handling
	if path == "/" {
		if n.path == "/" && n.handler != nil {
			return n.handler
		}
		// Check for root handler in children
		for _, child := range n.children {
			if child.path == "/" && child.handler != nil {
				return child.handler
			}
		}
	}

	// Empty path - current node has handler
	if path == "" && n.handler != nil {
		return n.handler
	}

	// If path is empty, no match
	if path == "" {
		return nil
	}

	// Remove leading slash for processing
	if path[0] == '/' {
		path = path[1:]
	}

	// If path is now empty after removing slash, check for handler
	if path == "" && n.handler != nil {
		return n.handler
	}

	// Find the next segment
	var segment string
	var remaining string

	slashPos := strings.Index(path, "/")
	if slashPos == -1 {
		// No more slashes, this is the last segment
		segment = path
		remaining = ""
	} else {
		segment = path[:slashPos]
		remaining = path[slashPos:] // Keep the leading slash
	}

	// Try static children first
	for _, child := range n.children {
		if child.isParam || child.wildcard {
			continue
		}

		if child.path == segment {
			if remaining == "" {
				// This is the end of the path
				if child.handler != nil {
					return child.handler
				}
			} else {
				// Continue with remaining path
				if handler := child.findHandler(remaining, params); handler != nil {
					return handler
				}
			}
		}
	}

	// Try parameter children
	for _, child := range n.children {
		if !child.isParam {
			continue
		}

		// Store parameter value
		params.Set(child.paramName, segment)

		if remaining == "" {
			// This is the end of the path
			if child.handler != nil {
				return child.handler
			}
		} else {
			// Continue with remaining path
			if handler := child.findHandler(remaining, params); handler != nil {
				return handler
			}
		}

		// Remove parameter if no match (backtrack)
		params.Remove(child.paramName)
	}

	// Try wildcard children
	for _, child := range n.children {
		if child.wildcard && child.handler != nil {
			return child.handler
		}
	}

	return nil
}

// Fixed addRoute method
func (n *node) addRoute(path string, handler HandlerFunc) {
	// Handle empty path
	if path == "" {
		n.handler = handler
		return
	}

	// Handle root path
	if path == "/" {
		n.handler = handler
		return
	}

	// Remove leading slash for processing
	if path[0] == '/' {
		path = path[1:]
	}

	// If path is empty after removing slash, set handler on current node
	if path == "" {
		n.handler = handler
		return
	}

	// Find the first segment
	var segment string
	var remaining string

	slashPos := strings.Index(path, "/")
	if slashPos == -1 {
		// No more slashes, this is the last segment
		segment = path
		remaining = ""
	} else {
		segment = path[:slashPos]
		remaining = path[slashPos+1:] // Remove the leading slash from remaining
	}

	// Handle parameter segment
	if len(segment) > 0 && segment[0] == ':' {
		paramName := segment[1:]

		// Find or create parameter child
		var paramChild *node
		for _, child := range n.children {
			if child.isParam && child.paramName == paramName {
				paramChild = child
				break
			}
		}

		if paramChild == nil {
			paramChild = &node{
				isParam:   true,
				paramName: paramName,
				children:  []*node{},
			}
			n.children = append(n.children, paramChild)
		}

		if remaining == "" {
			// This is the end of the path
			paramChild.handler = handler
		} else {
			// Continue with remaining path
			paramChild.addRoute(remaining, handler)
		}
		return
	}

	// Handle wildcard segment
	if segment == "*" {
		wildcardChild := &node{
			wildcard: true,
			handler:  handler,
		}
		n.children = append(n.children, wildcardChild)
		return
	}

	// Handle static segment
	var staticChild *node
	for _, child := range n.children {
		if !child.isParam && !child.wildcard && child.path == segment {
			staticChild = child
			break
		}
	}

	if staticChild == nil {
		staticChild = &node{
			path:     segment,
			children: []*node{},
		}
		n.children = append(n.children, staticChild)
	}

	if remaining == "" {
		// This is the end of the path
		staticChild.handler = handler
	} else {
		// Continue with remaining path
		staticChild.addRoute(remaining, handler)
	}
}

// Helper function to find longest common prefix
func longestCommonPrefix(a, b string) string {
	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}

	for i := 0; i < minLen; i++ {
		if a[i] != b[i] {
			return a[:i]
		}
	}
	return a[:minLen]
}
