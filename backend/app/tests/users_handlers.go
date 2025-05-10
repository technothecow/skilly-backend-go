package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func RegisterUser(t *testing.T, httpClient *http.Client, username string, password string, bio string, teaching []string, learning []string) *http.Response {
	body := MarshalBody(t, map[string]any{
		"username": username,
		"password": password,
		"bio":      bio,
		"teaching": teaching,
		"learning": learning,
	})

	resp, err := httpClient.Post("http://localhost:8000/register", "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)

	if resp.StatusCode == http.StatusCreated {
		assert.Regexp(t, AuthCookieRegexp, resp.Header.Get("Set-Cookie"))
	}

	return resp
}

func CheckUsernameAvailability(t *testing.T, httpClient *http.Client, username string) bool {
	resp, err := httpClient.Get("http://localhost:8000/check-username?username=" + username)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	response := ParseBody(t, resp)
	return response["available"].(bool)
}

func LoginUser(t *testing.T, httpClient *http.Client, username string, password string) *http.Response {
	body := MarshalBody(t, map[string]any{
		"username": username,
		"password": password,
	})

	resp, err := httpClient.Post("http://localhost:8000/login", "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)

	if resp.StatusCode == http.StatusOK {
		assert.Regexp(t, AuthCookieRegexp, resp.Header.Get("Set-Cookie"))
	}

	return resp
}

func ViewUserProfile(t *testing.T, httpClient *http.Client, username string) *http.Response {
	resp, err := httpClient.Post("http://localhost:8000/profile/view?username="+username, "none", bytes.NewBuffer([]byte{}))
	assert.NoError(t, err)

	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	return resp
}

func EditUserProfile(t *testing.T, httpClient *http.Client, password string, bio string, teaching []string, learning []string) *http.Response {
	bodyRaw := map[string]any{}

	if len(password) > 0 {
		bodyRaw["password"] = password
	}
	if len(bio) > 0 {
		bodyRaw["bio"] = bio
	}
	if len(teaching) > 0 {
		bodyRaw["teaching"] = teaching
	}
	if len(learning) > 0 {
		bodyRaw["learning"] = learning
	}

	body, err := json.Marshal(bodyRaw)
	assert.NoError(t, err)

	resp, err := httpClient.Post("http://localhost:8000/profile/edit", "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)

	return resp
}

func SearchUsers(t *testing.T, httpClient *http.Client, username string, skills []string, page int, pagesize int) *http.Response {
	bodyRaw := map[string]any{}

	if len(username) > 0 {
		bodyRaw["username"] = username
	}
	if len(skills) > 0 {
		bodyRaw["skills"] = skills
	}
	if page > -1 {
		bodyRaw["page"] = page
	}
	if pagesize > -1 {
		bodyRaw["pagesize"] = pagesize
	}

	body := MarshalBody(t, bodyRaw)

	resp, err := httpClient.Post("http://localhost:8000/search", "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	
	return resp
}
