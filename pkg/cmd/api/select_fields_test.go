package api

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_matchEndpoint_repository(t *testing.T) {
	pat, m := matchEndpoint("repos/cli/cli")
	require.NotNil(t, pat)
	assert.Equal(t, "Repository", pat.kind)
	assert.Equal(t, "cli", m[1])
	assert.Equal(t, "cli", m[2])
}

func Test_matchEndpoint_repositoryWithSlash(t *testing.T) {
	pat, m := matchEndpoint("/repos/cli/cli")
	require.NotNil(t, pat)
	assert.Equal(t, "Repository", pat.kind)
	assert.Equal(t, "cli", m[1])
	assert.Equal(t, "cli", m[2])
}

func Test_matchEndpoint_issue(t *testing.T) {
	pat, m := matchEndpoint("repos/cli/cli/issues/42")
	require.NotNil(t, pat)
	assert.Equal(t, "Issue", pat.kind)
	assert.Equal(t, "cli", m[1])
	assert.Equal(t, "cli", m[2])
	assert.Equal(t, "42", m[3])
}

func Test_matchEndpoint_pullRequest(t *testing.T) {
	pat, m := matchEndpoint("repos/cli/cli/pulls/123")
	require.NotNil(t, pat)
	assert.Equal(t, "PullRequest", pat.kind)
	assert.Equal(t, "123", m[3])
}

func Test_matchEndpoint_noMatch(t *testing.T) {
	tests := []string{
		"repos/cli/cli/commits",
		"repos/cli/cli/issues",
		"users/octocat",
		"orgs/github",
		"graphql",
	}
	for _, path := range tests {
		pat, _ := matchEndpoint(path)
		assert.Nil(t, pat, "expected no match for %q", path)
	}
}

func Test_matchEndpoint_withQueryString(t *testing.T) {
	pat, m := matchEndpoint("repos/cli/cli?per_page=10")
	require.NotNil(t, pat)
	assert.Equal(t, "Repository", pat.kind)
	assert.Equal(t, "cli", m[1])
}

func Test_buildSelectQuery_repository(t *testing.T) {
	query, vars, dataPath, err := buildSelectQuery("repos/cli/cli", []string{"name", "defaultBranchRef"})
	require.NoError(t, err)
	assert.Contains(t, query, "repository(owner: $owner, name: $name)")
	assert.Contains(t, query, "name")
	assert.Contains(t, query, "defaultBranchRef")
	assert.Equal(t, "cli", vars["owner"])
	assert.Equal(t, "cli", vars["name"])
	assert.Equal(t, []string{"repository"}, dataPath)
}

func Test_buildSelectQuery_issue(t *testing.T) {
	query, vars, dataPath, err := buildSelectQuery("repos/cli/cli/issues/42", []string{"title", "state"})
	require.NoError(t, err)
	assert.Contains(t, query, "issue(number: $number)")
	assert.Contains(t, query, "title")
	assert.Equal(t, 42, vars["number"])
	assert.Equal(t, []string{"repository", "issue"}, dataPath)
}

func Test_buildSelectQuery_pullRequest(t *testing.T) {
	query, vars, dataPath, err := buildSelectQuery("repos/cli/cli/pulls/7", []string{"title", "isDraft"})
	require.NoError(t, err)
	assert.Contains(t, query, "pullRequest(number: $number)")
	assert.Equal(t, 7, vars["number"])
	assert.Equal(t, []string{"repository", "pullRequest"}, dataPath)
}

func Test_buildSelectQuery_unsupportedEndpoint(t *testing.T) {
	_, _, _, err := buildSelectQuery("users/octocat", []string{"login"})
	assert.ErrorContains(t, err, "--select is not supported for endpoint")
	assert.ErrorContains(t, err, "Supported patterns:")
}

func Test_buildSelectQuery_unknownField(t *testing.T) {
	_, _, _, err := buildSelectQuery("repos/cli/cli", []string{"name", "nonexistentField"})
	assert.ErrorContains(t, err, `unknown Repository field: "nonexistentField"`)
	assert.ErrorContains(t, err, "Available fields:")
}

func Test_unwrapGraphQLData_repository(t *testing.T) {
	resp := `{"data":{"repository":{"name":"cli","defaultBranchRef":{"name":"trunk"}}}}`
	result, err := unwrapGraphQLData([]byte(resp), []string{"repository"})
	require.NoError(t, err)

	var obj map[string]interface{}
	require.NoError(t, json.Unmarshal(result, &obj))
	assert.Equal(t, "cli", obj["name"])
}

func Test_unwrapGraphQLData_issue(t *testing.T) {
	resp := `{"data":{"repository":{"issue":{"title":"Bug","state":"OPEN"}}}}`
	result, err := unwrapGraphQLData([]byte(resp), []string{"repository", "issue"})
	require.NoError(t, err)

	var obj map[string]interface{}
	require.NoError(t, json.Unmarshal(result, &obj))
	assert.Equal(t, "Bug", obj["title"])
	assert.Equal(t, "OPEN", obj["state"])
}

func Test_unwrapGraphQLData_withErrors(t *testing.T) {
	resp := `{"data":null,"errors":[{"message":"Not Found"}]}`
	result, err := unwrapGraphQLData([]byte(resp), []string{"repository"})
	require.NoError(t, err)
	assert.Contains(t, string(result), "Not Found")
}

func Test_unwrapGraphQLData_missingKey(t *testing.T) {
	resp := `{"data":{"repository":null}}`
	_, err := unwrapGraphQLData([]byte(resp), []string{"repository", "issue"})
	assert.Error(t, err)
}
