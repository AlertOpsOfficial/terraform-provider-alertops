package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// decodeObject unmarshals a JSON object response. An empty body yields an empty map.
func decodeObject(body []byte) (map[string]interface{}, error) {
	result := map[string]interface{}{}
	if len(body) == 0 {
		return result, nil
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// decodeArray unmarshals a JSON array response. An empty body or `{}` (204) yields an empty slice.
func decodeArray(body []byte) ([]interface{}, error) {
	var result []interface{}
	if len(body) == 0 || string(body) == "{}" {
		return result, nil
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// idFromResult reads a numeric or string identifier from an API response.
func idFromResult(result map[string]interface{}, key string) (string, bool) {
	switch v := result[key].(type) {
	case float64:
		return strconv.FormatInt(int64(v), 10), true
	case string:
		if v != "" {
			return v, true
		}
	}
	return "", false
}

func mapString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// mapScalarString renders a string, number or bool value as a string.
func mapScalarString(m map[string]interface{}, key string) string {
	switch v := m[key].(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	}
	return ""
}

func mapInt64(m map[string]interface{}, key string) int64 {
	switch v := m[key].(type) {
	case float64:
		return int64(v)
	case string:
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return 0
}

// mapBool reads a boolean, accepting the "true"/"1" strings the API sometimes returns.
func mapBool(m map[string]interface{}, key string) bool {
	switch v := m[key].(type) {
	case bool:
		return v
	case string:
		b, _ := strconv.ParseBool(v)
		return b
	case float64:
		return v != 0
	}
	return false
}

func mapObject(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key].(map[string]interface{}); ok {
		return v
	}
	return map[string]interface{}{}
}

func mapArray(m map[string]interface{}, key string) []interface{} {
	if v, ok := m[key].([]interface{}); ok {
		return v
	}
	return nil
}

// stringListFromAPI converts a JSON array of strings into a Terraform list value.
func stringListFromAPI(raw []interface{}) types.List {
	vals := []attr.Value{}
	for _, item := range raw {
		if s, ok := item.(string); ok {
			vals = append(vals, types.StringValue(s))
		}
	}
	return types.ListValueMust(types.StringType, vals)
}

func stringValues(values []string) []attr.Value {
	out := make([]attr.Value, 0, len(values))
	for _, v := range values {
		out = append(out, types.StringValue(v))
	}
	return out
}

// stringsFromElements returns the known string elements of a list or set value.
func stringsFromElements(elems []attr.Value) []string {
	out := []string{}
	for _, e := range elems {
		if s, ok := e.(types.String); ok && !s.IsNull() && !s.IsUnknown() {
			out = append(out, s.ValueString())
		}
	}
	return out
}

// int64sFromSet returns the known int64 elements of a set value.
func int64sFromSet(set types.Set) []int64 {
	out := []int64{}
	for _, e := range set.Elements() {
		if v, ok := e.(types.Int64); ok && !v.IsNull() && !v.IsUnknown() {
			out = append(out, v.ValueInt64())
		}
	}
	return out
}

func int64Set(values []int64) types.Set {
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	elems := make([]attr.Value, 0, len(values))
	for _, v := range values {
		elems = append(elems, types.Int64Value(v))
	}
	return types.SetValueMust(types.Int64Type, elems)
}

// isKnown reports whether a Terraform value is set and known.
func isKnown(v attr.Value) bool {
	return v != nil && !v.IsNull() && !v.IsUnknown()
}

// parseID converts a Terraform string ID to an integer for request bodies.
func parseID(s string) (int64, error) {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("expected a numeric ID, got %q", s)
	}
	return n, nil
}

// workflowIDsFromAPI extracts workflow_id values from a `workflows` array.
func workflowIDsFromAPI(raw []interface{}) []int64 {
	ids := []int64{}
	for _, item := range raw {
		if m, ok := item.(map[string]interface{}); ok {
			if id := mapInt64(m, "workflow_id"); id != 0 {
				ids = append(ids, id)
			}
		}
	}
	return ids
}

// reconcileWorkflowAttachments attaches/detaches workflows on an escalation policy or
// response play (basePath e.g. "/response_plays/123") so that exactly `desired` are attached.
func reconcileWorkflowAttachments(ctx context.Context, c *Client, basePath string, current, desired []int64) error {
	have := map[int64]bool{}
	for _, id := range current {
		have[id] = true
	}
	want := map[int64]bool{}
	for _, id := range desired {
		want[id] = true
	}

	var remove, add []map[string]interface{}
	for _, id := range current {
		if !want[id] {
			remove = append(remove, map[string]interface{}{"workflow_id": id})
		}
	}
	for _, id := range desired {
		if !have[id] {
			add = append(add, map[string]interface{}{"workflow_id": id})
			have[id] = true
		}
	}

	if len(remove) > 0 {
		if _, err := c.DeleteWithBody(ctx, basePath+"/workflows", remove); err != nil {
			return fmt.Errorf("error detaching workflows: %w", err)
		}
	}
	if len(add) > 0 {
		if _, err := c.Post(ctx, basePath+"/workflows", add); err != nil {
			return fmt.Errorf("error attaching workflows (only enabled workflows can be attached): %w", err)
		}
	}
	return nil
}
