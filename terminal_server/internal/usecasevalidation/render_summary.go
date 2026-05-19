package usecasevalidation

import (
	"fmt"
	"sort"
	"strings"

	uiv1 "github.com/curtcox/terminals/terminal_server/gen/go/ui/v1"
)

func summarizeNode(node *uiv1.Node) string {
	if node == nil {
		return "(empty)"
	}
	parts := summarizeNodeHead(node)
	parts = append(parts, summarizeNodePropsList(node)...)
	if childSummary := summarizeNodeChildrenList(node); childSummary != "" {
		parts = append(parts, childSummary)
	}
	return strings.Join(parts, " ")
}

func summarizeNodeHead(node *uiv1.Node) []string {
	parts := []string{nodeWidgetName(node)}
	if node.GetId() != "" {
		parts = append(parts, "#"+node.GetId())
	}
	if text := node.GetText(); text != nil && text.GetValue() != "" {
		parts = append(parts, quoteSummary(text.GetValue()))
	}
	if button := node.GetButton(); button != nil && button.GetLabel() != "" {
		parts = append(parts, "button "+quoteSummary(button.GetLabel()))
	}
	return parts
}

func summarizeNodePropsList(node *uiv1.Node) []string {
	props := node.GetProps()
	if len(props) == 0 {
		return nil
	}
	keys := make([]string, 0, len(props))
	for key := range props {
		if key == "id" || key == "draw_ops_json" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		if props[key] != "" {
			out = append(out, key+"="+quoteSummary(props[key]))
		}
	}
	return out
}

func summarizeNodeChildrenList(node *uiv1.Node) string {
	children := node.GetChildren()
	if len(children) == 0 {
		return ""
	}
	childSummaries := make([]string, 0, min(len(children), 4))
	for _, child := range children {
		if len(childSummaries) == 4 {
			break
		}
		childSummaries = append(childSummaries, summarizeNode(child))
	}
	if len(children) > 4 {
		childSummaries = append(childSummaries, fmt.Sprintf("%d more", len(children)-4))
	}
	return "children=[" + strings.Join(childSummaries, "; ") + "]"
}
