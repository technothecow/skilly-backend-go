package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/url"
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

	response := ParseBody(t, resp)
	assert.Equal(t, true, response["available"])
}

func TestUserFlow(t *testing.T) {
	// setup
	ctx := context.Background()
	close, err := TestUsingDockerCompose(ctx, t)
	assert.NoError(t, err)
	defer close()

	httpClient := &http.Client{}

	t.Run("register", func(t *testing.T) {
		body := MarshalBody(t, map[string]any{
			"username": "test",
			"password": "test",
			"bio":      "test",
			"teaching": []string{"test"},
			"learning": []string{"test"},
		})

		resp, err := httpClient.Post("http://localhost:8000/register", "application/json", bytes.NewBuffer(body))
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		assert.Regexp(t, AuthCookieRegexp, resp.Header.Get("Set-Cookie"))
	})

	t.Run("check-username", func(t *testing.T) {
		resp, err := httpClient.Get("http://localhost:8000/check-username?username=test")
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		response := ParseBody(t, resp)

		assert.Equal(t, false, response["available"])
	})

	t.Run("register-existing-username", func(t *testing.T) {
		body := MarshalBody(t, map[string]any{
			"username": "test",
			"password": "test",
			"bio":      "test",
			"teaching": []string{"test"},
			"learning": []string{"test"},
		})

		resp, err := httpClient.Post("http://localhost:8000/register", "application/json", bytes.NewBuffer(body))
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)
		assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

		response := ParseBody(t, resp)

		assert.Equal(t, "username_already_exists", response["code"])
	})

	t.Run("login", func(t *testing.T) {
		body := MarshalBody(t, map[string]any{
			"username": "test",
			"password": "test",
		})

		resp, err := httpClient.Post("http://localhost:8000/login", "application/json", bytes.NewBuffer(body))
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Regexp(t, AuthCookieRegexp, resp.Header.Get("Set-Cookie"))
	})

	t.Run("login-wrong-password", func(t *testing.T) {
		body := MarshalBody(t, map[string]any{
			"username": "test",
			"password": "wrong",
		})

		resp, err := httpClient.Post("http://localhost:8000/login", "application/json", bytes.NewBuffer(body))
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

		response := ParseBody(t, resp)

		assert.Equal(t, "invalid_credentials", response["code"])
	})

	t.Run("logout", func(t *testing.T) {
		resp, err := httpClient.Post("http://localhost:8000/logout", "none", bytes.NewBuffer([]byte{}))
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		assert.Equal(t, "authToken=; Path=/; Max-Age=0; HttpOnly; SameSite=Strict", resp.Header.Get("Set-Cookie"))
	})

	t.Run("profile-view", func(t *testing.T) {
		resp, err := httpClient.Post("http://localhost:8000/profile/view?username=test", "none", bytes.NewBuffer([]byte{}))
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

		response := ParseBody(t, resp)

		assert.Equal(t, "test", response["username"])
		assert.Equal(t, "", response["pictureUrl"])
		assert.Equal(t, "test", response["bio"])
		assert.Equal(t, []interface{}{"test"}, response["teaching"].([]interface{}))
		assert.Equal(t, []interface{}{"test"}, response["learning"].([]interface{}))
	})

	t.Run("profile-view-nonexistent-username", func(t *testing.T) {
		resp, err := httpClient.Post("http://localhost:8000/profile/view?username=wrong", "none", bytes.NewBuffer([]byte{}))
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

		response := ParseBody(t, resp)

		assert.Equal(t, "username_not_found", response["code"])
	})

	t.Run("profile-edit", func(t *testing.T) {
		// logging in
		body := MarshalBody(t, map[string]any{
			"username": "test",
			"password": "test",
		})

		resp, err := httpClient.Post("http://localhost:8000/login", "application/json", bytes.NewBuffer(body))
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Regexp(t, AuthCookieRegexp, resp.Header.Get("Set-Cookie"))
		
		// setting cookie
		cookies := resp.Cookies()
		url, err := url.Parse("http://localhost:8000/profile/edit")
		assert.NoError(t, err)
		jar, err := cookiejar.New(nil)
		assert.NoError(t, err)
		jar.SetCookies(url, cookies)
		httpClient.Jar = jar
		defer httpClient.Jar.SetCookies(url, nil)

		// editing profile
		body, err = json.Marshal(
			map[string]any{
				"password": "new",
				"bio":      "new",
				"teaching": []string{"new"},
				"learning": []string{"new"},
			},
		)
		assert.NoError(t, err)

		resp, err = httpClient.Post("http://localhost:8000/profile/edit", "application/json", bytes.NewBuffer(body))
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

		response := ParseBody(t, resp)

		assert.Equal(t, "test", response["username"])
		assert.Equal(t, "", response["pictureUrl"])
		assert.Equal(t, "new", response["bio"])
		assert.Equal(t, []interface{}{"new"}, response["teaching"].([]interface{}))
		assert.Equal(t, []interface{}{"new"}, response["learning"].([]interface{}))

		// checking if the profile was updated by viewing it
		resp, err = httpClient.Post("http://localhost:8000/profile/view?username=test", "none", bytes.NewBuffer([]byte{}))
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

		response = ParseBody(t, resp)

		assert.Equal(t, "test", response["username"])
		assert.Equal(t, "", response["pictureUrl"])
		assert.Equal(t, "new", response["bio"])
		assert.Equal(t, []interface{}{"new"}, response["teaching"].([]interface{}))
		assert.Equal(t, []interface{}{"new"}, response["learning"].([]interface{}))

		// checking if the profile was updated by attempting to login to it with the old password
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

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		// checking if the profile was updated by attempting to login to it with the new password
		body, err = json.Marshal(
			map[string]any{
				"username": "test",
				"password": "new",
			},
		)
		assert.NoError(t, err)

		resp, err = httpClient.Post("http://localhost:8000/login", "application/json", bytes.NewBuffer(body))
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Regexp(t, AuthCookieRegexp, resp.Header.Get("Set-Cookie"))
	})
}
