package report

import (
	"bytes"
	"embed"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/apikdech/gws-weekly-report/internal/pipeline"
	"github.com/apikdech/gws-weekly-report/internal/sources/hackernews"
)

//go:embed report.tmpl
var reportTemplateFS embed.FS

var reportTmpl = mustParseReportTemplate()

func mustParseReportTemplate() *template.Template {
	t := template.New("report.tmpl").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"formatTechHighlightBody": formatTechHighlightBody,
	})
	t, err := t.ParseFS(reportTemplateFS, "report.tmpl")
	if err != nil {
		panic("report: parse embedded report.tmpl: " + err.Error())
	}
	return t
}

type templateData struct {
	ReportName           string
	Week                 pipeline.WeekRange
	SortedRepos          []*pipeline.RepoPRs
	Events               []pipeline.CalendarEvent
	OutOfOfficeBlock     string
	KeyMetrics           string
	NextActionLines      []string // "1. ...", "2. ..."
	TechnologyHighlights []hackernews.TechHighlight
}

// Render produces the weekly report markdown string from ReportData.
func Render(data *pipeline.ReportData) (string, error) {
	// Sort repos alphabetically for deterministic output.
	repos := make([]*pipeline.RepoPRs, 0, len(data.PRsByRepo))
	for _, r := range data.PRsByRepo {
		repos = append(repos, r)
	}
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].RepoName < repos[j].RepoName
	})

	var oooBlock string
	if n := len(data.OutOfOfficeDates); n > 0 {
		lines := make([]string, n)
		for i, d := range data.OutOfOfficeDates {
			lines[i] = fmt.Sprintf("%d. %s", i+1, d)
		}
		oooBlock = strings.Join(lines, "\n")
	}

	nextLines := make([]string, len(data.NextActions))
	for i, a := range data.NextActions {
		nextLines[i] = fmt.Sprintf("%d. %s", i+1, strings.TrimSpace(a))
	}

	td := templateData{
		ReportName:           data.ReportName,
		Week:                 data.Week,
		SortedRepos:          repos,
		Events:               data.Events,
		OutOfOfficeBlock:     oooBlock,
		KeyMetrics:           formatKeyMetricsForMarkdown(data.KeyMetrics),
		NextActionLines:      nextLines,
		TechnologyHighlights: data.TechnologyHighlights,
	}

	var buf bytes.Buffer
	if err := reportTmpl.Execute(&buf, td); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}

// formatKeyMetricsForMarkdown adjusts raw Google Chat text so Markdown→Docs
// conversion keeps structure: single newlines inside a block become hard line
// breaks, blank lines stay paragraph breaks, and leading spaces become NBSP so
// indentation is not stripped.
func formatKeyMetricsForMarkdown(raw string) string {
	raw = strings.ReplaceAll(strings.ReplaceAll(raw, "\r\n", "\n"), "\r", "\n")
	if raw == "" {
		return ""
	}
	lines := strings.Split(raw, "\n")
	var paras [][]string
	var cur []string
	flush := func() {
		if len(cur) > 0 {
			paras = append(paras, cur)
			cur = nil
		}
	}
	for _, ln := range lines {
		if ln == "" {
			flush()
		} else {
			cur = append(cur, ln)
		}
	}
	flush()

	var b strings.Builder
	for pi, para := range paras {
		if pi > 0 {
			b.WriteString("\n\n")
		}
		for li, line := range para {
			if li > 0 {
				b.WriteString("  \n")
			}
			b.WriteString(leadingSpacesToNbsp(line))
		}
	}
	return b.String()
}

func leadingSpacesToNbsp(s string) string {
	n := 0
	for n < len(s) && s[n] == ' ' {
		n++
	}
	if n == 0 {
		return s
	}
	return strings.Repeat("\u00a0", n) + s[n:]
}

// formatTechHighlightBody indents highlight text so Markdown parsers (including
// Google Docs import) keep the summary and bullets inside the same ordered-list
// item. Unindented lines after "N. [title](url)" close the list and break numbering.
func formatTechHighlightBody(raw string) string {
	raw = strings.ReplaceAll(strings.ReplaceAll(raw, "\r\n", "\n"), "\r", "\n")
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	const indent = "   " // content column under "N. " for list continuation
	lines := strings.Split(raw, "\n")
	var b strings.Builder
	var wroteText bool
	var wroteBlankBeforeBullets bool
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		md := techHighlightLineToMarkdown(trimmed)
		isBullet := strings.HasPrefix(md, "- ")
		if isBullet && wroteText && !wroteBlankBeforeBullets {
			b.WriteString(indent)
			b.WriteByte('\n')
			wroteBlankBeforeBullets = true
		}
		if !isBullet {
			wroteText = true
		}
		b.WriteString(indent)
		b.WriteString(md)
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

func techHighlightLineToMarkdown(s string) string {
	switch {
	case strings.HasPrefix(s, "• "):
		return "- " + s[len("• "):]
	case strings.HasPrefix(s, "•"):
		return "- " + strings.TrimSpace(s[len("•"):])
	case strings.HasPrefix(s, "- "):
		return s
	case strings.HasPrefix(s, "* "):
		return "- " + s[2:]
	default:
		return s
	}
}
