package matcher

import (
	"testing"
)

var testSlugs = []string{
	"cogover-subscription-app",
	"api.stringeex.com",
	"cogover-web-admin",
	"cogover-api-gateway",
	"stringeex-dashboard",
}

func TestSingleTermMatch(t *testing.T) {
	result := Match(testSlugs, []string{"subscription"})
	if len(result.Matched) != 1 || result.Matched[0] != "cogover-subscription-app" {
		t.Errorf("expected [cogover-subscription-app], got %v", result.Matched)
	}
	if len(result.Unmatched) != 0 {
		t.Errorf("expected no unmatched, got %v", result.Unmatched)
	}
}

func TestMultiTermAND(t *testing.T) {
	result := Match(testSlugs, []string{"api stringeex"})
	if len(result.Matched) != 1 || result.Matched[0] != "api.stringeex.com" {
		t.Errorf("expected [api.stringeex.com], got %v", result.Matched)
	}
}

func TestNoMatch(t *testing.T) {
	result := Match(testSlugs, []string{"nonexistent"})
	if len(result.Matched) != 0 {
		t.Errorf("expected no matches, got %v", result.Matched)
	}
	if len(result.Unmatched) != 1 || result.Unmatched[0] != "nonexistent" {
		t.Errorf("expected [nonexistent] unmatched, got %v", result.Unmatched)
	}
}

func TestExactMatch(t *testing.T) {
	result := Match(testSlugs, []string{"cogover-web-admin"})
	if len(result.Matched) != 1 || result.Matched[0] != "cogover-web-admin" {
		t.Errorf("expected [cogover-web-admin], got %v", result.Matched)
	}
}

func TestDeduplication(t *testing.T) {
	// Both patterns match the same repo
	result := Match(testSlugs, []string{"subscription", "cogover-subscription"})
	if len(result.Matched) != 1 {
		t.Errorf("expected 1 deduplicated match, got %v", result.Matched)
	}
}

func TestCaseInsensitive(t *testing.T) {
	result := Match(testSlugs, []string{"SUBSCRIPTION"})
	if len(result.Matched) != 1 || result.Matched[0] != "cogover-subscription-app" {
		t.Errorf("expected case-insensitive match, got %v", result.Matched)
	}
}

func TestMultiplePatterns(t *testing.T) {
	result := Match(testSlugs, []string{"subscription", "dashboard"})
	if len(result.Matched) != 2 {
		t.Errorf("expected 2 matches, got %v", result.Matched)
	}
}

func TestEmptyPatterns(t *testing.T) {
	result := Match(testSlugs, []string{})
	if len(result.Matched) != 0 {
		t.Errorf("expected no matches, got %v", result.Matched)
	}
}

func TestEmptySlugs(t *testing.T) {
	result := Match([]string{}, []string{"something"})
	if len(result.Matched) != 0 {
		t.Errorf("expected no matches, got %v", result.Matched)
	}
	if len(result.Unmatched) != 1 {
		t.Errorf("expected 1 unmatched, got %v", result.Unmatched)
	}
}

func TestWhitespacePattern(t *testing.T) {
	result := Match(testSlugs, []string{"  ", ""})
	if len(result.Matched) != 0 {
		t.Errorf("expected no matches for whitespace patterns, got %v", result.Matched)
	}
}

func TestPatternMatchesMultipleRepos(t *testing.T) {
	result := Match(testSlugs, []string{"cogover"})
	if len(result.Matched) != 3 {
		t.Errorf("expected 3 repos matching 'cogover', got %v", result.Matched)
	}
}
