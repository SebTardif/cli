package api

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_parseBatchInput_plainPaths(t *testing.T) {
	input := "repos/cli/cli\nrepos/cli/cli/issues/1\n\nrepos/cli/cli/pulls/2\n"
	requests, err := parseBatchInput(strings.NewReader(input))
	require.NoError(t, err)
	assert.Len(t, requests, 3)
	assert.Equal(t, "GET", requests[0].Method)
	assert.Equal(t, "repos/cli/cli", requests[0].URL)
	assert.Equal(t, "repos/cli/cli/issues/1", requests[1].URL)
	assert.Equal(t, "repos/cli/cli/pulls/2", requests[2].URL)
}

func Test_parseBatchInput_jsonLines(t *testing.T) {
	input := `{"method":"GET","url":"repos/cli/cli"}
{"method":"POST","url":"repos/cli/cli/issues","body":{"title":"test"}}
{"url":"repos/cli/cli/pulls/1"}
`
	requests, err := parseBatchInput(strings.NewReader(input))
	require.NoError(t, err)
	assert.Len(t, requests, 3)

	assert.Equal(t, "GET", requests[0].Method)
	assert.Equal(t, "repos/cli/cli", requests[0].URL)

	assert.Equal(t, "POST", requests[1].Method)
	assert.Equal(t, "repos/cli/cli/issues", requests[1].URL)
	assert.Equal(t, "test", requests[1].Body["title"])

	assert.Equal(t, "GET", requests[2].Method)
	assert.Equal(t, "repos/cli/cli/pulls/1", requests[2].URL)
}

func Test_parseBatchInput_invalidJSON(t *testing.T) {
	input := `{"method":"GET"
`
	_, err := parseBatchInput(strings.NewReader(input))
	assert.ErrorContains(t, err, "line 1: invalid JSON")
}

func Test_parseBatchInput_missingURL(t *testing.T) {
	input := `{"method":"GET"}
`
	_, err := parseBatchInput(strings.NewReader(input))
	assert.ErrorContains(t, err, `line 1: missing "url" field`)
}

func Test_parseBatchInput_empty(t *testing.T) {
	requests, err := parseBatchInput(strings.NewReader(""))
	require.NoError(t, err)
	assert.Nil(t, requests)
}

func Test_parseBatchInput_withHeaders(t *testing.T) {
	input := `{"url":"repos/cli/cli","headers":{"Accept":"application/vnd.github.raw+json"}}
`
	requests, err := parseBatchInput(strings.NewReader(input))
	require.NoError(t, err)
	assert.Len(t, requests, 1)
	assert.Equal(t, "application/vnd.github.raw+json", requests[0].Headers["Accept"])
}

func Test_writeBatchResponse(t *testing.T) {
	var buf bytes.Buffer
	err := writeBatchResponse(&buf, 200, "repos/cli/cli", []byte(`{"full_name":"cli/cli"}`))
	require.NoError(t, err)
	assert.Equal(t, `{"status":200,"path":"repos/cli/cli","body":{"full_name":"cli/cli"}}`+"\n", buf.String())
}

func Test_writeBatchResponse_errorStatus(t *testing.T) {
	var buf bytes.Buffer
	err := writeBatchResponse(&buf, 404, "repos/cli/nope", []byte(`{"message":"Not Found"}`))
	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"status":404`)
	assert.Contains(t, buf.String(), `"Not Found"`)
}
