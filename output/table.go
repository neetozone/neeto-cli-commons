package output

import (
	"fmt"
	"sort"
	"strings"
)

func (pr *Printer) printTable(rows []map[string]interface{}, answers []string, indent int) {
	cols := pr.tableColumns(rows, answers)
	if len(cols) == 0 {
		pr.printIndentedJSON(rows, indent)
		return
	}

	headers := make([]string, len(cols))
	for i, col := range cols {
		headers[i] = FormatHeader(col)
	}

	grid := make([][]string, len(rows))
	for i, row := range rows {
		grid[i] = make([]string, len(cols))
		for j, col := range cols {
			grid[i][j] = formatValue(row[col])
		}
	}

	pr.renderGrid(headers, grid, indent)
}

func (pr *Printer) renderGrid(headers []string, grid [][]string, indent int) {
	prefix := indentPrefix(indent)
	widths := pr.calculateWidths(headers, grid, indent)
	pad := strings.Repeat(" ", colPadding)

	pr.printGridLine(headers, widths, pad, prefix, nil)

	_, _ = fmt.Fprint(pr.w(), prefix)
	for i, w := range widths {
		if i > 0 {
			_, _ = fmt.Fprint(pr.w(), pad)
		}
		_, _ = fmt.Fprint(pr.w(), strings.Repeat("─", w))
	}
	_, _ = fmt.Fprintln(pr.w())

	wrapping := wrappingColumns(grid)
	for _, row := range grid {
		pr.printGridLine(row, widths, pad, prefix, wrapping)
	}
}

func buildGrid(rows [][]interface{}) [][]string {
	grid := make([][]string, len(rows))
	for i, row := range rows {
		grid[i] = make([]string, len(row))
		for j, val := range row {
			grid[i][j] = formatValue(val)
		}
	}
	return grid
}

func wrappingColumns(grid [][]string) []bool {
	if len(grid) == 0 {
		return nil
	}
	wrapping := make([]bool, len(grid[0]))
	for _, row := range grid {
		for i, val := range row {
			if i < len(wrapping) && isURL(val) {
				wrapping[i] = true
			}
		}
	}
	return wrapping
}

func (pr *Printer) printGridLine(cells []string, widths []int, pad, prefix string, wrapping []bool) {
	segments := make([][]string, len(cells))
	height := 1
	for i, val := range cells {
		if wrapping != nil && i < len(wrapping) && wrapping[i] {
			segments[i] = wrapRunes(val, widths[i])
		} else {
			segments[i] = []string{truncate(val, widths[i])}
		}
		if len(segments[i]) > height {
			height = len(segments[i])
		}
	}

	for line := 0; line < height; line++ {
		_, _ = fmt.Fprint(pr.w(), prefix)
		for i := range cells {
			if i > 0 {
				_, _ = fmt.Fprint(pr.w(), pad)
			}
			var cell string
			if line < len(segments[i]) {
				cell = segments[i][line]
			}
			if i < len(cells)-1 {
				cell = padRight(cell, widths[i])
			}
			_, _ = fmt.Fprint(pr.w(), cell)
		}
		_, _ = fmt.Fprintln(pr.w())
	}
}

func wrapRunes(s string, width int) []string {
	runes := []rune(s)
	if width <= 0 || len(runes) <= width {
		return []string{s}
	}
	var out []string
	for start := 0; start < len(runes); start += width {
		end := start + width
		if end > len(runes) {
			end = len(runes)
		}
		out = append(out, string(runes[start:end]))
	}
	return out
}

func (pr *Printer) pickColumns(rows []map[string]interface{}) []string {
	displayable := map[string]bool{}
	for k, v := range rows[0] {
		if !isDisplayable(v) {
			continue
		}
		if _, isSlice := v.([]interface{}); isSlice && !anyNonEmptySlice(rows, k) {
			continue
		}
		displayable[k] = true
	}
	for _, row := range rows[1:] {
		for k, v := range row {
			if displayable[k] && !isDisplayable(v) {
				delete(displayable, k)
			}
		}
	}

	urlFields := map[string]bool{}
	var urlCols []string
	for _, row := range rows {
		for k, v := range row {
			if urlFields[k] {
				continue
			}
			if s, ok := v.(string); ok && isURL(s) {
				urlFields[k] = true
				urlCols = append(urlCols, k)
			}
		}
	}
	sort.Strings(urlCols)

	budget := max(1, maxTableColumns-len(urlCols))

	var candidates []string
	for k := range displayable {
		if !urlFields[k] {
			candidates = append(candidates, k)
		}
	}

	cols := pr.orderedKeys(candidates)
	if len(cols) > budget {
		cols = cols[:budget]
	}

	return append(cols, urlCols...)
}

func (pr *Printer) orderedKeys(keys []string) []string {
	remaining := make(map[string]bool, len(keys))
	for _, k := range keys {
		remaining[k] = true
	}

	var ordered []string
	for _, f := range pr.priorityFields() {
		if remaining[f] {
			ordered = append(ordered, f)
			delete(remaining, f)
		}
	}

	var rest []string
	for _, k := range keys {
		if remaining[k] {
			rest = append(rest, k)
		}
	}
	sort.Strings(rest)

	return append(ordered, rest...)
}

func (pr *Printer) tableUninformative(rows []map[string]interface{}, answers []string) bool {
	if len(pr.tableColumns(rows, answers)) > 2 {
		return false
	}
	for _, row := range rows {
		if hasNestedData(row) {
			return true
		}
	}
	return false
}

func hasNestedData(row map[string]interface{}) bool {
	for _, v := range row {
		if !isDisplayable(v) && !isEmptyContainer(v) {
			return true
		}
	}
	return false
}

func (pr *Printer) calculateWidths(headers []string, grid [][]string, indent int) []int {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = displayWidth(h)
	}
	for _, row := range grid {
		for i, val := range row {
			if w := displayWidth(val); w > widths[i] {
				widths[i] = w
			}
		}
	}

	totalPad := (len(headers)-1)*colPadding + indent*2
	available := pr.terminalWidth() - totalPad

	total := 0
	for _, w := range widths {
		total += w
	}

	if total <= available {
		return widths
	}

	for i := range widths {
		widths[i] = min(widths[i], max(minColWidth, widths[i]*available/total))
	}

	return shrinkToFit(widths, available)
}

func shrinkToFit(widths []int, available int) []int {
	for {
		total := 0
		for _, w := range widths {
			total += w
		}
		if total <= available {
			return widths
		}

		widest, target := 0, -1
		for i, w := range widths {
			if w > minColWidth && w > widest {
				widest, target = w, i
			}
		}
		if target < 0 {
			return widths
		}
		widths[target]--
	}
}

func anyNonEmptySlice(rows []map[string]interface{}, key string) bool {
	for _, row := range rows {
		if arr, ok := row[key].([]interface{}); ok && len(arr) > 0 {
			return true
		}
	}
	return false
}

func flattenLabelValues(rows []map[string]interface{}) ([]map[string]interface{}, []string) {
	keys := labelValueKeys(rows)
	if len(keys) == 0 {
		return rows, nil
	}

	flattened := make([]map[string]interface{}, len(rows))
	for i, row := range rows {
		flattened[i] = make(map[string]interface{}, len(row))
		for k, v := range row {
			flattened[i][k] = v
		}
	}

	reserved := reservedKeys(rows, keys)

	var labels []string
	seen := map[string]bool{}
	for _, key := range keys {
		for i, row := range rows {
			items, ok := row[key].([]interface{})
			if !ok {
				continue
			}
			delete(flattened[i], key)
			for _, item := range items {
				obj, _ := item.(map[string]interface{})
				label := responseLabel(obj)
				if reserved[label] {
					continue
				}
				flattened[i][label] = obj["value"]
				if !seen[label] {
					seen[label] = true
					labels = append(labels, label)
				}
			}
		}
	}

	return flattened, labels
}

func reservedKeys(rows []map[string]interface{}, flattenedKeys []string) map[string]bool {
	flattening := make(map[string]bool, len(flattenedKeys))
	for _, key := range flattenedKeys {
		flattening[key] = true
	}

	reserved := map[string]bool{}
	for _, row := range rows {
		for key := range row {
			if !flattening[key] {
				reserved[key] = true
			}
		}
	}
	return reserved
}

func labelValueKeys(rows []map[string]interface{}) []string {
	var keys []string
	seen := map[string]bool{}
	for _, row := range rows {
		for key, value := range row {
			items, ok := value.([]interface{})
			if seen[key] || !ok || !isLabelValueList(items) {
				continue
			}
			seen[key] = true
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func (pr *Printer) tableColumns(rows []map[string]interface{}, answers []string) []string {
	if len(answers) == 0 {
		return pr.pickColumns(rows)
	}

	cols := pr.pickColumns(withoutKeys(rows, answers))
	for _, answer := range answers {
		if len(cols) >= maxTableColumns {
			break
		}
		cols = append(cols, answer)
	}
	return cols
}

func withoutKeys(rows []map[string]interface{}, keys []string) []map[string]interface{} {
	dropped := make(map[string]bool, len(keys))
	for _, key := range keys {
		dropped[key] = true
	}

	out := make([]map[string]interface{}, len(rows))
	for i, row := range rows {
		out[i] = make(map[string]interface{}, len(row))
		for k, v := range row {
			if !dropped[k] {
				out[i][k] = v
			}
		}
	}
	return out
}
