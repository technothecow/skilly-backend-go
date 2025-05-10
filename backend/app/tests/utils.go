package tests

import (
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/url"
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

func GetUrl(t *testing.T) *url.URL {
	url, err := url.Parse("http://localhost:8000/")
	assert.NoError(t, err)
	return url
}

func SetCookies(t *testing.T, httpClient *http.Client, cookies []*http.Cookie) func() {
	url := GetUrl(t)

	jar, err := cookiejar.New(nil)
	assert.NoError(t, err)

	jar.SetCookies(url, cookies)

	httpClient.Jar = jar

	return func() {
		httpClient.Jar = nil
	}
}
