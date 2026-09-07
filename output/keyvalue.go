package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type field struct {
	key   string
	raw   json.RawMessage
	value interface{}
}

func decodeFields(data json.RawMessage) ([]field, bool) {
	dec := json.NewDecoder(bytes.NewReader(data))
	t, err := dec.Token()
	if err != nil || t != json.Delim('{') {
		return nil, false
	}

	var fields []field
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return nil, false
		}
		key, ok := t.(string)
		if !ok {
			return nil, false
		}

		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, false
		}
		var value interface{}
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, false
		}
		fields = append(fields, field{key: key, raw: raw, value: value})
	}
	return fields, true
}

func singleNestedObject(data json.RawMessage) (json.RawMessage, bool) {
	fields, ok := decodeFields(data)
	if !ok || len(fields) != 1 {
		return nil, false
	}
	if _, ok := fields[0].value.(map[string]interface{}); ok {
		return fields[0].raw, true
	}
	return nil, false
}

func (pr *Printer) printObject(data json.RawMessage, depth int) {
	fields, ok := decodeFields(data)
	if !ok {
		pr.printIndentedJSON(data, depth)
		return
	}
	pr.renderFields(fields, depth)
}

func (pr *Printer) renderFields(fields []field, depth int) {
	prefix := indentPrefix(depth)

	labelWidth := 0
	for _, f := range fields {
		if expands(f.value, depth) {
			continue
		}
		if l := len(FormatHeader(f.key)); l > labelWidth {
			labelWidth = l
		}
	}

	for _, f := range fields {
		label := FormatHeader(f.key)
		if !expands(f.value, depth) {
			_, _ = fmt.Fprintf(pr.w(), "%s%-*s  %s\n", prefix, labelWidth, label, inlineValue(f.value))
			continue
		}
		_, _ = fmt.Fprintf(pr.w(), "%s%s\n", prefix, label)
		pr.expand(f, depth+1)
	}
}

func expands(v interface{}, depth int) bool {
	if depth >= maxRenderDepth {
		return false
	}
	switch val := v.(type) {
	case map[string]interface{}:
		return len(val) > 0
	case []interface{}:
		return len(val) > 0 && allObjects(val)
	}
	return false
}

func (pr *Printer) expand(f field, depth int) {
	switch val := f.value.(type) {
	case map[string]interface{}:
		pr.printObject(f.raw, depth)
	case []interface{}:
		if isLabelValueList(val) {
			pr.printLabelValues(val, depth)
			return
		}
		pr.printArray(f.raw, depth)
	}
}

func (pr *Printer) printArray(data json.RawMessage, depth int) {
	var rows []map[string]interface{}
	if err := json.Unmarshal(data, &rows); err == nil && len(rows) > 0 {
		pr.printTable(rows, depth)
		return
	}
	pr.printIndentedJSON(data, depth)
}

func (pr *Printer) printLabelValues(items []interface{}, depth int) {
	prefix := indentPrefix(depth)

	labelWidth := 0
	for _, item := range items {
		obj, _ := item.(map[string]interface{})
		if l := len(responseLabel(obj)); l > labelWidth {
			labelWidth = l
		}
	}

	for _, item := range items {
		obj, _ := item.(map[string]interface{})
		_, _ = fmt.Fprintf(pr.w(), "%s%-*s  %s\n", prefix, labelWidth, responseLabel(obj), inlineValue(obj["value"]))
	}
}

func isLabelValueList(items []interface{}) bool {
	if len(items) == 0 {
		return false
	}
	for _, item := range items {
		if !isLabelValue(item) {
			return false
		}
	}
	return true
}

func isLabelValue(item interface{}) bool {
	obj, ok := item.(map[string]interface{})
	if !ok {
		return false
	}
	if _, ok := obj["value"]; !ok {
		return false
	}
	return responseLabel(obj) != ""
}

func responseLabel(obj map[string]interface{}) string {
	for _, key := range []string{"label", "name", "title"} {
		if s, ok := obj[key].(string); ok && s != "" {
			return s
		}
	}
	return ""
}

func (pr *Printer) printRecordBlocks(raws []json.RawMessage, rows []map[string]interface{}) {
	for i := range rows {
		if i > 0 {
			_, _ = fmt.Fprintln(pr.w())
		}
		fields, ok := decodeFields(raws[i])
		if !ok {
			pr.printIndentedJSON(raws[i], 1)
			continue
		}
		pr.renderFields(pr.orderBlockFields(fields, rows[i]), 1)
	}
}

func (pr *Printer) orderBlockFields(fields []field, row map[string]interface{}) []field {
	index := make(map[string]field, len(fields))
	for _, f := range fields {
		index[f.key] = f
	}

	used := map[string]bool{}
	var order []string
	for _, k := range pr.pickColumns([]map[string]interface{}{row}) {
		if _, ok := index[k]; ok {
			order = append(order, k)
			used[k] = true
		}
	}

	var inline, nested []string
	for _, f := range fields {
		switch {
		case used[f.key]:
		case isDisplayable(f.value):
			inline = append(inline, f.key)
		default:
			nested = append(nested, f.key)
		}
	}
	sort.Strings(inline)
	sort.Strings(nested)

	order = append(order, inline...)
	order = append(order, nested...)

	out := make([]field, 0, len(order))
	for _, k := range order {
		out = append(out, index[k])
	}
	return out
}

func (pr *Printer) printIndentedJSON(v interface{}, indent int) {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		if raw, ok := v.(json.RawMessage); ok {
			_, _ = fmt.Fprintf(pr.w(), "%s%s\n", indentPrefix(indent), string(raw))
		}
		return
	}
	prefix := indentPrefix(indent)
	for _, line := range strings.Split(string(out), "\n") {
		_, _ = fmt.Fprintf(pr.w(), "%s%s\n", prefix, line)
	}
}
