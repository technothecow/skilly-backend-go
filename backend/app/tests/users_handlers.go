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

	resp, err := httpClient.Post(Url + "/register", "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)

	if resp.StatusCode == http.StatusCreated {
		assert.Regexp(t, AuthCookieRegexp, resp.Header.Get("Set-Cookie"))
	}

	return resp
}

func CheckUsernameAvailability(t *testing.T, httpClient *http.Client, username string) bool {
	resp, err := httpClient.Get(Url + "/check-username?username=" + username)
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

	resp, err := httpClient.Post(Url + "/login", "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)

	if resp.StatusCode == http.StatusOK {
		assert.Regexp(t, AuthCookieRegexp, resp.Header.Get("Set-Cookie"))
	}

	return resp
}

func ViewUserProfile(t *testing.T, httpClient *http.Client, username string) *http.Response {
	resp, err := httpClient.Post(Url + "/profile/view?username="+username, "none", bytes.NewBuffer([]byte{}))
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

	resp, err := httpClient.Post(Url + "/profile/edit", "application/json", bytes.NewBuffer(body))
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

	resp, err := httpClient.Post(Url + "/search", "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	
	return resp
}

func SetProfilePicture(t *testing.T, httpClient *http.Client, blob []byte) *http.Response {
	resp, err := httpClient.Post(Url + "/profile/set_picture", "application/json", bytes.NewBuffer([]byte{}))
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	respBody := ParseBody(t, resp)
	defer resp.Body.Close()

	url := respBody["url"].(string)
	assert.NotEmpty(t, url)

	request, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(blob))
	assert.NoError(t, err)

	resp, err = httpClient.Do(request)
	assert.NoError(t, err)

	return resp
}

func GetProfilePicture(t *testing.T, httpClient *http.Client, username string) *http.Response {
	resp, err := httpClient.Get(Url + "/profile/get_picture?username=" + username)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	defer resp.Body.Close()
	respBody := ParseBody(t, resp)

	resp, err = httpClient.Get(respBody["url"].(string))
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	return resp
}