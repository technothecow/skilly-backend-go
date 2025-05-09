package tests

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func ParseBody(t *testing.T, resp *http.Response) map[string]any {
	var response map[string]any
	err := json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	return response
}

func MarshalBody(t *testing.T, obj any) []byte {
	body, err := json.Marshal(obj)
	assert.NoError(t, err)
	return body
}
