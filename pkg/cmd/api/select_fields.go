package api

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	ghAPI "github.com/cli/cli/v2/api"
)

// resourcePattern describes a REST endpoint pattern that can be translated to a GraphQL query.
type resourcePattern struct {
	// regex matches the resolved REST path (after placeholder expansion).
	regex *regexp.Regexp
	// kind identifies the resource type for field validation and query building.
	kind string
	// buildQuery constructs a complete GraphQL query string from extracted path
	// segments and the validated field list.
	buildQuery func(matches []string, fields []string) (query string, variables map[string]interface{}, dataPath []string)
}

var resourcePatterns = []resourcePattern{
	{
		regex: regexp.MustCompile(`^repos/([^/]+)/([^/]+)$`),
		kind:  "Repository",
		buildQuery: func(m []string, fields []string) (string, map[string]interface{}, []string) {
			fragment := ghAPI.RepositoryGraphQL(fields)
			q := fmt.Sprintf(`query ($owner: String!, $name: String!) {
  repository(owner: $owner, name: $name) {%s}
}`, fragment)
			vars := map[string]interface{}{
				"owner": m[1],
				"name":  m[2],
			}
			return q, vars, []string{"repository"}
		},
	},
	{
		regex: regexp.MustCompile(`^repos/([^/]+)/([^/]+)/issues/(\d+)$`),
		kind:  "Issue",
		buildQuery: func(m []string, fields []string) (string, map[string]interface{}, []string) {
			fragment := ghAPI.IssueGraphQL(fields)
			q := fmt.Sprintf(`query ($owner: String!, $repo: String!, $number: Int!) {
  repository(owner: $owner, name: $repo) {
    issue(number: $number) {%s}
  }
}`, fragment)
			num, _ := strconv.Atoi(m[3])
			vars := map[string]interface{}{
				"owner":  m[1],
				"repo":   m[2],
				"number": num,
			}
			return q, vars, []string{"repository", "issue"}
		},
	},
	{
		regex: regexp.MustCompile(`^repos/([^/]+)/([^/]+)/pulls/(\d+)$`),
		kind:  "PullRequest",
		buildQuery: func(m []string, fields []string) (string, map[string]interface{}, []string) {
			fragment := ghAPI.PullRequestGraphQL(fields)
			q := fmt.Sprintf(`query ($owner: String!, $repo: String!, $number: Int!) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $number) {%s}
  }
}`, fragment)
			num, _ := strconv.Atoi(m[3])
			vars := map[string]interface{}{
				"owner":  m[1],
				"repo":   m[2],
				"number": num,
			}
			return q, vars, []string{"repository", "pullRequest"}
		},
	},
}

// allowedFieldsForKind returns the known field names for a resource kind.
func allowedFieldsForKind(kind string) []string {
	switch kind {
	case "Repository":
		return ghAPI.RepositoryFields
	case "Issue":
		return ghAPI.IssueFields
	case "PullRequest":
		return ghAPI.PullRequestFields
	default:
		return nil
	}
}

// matchEndpoint attempts to match a resolved REST path against known patterns.
// It returns the matched pattern and regex groups, or nil if no match.
func matchEndpoint(path string) (*resourcePattern, []string) {
	path = strings.TrimPrefix(path, "/")
	// Remove query string for matching purposes.
	if idx := strings.IndexByte(path, '?'); idx >= 0 {
		path = path[:idx]
	}
	for i := range resourcePatterns {
		if m := resourcePatterns[i].regex.FindStringSubmatch(path); m != nil {
			return &resourcePatterns[i], m
		}
	}
	return nil, nil
}

// buildSelectQuery validates the requested fields and builds a GraphQL query
// that fetches only those fields from the server.
func buildSelectQuery(path string, selectFields []string) (query string, variables map[string]interface{}, dataPath []string, err error) {
	pat, matches := matchEndpoint(path)
	if pat == nil {
		supported := []string{
			"repos/{owner}/{repo}",
			"repos/{owner}/{repo}/issues/{number}",
			"repos/{owner}/{repo}/pulls/{number}",
		}
		err = fmt.Errorf(
			"--select is not supported for endpoint %q\nSupported patterns:\n  %s",
			path, strings.Join(supported, "\n  "),
		)
		return
	}

	allowed := allowedFieldsForKind(pat.kind)
	allowedSet := make(map[string]bool, len(allowed))
	for _, f := range allowed {
		allowedSet[f] = true
	}
	for _, f := range selectFields {
		if !allowedSet[f] {
			err = fmt.Errorf(
				"unknown %s field: %q\nAvailable fields:\n  %s",
				pat.kind, f, strings.Join(allowed, "\n  "),
			)
			return
		}
	}

	query, variables, dataPath = pat.buildQuery(matches, selectFields)
	return
}

// unwrapGraphQLData extracts the inner resource object from a GraphQL response
// by traversing the dataPath (e.g., ["repository"] or ["repository","issue"]).
func unwrapGraphQLData(raw []byte, dataPath []string) ([]byte, error) {
	var envelope struct {
		Data   json.RawMessage `json:"data"`
		Errors json.RawMessage `json:"errors"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse GraphQL response: %w", err)
	}
	if len(envelope.Errors) > 0 && string(envelope.Errors) != "null" {
		return raw, nil
	}

	current := []byte(envelope.Data)
	for _, key := range dataPath {
		var m map[string]json.RawMessage
		if err := json.Unmarshal(current, &m); err != nil {
			return nil, fmt.Errorf("failed to unwrap GraphQL response at %q: %w", key, err)
		}
		val, ok := m[key]
		if !ok {
			return nil, fmt.Errorf("GraphQL response missing expected key %q", key)
		}
		current = val
	}
	return current, nil
}
