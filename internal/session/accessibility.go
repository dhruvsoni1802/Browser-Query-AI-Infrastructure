package session

import (
	"encoding/json"
	"fmt"
)

// AXNode represents a node in the accessibility tree
type AXNode struct {
	Role      string    `json:"role"`
	Name      string    `json:"name,omitempty"`
	Level     int       `json:"level,omitempty"`
	Value     string    `json:"value,omitempty"`
	Focusable bool      `json:"focusable,omitempty"`
	Children  []*AXNode `json:"children"`
}

// AccessibilityTree represents the full accessibility tree for a page
type AccessibilityTree struct {
	PageID string    `json:"page_id"`
	Nodes  []*AXNode `json:"nodes"`
}

// cdpAXNode represents a raw CDP accessibility node from Accessibility.getFullAXTree
type cdpAXNode struct {
	NodeID     string        `json:"nodeId"`
	Role       cdpAXValue    `json:"role"`
	Name       *cdpAXValue   `json:"name,omitempty"`
	Value      *cdpAXValue   `json:"value,omitempty"`
	Properties []cdpAXProp   `json:"properties,omitempty"`
	ChildIDs   []string      `json:"childIds,omitempty"`
	Ignored    bool          `json:"ignored"`
}

// cdpAXValue represents a CDP accessibility value
type cdpAXValue struct {
	Type  string `json:"type"`
	Value interface{} `json:"value"`
}

// cdpAXProp represents a CDP accessibility property
type cdpAXProp struct {
	Name  string     `json:"name"`
	Value cdpAXValue `json:"value"`
}

// GetAccessibilityTree retrieves the accessibility tree for a page using CDP
func (s *Session) GetAccessibilityTree(targetID string) (*AccessibilityTree, error) {
	// Call CDP Accessibility.getFullAXTree
	result, err := s.CDPClient.SendCommandToTarget(targetID, "Accessibility.getFullAXTree", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get accessibility tree: %w", err)
	}

	// Parse the raw CDP response
	var response struct {
		Nodes []cdpAXNode `json:"nodes"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return nil, fmt.Errorf("failed to parse accessibility tree response: %w", err)
	}

	// Build a lookup map for parent-child relationships
	nodeMap := make(map[string]*cdpAXNode, len(response.Nodes))
	for i := range response.Nodes {
		nodeMap[response.Nodes[i].NodeID] = &response.Nodes[i]
	}

	// Find root nodes (nodes that are not children of any other node)
	childSet := make(map[string]bool)
	for _, node := range response.Nodes {
		for _, childID := range node.ChildIDs {
			childSet[childID] = true
		}
	}

	// Build the tree starting from root nodes
	var roots []*AXNode
	for _, node := range response.Nodes {
		if !childSet[node.NodeID] && !node.Ignored {
			roots = append(roots, buildAXTree(&node, nodeMap))
		}
	}

	// If no explicit roots found, build from the first non-ignored node
	if len(roots) == 0 && len(response.Nodes) > 0 {
		for i := range response.Nodes {
			if !response.Nodes[i].Ignored {
				roots = append(roots, buildAXTree(&response.Nodes[i], nodeMap))
				break
			}
		}
	}

	return &AccessibilityTree{
		PageID: targetID,
		Nodes:  roots,
	}, nil
}

// buildAXTree recursively converts a CDP AX node into our clean AXNode format
func buildAXTree(cdpNode *cdpAXNode, nodeMap map[string]*cdpAXNode) *AXNode {
	node := &AXNode{
		Role:     stringValue(cdpNode.Role),
		Children: make([]*AXNode, 0),
	}

	// Extract name
	if cdpNode.Name != nil {
		node.Name = stringValue(*cdpNode.Name)
	}

	// Extract value
	if cdpNode.Value != nil {
		node.Value = stringValue(*cdpNode.Value)
	}

	// Extract properties (level, focusable, etc.)
	for _, prop := range cdpNode.Properties {
		switch prop.Name {
		case "level":
			if v, ok := prop.Value.Value.(float64); ok {
				node.Level = int(v)
			}
		case "focusable":
			if v, ok := prop.Value.Value.(bool); ok {
				node.Focusable = v
			}
		}
	}

	// Recursively build children
	for _, childID := range cdpNode.ChildIDs {
		childCDP, exists := nodeMap[childID]
		if !exists || childCDP.Ignored {
			continue
		}
		node.Children = append(node.Children, buildAXTree(childCDP, nodeMap))
	}

	return node
}

// stringValue extracts a string from a cdpAXValue
func stringValue(v cdpAXValue) string {
	if s, ok := v.Value.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v.Value)
}
