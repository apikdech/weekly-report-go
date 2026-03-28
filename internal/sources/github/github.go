package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"

	"github.com/apikdech/gws-weekly-report/internal/pipeline"
)

// CleanTitle removes ellipsis characters and trims whitespace from a PR title.
func CleanTitle(title string) string {
	title = strings.ReplaceAll(title, "…", "")
	return strings.TrimSpace(title)
}

// GroupByRepo merges implemented and reviewed PR maps into a map of RepoPRs.
func GroupByRepo(implemented, reviewed map[string][]pipeline.PR) map[string]*pipeline.RepoPRs {
	result := make(map[string]*pipeline.RepoPRs)

	for repo, prs := range implemented {
		if _, ok := result[repo]; !ok {
			result[repo] = &pipeline.RepoPRs{RepoName: repo}
		}
		result[repo].Implemented = append(result[repo].Implemented, prs...)
	}
	for repo, prs := range reviewed {
		if _, ok := result[repo]; !ok {
			result[repo] = &pipeline.RepoPRs{RepoName: repo}
		}
		result[repo].Reviewed = append(result[repo].Reviewed, prs...)
	}
	return result
}

// Source fetches GitHub PRs authored and reviewed by the user for the week.
type Source struct {
	token     string
	username  string
	prsByRepo map[string]*pipeline.RepoPRs
}

// NewSource creates a GitHubSource.
func NewSource(token, username string) *Source {
	return &Source{token: token, username: username}
}

// Name implements DataSource.
func (s *Source) Name() string { return "github" }

// Fetch queries GitHub GraphQL API for authored and reviewed PRs within the week range.
func (s *Source) Fetch(ctx context.Context, week pipeline.WeekRange) error {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: s.token})
	tc := oauth2.NewClient(ctx, ts)
	client := githubv4.NewClient(tc)

	implemented, err := s.fetchPRs(ctx, client,
		fmt.Sprintf("author:%s is:pr created:%s..%s",
			s.username,
			week.Start.Format("2006-01-02T15:04:05Z"),
			week.End.Format("2006-01-02T15:04:05Z"),
		),
	)
	if err != nil {
		return fmt.Errorf("fetch implemented PRs: %w", err)
	}

	reviewed, err := s.fetchPRs(ctx, client,
		fmt.Sprintf("reviewed-by:%s is:pr created:%s..%s",
			s.username,
			week.Start.Format("2006-01-02T15:04:05Z"),
			week.End.Format("2006-01-02T15:04:05Z"),
		),
	)
	if err != nil {
		return fmt.Errorf("fetch reviewed PRs: %w", err)
	}

	s.prsByRepo = GroupByRepo(implemented, reviewed)
	return nil
}

func (s *Source) fetchPRs(ctx context.Context, client *githubv4.Client, searchQuery string) (map[string][]pipeline.PR, error) {
	var query struct {
		Search struct {
			PageInfo struct {
				EndCursor   githubv4.String
				HasNextPage bool
			}
			Nodes []struct {
				PullRequest struct {
					Title      githubv4.String
					URL        githubv4.URI
					Repository struct {
						NameWithOwner githubv4.String
					}
				} `graphql:"... on PullRequest"`
			}
		} `graphql:"search(query: $query, type: ISSUE, first: 100, after: $cursor)"`
	}

	variables := map[string]interface{}{
		"query":  githubv4.String(searchQuery),
		"cursor": (*githubv4.String)(nil),
	}

	result := make(map[string][]pipeline.PR)
	for {
		if err := client.Query(ctx, &query, variables); err != nil {
			return nil, fmt.Errorf("graphql query: %w", err)
		}
		for _, node := range query.Search.Nodes {
			pr := node.PullRequest
			repo := string(pr.Repository.NameWithOwner)
			if repo == "" {
				continue
			}
			result[repo] = append(result[repo], pipeline.PR{
				Title: CleanTitle(string(pr.Title)),
				URL:   pr.URL.String(),
			})
		}
		if !query.Search.PageInfo.HasNextPage {
			break
		}
		variables["cursor"] = githubv4.NewString(query.Search.PageInfo.EndCursor)
	}
	return result, nil
}

// Contribute sets PRsByRepo on the report.
func (s *Source) Contribute(report *pipeline.ReportData) error {
	if report.PRsByRepo == nil {
		report.PRsByRepo = make(map[string]*pipeline.RepoPRs)
	}
	for k, v := range s.prsByRepo {
		report.PRsByRepo[k] = v
	}
	return nil
}
