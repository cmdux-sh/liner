package app

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/table"
)

type projectRunUsage struct {
	Runs        int
	Input       int
	Output      int
	CacheRead   int
	CacheCreate int
	CostUSD     float64
	CostKnown   bool
}

func (u projectRunUsage) totalTokens() int {
	return u.Input + u.Output + u.CacheRead + u.CacheCreate
}

func (u *projectRunUsage) add(other projectRunUsage) {
	u.Runs += other.Runs
	u.Input += other.Input
	u.Output += other.Output
	u.CacheRead += other.CacheRead
	u.CacheCreate += other.CacheCreate
	if other.CostKnown {
		u.CostUSD += other.CostUSD
		u.CostKnown = true
	}
}

func readProjectRunUsage(project string) (projectRunUsage, bool) {
	if strings.TrimSpace(project) == "" {
		return projectRunUsage{}, false
	}
	base := projectAbsPath(project, ".liner-runs")
	if info, err := os.Stat(base); err != nil || !info.IsDir() {
		return projectRunUsage{}, false
	}
	var total projectRunUsage
	_ = filepath.WalkDir(base, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry == nil || entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			return nil
		}
		if usage, ok := parseRunUsageLog(path); ok {
			total.add(usage)
		}
		return nil
	})
	return total, total.Runs > 0
}

func parseRunUsageLog(path string) (projectRunUsage, bool) {
	file, err := os.Open(path)
	if err != nil {
		return projectRunUsage{}, false
	}
	defer file.Close()

	var incremental projectRunUsage
	var result projectRunUsage
	seenRun := false
	resultHasUsage := false
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var rec map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			continue
		}
		recType := stringField(rec, "type")
		kind := stringField(rec, "kind")
		switch {
		case recType == "_liner_meta" || kind == "runner_start":
			seenRun = true
		case recType == "turn.completed":
			seenRun = true
			incremental.add(tokenUsageFromValue(rec["usage"], codexUsageKeys))
		case recType == "assistant":
			seenRun = true
			if message, ok := rec["message"].(map[string]any); ok {
				incremental.add(tokenUsageFromValue(message["usage"], claudeUsageKeys))
			}
		case kind == "tokens":
			seenRun = true
			incremental.add(tokenUsageFromNormalizedEvent(rec))
		case recType == "result":
			seenRun = true
			result = tokenUsageFromValue(rec["usage"], claudeUsageKeys)
			if result.totalTokens() == 0 {
				result = tokenUsageFromModelUsage(rec["modelUsage"])
			}
			if cost, ok := numberField(rec, "total_cost_usd"); ok {
				result.CostUSD = cost
				result.CostKnown = true
			}
			resultHasUsage = result.totalTokens() > 0 || result.CostKnown
		case kind == "summary":
			seenRun = true
			if cost, ok := numberField(rec, "costUsd"); ok {
				incremental.CostUSD += cost
				incremental.CostKnown = true
			}
		}
	}
	if resultHasUsage {
		result.Runs = 1
		return result, true
	}
	if incremental.totalTokens() > 0 || incremental.CostKnown {
		incremental.Runs = 1
		return incremental, true
	}
	if seenRun {
		return projectRunUsage{Runs: 1}, true
	}
	return projectRunUsage{}, false
}

var codexUsageKeys = usageKeys{
	Input:       "input_tokens",
	Output:      "output_tokens",
	CacheRead:   "cached_input_tokens",
	CacheCreate: "cache_creation_input_tokens",
}

var claudeUsageKeys = usageKeys{
	Input:       "input_tokens",
	Output:      "output_tokens",
	CacheRead:   "cache_read_input_tokens",
	CacheCreate: "cache_creation_input_tokens",
}

type usageKeys struct {
	Input       string
	Output      string
	CacheRead   string
	CacheCreate string
}

func tokenUsageFromValue(value any, keys usageKeys) projectRunUsage {
	usage, ok := value.(map[string]any)
	if !ok {
		return projectRunUsage{}
	}
	cacheRead := intField(usage, keys.CacheRead)
	if cacheRead == 0 && keys.CacheRead != "cached_input_tokens" {
		cacheRead = intField(usage, "cached_input_tokens")
	}
	if cacheRead == 0 && keys.CacheRead != "cache_read_input_tokens" {
		cacheRead = intField(usage, "cache_read_input_tokens")
	}
	return projectRunUsage{
		Input:       intField(usage, keys.Input),
		Output:      intField(usage, keys.Output),
		CacheRead:   cacheRead,
		CacheCreate: intField(usage, keys.CacheCreate),
	}
}

func tokenUsageFromNormalizedEvent(rec map[string]any) projectRunUsage {
	return projectRunUsage{
		Input:       intField(rec, "inputTokens"),
		Output:      intField(rec, "outputTokens"),
		CacheRead:   intField(rec, "cacheReadTokens"),
		CacheCreate: intField(rec, "cacheCreationTokens"),
	}
}

func tokenUsageFromModelUsage(value any) projectRunUsage {
	models, ok := value.(map[string]any)
	if !ok {
		return projectRunUsage{}
	}
	var total projectRunUsage
	for _, raw := range models {
		modelUsage, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		total.Input += intField(modelUsage, "inputTokens")
		total.Output += intField(modelUsage, "outputTokens")
	}
	return total
}

func stringField(fields map[string]any, key string) string {
	value, _ := fields[key].(string)
	return value
}

func intField(fields map[string]any, key string) int {
	value, ok := numberField(fields, key)
	if !ok {
		return 0
	}
	return int(value)
}

func numberField(fields map[string]any, key string) (float64, bool) {
	switch value := fields[key].(type) {
	case float64:
		return value, true
	case int:
		return float64(value), true
	default:
		return 0, false
	}
}

func projectRunUsageView(project string, width int) string {
	usage, ok := readProjectRunUsage(project)
	if !ok {
		return ""
	}
	cost := "not reported"
	if usage.CostKnown {
		cost = fmt.Sprintf("$%.4f", usage.CostUSD)
	}
	rows := []table.Row{
		{"runs", formatCount(usage.Runs)},
		{"total tokens", formatCount(usage.totalTokens())},
		{"input", formatCount(usage.Input)},
		{"output", formatCount(usage.Output)},
		{"cache read", formatCount(usage.CacheRead)},
		{"cache create", formatCount(usage.CacheCreate)},
		{"cost", cost},
	}
	usageTable := newDataTable(
		[]table.Column{
			{Title: "Metric", Width: 14},
			{Title: "Value", Width: max(18, width-18)},
		},
		rows,
		width,
		len(rows)+1,
		false,
	)
	return usageTable.View()
}

func projectRunUsageSummary(project string) string {
	usage, ok := readProjectRunUsage(project)
	if !ok {
		return ""
	}
	if usage.Runs == 0 {
		return "no runs"
	}
	return intLabel(usage.Runs, "run")
}

func formatCount(value int) string {
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	text := fmt.Sprintf("%d", value)
	var groups []string
	for len(text) > 3 {
		groups = append([]string{text[len(text)-3:]}, groups...)
		text = text[:len(text)-3]
	}
	groups = append([]string{text}, groups...)
	return sign + strings.Join(groups, ",")
}
