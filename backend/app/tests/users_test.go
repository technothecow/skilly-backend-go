package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

const AuthCookieRegexp = "authToken=([^;]+);\\s*Path=/;\\s*Max-Age=\\d+;\\s*HttpOnly;\\s*SameSite=Strict$"

func TestUsernameAvailability(t *testing.T) {
	ctx := context.Background()
	close, err := TestUsingDockerCompose(ctx, t)
	assert.NoError(t, err)
	defer close()

	httpClient := &http.Client{}
	resp, err := httpClient.Get("http://localhost:8000/check-username?username=test")
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

	var response map[string]any
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)

	assert.Equal(t, true, response["available"])
}

func TestUserFlow(t *testing.T) {
	ctx := context.Background()
	close, err := TestUsingDockerCompose(ctx, t)
	assert.NoError(t, err)
	defer close()

	body, err := json.Marshal(
		map[string]any{
			"username": "test",
			"password": "test",
			"bio":      "test",
			"teaching": []string{"test"},
			"learning": []string{"test"},
		},
	)
	assert.NoError(t, err)

	httpClient := &http.Client{}
	resp, err := httpClient.Post("http://localhost:8000/register", "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Regexp(t, AuthCookieRegexp, resp.Header.Get("Set-Cookie"))

	resp, err = httpClient.Get("http://localhost:8000/check-username?username=test")
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]any
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)

	assert.Equal(t, false, response["available"])

	resp, err = httpClient.Post("http://localhost:8000/register", "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "username_already_exists", response["code"])

	body, err = json.Marshal(
		map[string]any{
			"username": "test",
			"password": "test",
		},
	)
	assert.NoError(t, err)

	resp, err = httpClient.Post("http://localhost:8000/login", "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Regexp(t, AuthCookieRegexp, resp.Header.Get("Set-Cookie"))

	body, err = json.Marshal(
		map[string]any{
			"username": "test",
			"password": "wrong",
		},
	)
	assert.NoError(t, err)

	resp, err = httpClient.Post("http://localhost:8000/login", "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "invalid_credentials", response["code"])

	resp, err = httpClient.Post("http://localhost:8000/logout", "none", bytes.NewBuffer([]byte{}))
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.Equal(t, "authToken=; Path=/; Max-Age=0; HttpOnly; SameSite=Strict", resp.Header.Get("Set-Cookie"))

	resp, err = httpClient.Post("http://localhost:8000/profile/view?username=test", "none", bytes.NewBuffer([]byte{}))
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "test", response["username"])
	assert.Equal(t, "", response["pictureUrl"])
	assert.Equal(t, "test", response["bio"])
	assert.Equal(t, []interface{}{"test"}, response["teaching"].([]interface{}))
	assert.Equal(t, []interface{}{"test"}, response["learning"].([]interface{}))

	resp, err = httpClient.Post("http://localhost:8000/profile/view?username=wrong", "none", bytes.NewBuffer([]byte{}))
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "username_not_found", response["code"])
}
