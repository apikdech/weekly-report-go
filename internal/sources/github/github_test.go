package github_test

import (
	"testing"

	"github.com/apikdech/gws-weekly-report/internal/pipeline"
	gh "github.com/apikdech/gws-weekly-report/internal/sources/github"
)

func TestCleanPRTitle(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"Add feature…", "Add feature"},
		{"  Fix bug  ", "Fix bug"},
		{"Normal title", "Normal title"},
		{"Title with … ellipsis", "Title with  ellipsis"},
	}
	for _, tc := range cases {
		got := gh.CleanTitle(tc.input)
		if got != tc.want {
			t.Errorf("CleanTitle(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestGroupByRepo_MergesImplementedAndReviewed(t *testing.T) {
	implemented := []pipeline.PR{
		{Title: "Add X", URL: "https://github.com/org/a/pull/1"},
	}
	reviewed := []pipeline.PR{
		{Title: "Fix Y", URL: "https://github.com/org/b/pull/2"},
	}
	repoImpl := map[string][]pipeline.PR{"org/a": implemented}
	repoReviewed := map[string][]pipeline.PR{"org/b": reviewed}

	result := gh.GroupByRepo(repoImpl, repoReviewed)

	if len(result) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(result))
	}
	if len(result["org/a"].Implemented) != 1 {
		t.Errorf("expected 1 implemented PR for org/a")
	}
	if len(result["org/b"].Reviewed) != 1 {
		t.Errorf("expected 1 reviewed PR for org/b")
	}
}
