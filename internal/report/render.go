package report

import (
	"bytes"
	"fmt"
	"sort"
	"text/template"

	"github.com/apikdech/gws-weekly-report/internal/pipeline"
)

const reportTemplate = `# [Weekly Report: {{ .ReportName }}] {{ .Week.HeaderLabel }}

## **Issues**

## **Accomplishment**
{{ range .SortedRepos -}}
### {{ .RepoName }}
#### Implemented PR
{{ range .Implemented -}}
1. [{{ .Title }}]({{ .URL }})
{{ end }}
#### Reviewed PR
{{ range .Reviewed -}}
1. [{{ .Title }}]({{ .URL }})
{{ end }}
{{ end }}
## **Meetings/Events/Training/Conferences**
{{ range .Events -}}
- {{ .Title }} ({{ .Date }})
{{ end }}
## **Key Metrics / OMTM**

## **Next Actions**
1. Continue implement admin dashboard features

## **Technology, Business, Communication, Leadership, Management & Marketing**

## Out of Office
`

type templateData struct {
	ReportName  string
	Week        pipeline.WeekRange
	SortedRepos []*pipeline.RepoPRs
	Events      []pipeline.CalendarEvent
}

// Render produces the weekly report markdown string from ReportData.
func Render(data *pipeline.ReportData) (string, error) {
	tmpl, err := template.New("report").Parse(reportTemplate)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	// Sort repos alphabetically for deterministic output.
	repos := make([]*pipeline.RepoPRs, 0, len(data.PRsByRepo))
	for _, r := range data.PRsByRepo {
		repos = append(repos, r)
	}
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].RepoName < repos[j].RepoName
	})

	td := templateData{
		ReportName:  data.ReportName,
		Week:        data.Week,
		SortedRepos: repos,
		Events:      data.Events,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, td); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}
