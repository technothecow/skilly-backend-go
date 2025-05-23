package tests

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const AuthCookieRegexp = "authToken=([^;]+);\\s*Path=/;\\s*Max-Age=\\d+;\\s*HttpOnly;\\s*SameSite=Strict$"

/*
	Utils
*/

func AuthorizeClient(t *testing.T, httpClient *http.Client, username string, password string) (func(), error) {
	resp := LoginUser(t, httpClient, username, password)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	cancel := SetCookies(t, httpClient, resp.Cookies())

	return cancel, nil
}

/*
	Tests
*/

func TestUserFlow(t *testing.T) {
	// setup
	ctx := context.Background()
	close, err := TestUsingDockerCompose(ctx, t)
	assert.NoError(t, err)
	defer close()

	httpClient := &http.Client{}

	t.Run("check-username-available", func(t *testing.T) {
		assert.True(t, CheckUsernameAvailability(t, httpClient, "test"))
	})

	t.Run("register", func(t *testing.T) {
		resp := RegisterUser(t, httpClient, "test", "test", "test", []string{"test"}, []string{"test"})
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("check-taken-username-available", func(t *testing.T) {
		assert.False(t, CheckUsernameAvailability(t, httpClient, "test"))
	})

	t.Run("register-existing-username", func(t *testing.T) {
		resp := RegisterUser(t, httpClient, "test", "test", "test", []string{"test"}, []string{"test"})
		defer resp.Body.Close()
		respBody := ParseBody(t, resp)

		assert.Equal(t, "username_already_exists", respBody["code"])
	})

	t.Run("login", func(t *testing.T) {
		resp := LoginUser(t, httpClient, "test", "test")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Regexp(t, AuthCookieRegexp, resp.Header.Get("Set-Cookie"))
	})

	t.Run("login-wrong-password", func(t *testing.T) {
		resp := LoginUser(t, httpClient, "test", "wrong")
		defer resp.Body.Close()
		respBody := ParseBody(t, resp)

		assert.Equal(t, "invalid_credentials", respBody["code"])
	})

	t.Run("logout", func(t *testing.T) {
		resp, err := httpClient.Post(Url+"/logout", "none", bytes.NewBuffer([]byte{}))
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
		assert.Equal(t, "authToken=; Path=/; Max-Age=0; HttpOnly; SameSite=Strict", resp.Header.Get("Set-Cookie"))
	})

	t.Run("profile-view-unauthorized", func(t *testing.T) {
		resp := ViewUserProfile(t, httpClient, "test")
		defer resp.Body.Close()
		response := ParseBody(t, resp)

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		assert.Equal(t, "unauthorized", response["code"])
	})

	t.Run("profile-view", func(t *testing.T) {
		cancel, err := AuthorizeClient(t, httpClient, "test", "test")
		assert.NoError(t, err)
		defer cancel()

		resp := ViewUserProfile(t, httpClient, "test")
		defer resp.Body.Close()
		respBody := ParseBody(t, resp)

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		assert.Equal(t, "test", respBody["username"])
		assert.Equal(t, "test", respBody["bio"])
		assert.Equal(t, []interface{}{"test"}, respBody["teaching"].([]interface{}))
		assert.Equal(t, []interface{}{"test"}, respBody["learning"].([]interface{}))
	})

	t.Run("profile-view-nonexistent-username", func(t *testing.T) {
		cancel, err := AuthorizeClient(t, httpClient, "test", "test")
		assert.NoError(t, err)
		defer cancel()

		resp := ViewUserProfile(t, httpClient, "wrong")
		defer resp.Body.Close()
		response := ParseBody(t, resp)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		assert.Equal(t, "username_not_found", response["code"])
	})

	t.Run("profile-edit", func(t *testing.T) {
		cancel, err := AuthorizeClient(t, httpClient, "test", "test")
		assert.NoError(t, err)
		defer cancel()

		// editing profile
		resp := EditUserProfile(t, httpClient, "new", "new", []string{"new"}, []string{"new"})
		defer resp.Body.Close()
		respBody := ParseBody(t, resp)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json; charset=utf-8", resp.Header.Get("Content-Type"))

		assert.Equal(t, "test", respBody["username"])
		assert.Equal(t, "new", respBody["bio"])
		assert.Equal(t, []interface{}{"new"}, respBody["teaching"].([]interface{}))
		assert.Equal(t, []interface{}{"new"}, respBody["learning"].([]interface{}))

		// checking if the profile was updated by viewing it
		resp = ViewUserProfile(t, httpClient, "test")
		defer resp.Body.Close()
		respBody = ParseBody(t, resp)

		assert.Equal(t, "test", respBody["username"])
		assert.Equal(t, "new", respBody["bio"])
		assert.Equal(t, []interface{}{"new"}, respBody["teaching"].([]interface{}))
		assert.Equal(t, []interface{}{"new"}, respBody["learning"].([]interface{}))

		// checking if the profile was updated by attempting to login to it with the old password
		resp = LoginUser(t, httpClient, "test", "test")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		// checking if the profile was updated by attempting to login to it with the new password
		resp = LoginUser(t, httpClient, "test", "new")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Regexp(t, AuthCookieRegexp, resp.Header.Get("Set-Cookie"))
	})

	t.Run("search-users", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			resp := RegisterUser(
				t, httpClient, fmt.Sprintf("test%d", i),
				"testpswd", "",
				[]string{"testTeach1", "testTeach2"},
				[]string{"testLearn1", "testLearn2"},
			)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusCreated, resp.StatusCode)
		}

		cancel, err := AuthorizeClient(t, httpClient, "test", "new")
		assert.NoError(t, err)
		defer cancel()

		// no results since no user is learning "new"
		resp := SearchUsers(t, httpClient, "", []string{}, -1, -1)
		defer resp.Body.Close()
		respBody := ParseBody(t, resp)

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		assert.Equal(t, 0, len(respBody["users"].([]interface{})))

		// all users (but not "test") are teaching "testTeach"
		resp = EditUserProfile(t, httpClient, "", "", []string{"testLearn1"}, []string{"testLearn1"})
		defer resp.Body.Close()
		respBody = ParseBody(t, resp)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "test", respBody["username"])

		resp = SearchUsers(t, httpClient, "", []string{}, -1, -1)
		defer resp.Body.Close()
		respBody = ParseBody(t, resp)

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		assert.Equal(t, 3, len(respBody["users"].([]interface{})))

		// nothing since no user is teaching "testTeach"
		resp = SearchUsers(t, httpClient, "", []string{"testTeach"}, -1, -1)
		defer resp.Body.Close()
		respBody = ParseBody(t, resp)

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		assert.Equal(t, 0, len(respBody["users"].([]interface{})))
	})

	t.Run("set-profile-picture-unauthorized", func(t *testing.T) {
		resp, err := httpClient.Post(Url+"/profile/set_picture", "application/json", bytes.NewBuffer([]byte{}))
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("set-profile-picture", func(t *testing.T) {
		cancel, err := AuthorizeClient(t, httpClient, "test", "new")
		assert.NoError(t, err)
		defer cancel()

		resp := SetProfilePicture(t, httpClient, []byte(
			"\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x06\xed\x00\x00\x06V\b\x06\x00\x00\x00\xa8"))
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Empty(t, resp.Body)

		// make sure the picture is not reset by worker
		buf := make([]byte, 16)
		for range 10 {
			resp = GetProfilePicture(t, httpClient, "test")
			defer resp.Body.Close()

			n, err := resp.Body.Read(buf)
			if n == 0 || err == io.EOF {
				assert.Equal(t, "", "picture was reset by worker")
				break
			}
			time.Sleep(time.Millisecond * 500)
		}
	})

	t.Run("set-profile-picture-bad-type", func(t *testing.T) {
		cancel, err := AuthorizeClient(t, httpClient, "test", "new")
		assert.NoError(t, err)
		defer cancel()

		resp := SetProfilePicture(t, httpClient, []byte("\x1a"))
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Empty(t, resp.Body)

		// make sure the picture is reset
		buf := make([]byte, 16)
		for i := range 10 {
			assert.NotEqual(t, 9, i)
			resp = GetProfilePicture(t, httpClient, "test")
			defer resp.Body.Close()

			n, err := resp.Body.Read(buf)
			if n == 0 || err == io.EOF {
				break
			}
			time.Sleep(time.Millisecond * 500)
		}
	})
}
