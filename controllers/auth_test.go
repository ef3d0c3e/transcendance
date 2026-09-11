package controllers_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"transcendance/models"
	"transcendance/tests"

	"github.com/stretchr/testify/require"
)

func TestRegister(t *testing.T) {
	form := url.Values{
		"username":         {"alice"},
		"password":         {"foo-bar-qux"},
		"password_confirm": {"foo-bar-qux"},
		"tos_agree":        {"on"},
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/register",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()

	router := tests.Run(tests.DB)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)

	var user models.User
	err := tests.DB.
		Where("username = ?", "alice").
		First(&user).
		Error

	require.NoError(t, err)
	require.Equal(t, "alice", user.Username)
	require.NotEqual(t, form.Get("password"), user.PasswordHash)
	require.NotEmpty(t, user.PasswordHash)
}

func TestUsernameTaken(t *testing.T) {
	{
		form := url.Values{
			"username":         {"eve"},
			"password":         {"foo-bar-baz"},
			"password_confirm": {"foo-bar-baz"},
			"tos_agree":        {"on"},
		}

		req := httptest.NewRequest(
			http.MethodPost,
			"/register",
			strings.NewReader(form.Encode()),
		)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rec := httptest.NewRecorder()

		router := tests.Run(tests.DB)
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)
	}
	{
		form := url.Values{
			"username":         {"eve"},
			"password":         {"foo-bar-quz"},
			"password_confirm": {"foo-bar-quz"},
			"tos_agree":        {"on"},
		}

		req := httptest.NewRequest(
			http.MethodPost,
			"/register",
			strings.NewReader(form.Encode()),
		)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rec := httptest.NewRecorder()

		router := tests.Run(tests.DB)
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusConflict, rec.Code)
	}
}

func TestUsernameTooShort(t *testing.T) {
	form := url.Values{
		"username":         {"al"},
		"password":         {"foo-bar-qux"},
		"password_confirm": {"foo-bar-qux"},
		"tos_agree":        {"on"},
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/register",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()

	router := tests.Run(tests.DB)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUsernameTooLong(t *testing.T) {
	form := url.Values{
		"username":         {"alice123456789abcdef"},
		"password":         {"foo-bar-qux"},
		"password_confirm": {"foo-bar-qux"},
		"tos_agree":        {"on"},
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/register",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()

	router := tests.Run(tests.DB)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUsernameInvalid(t *testing.T) {
	form := url.Values{
		"username":         {"alice$"},
		"password":         {"foo-bar-qux"},
		"password_confirm": {"foo-bar-qux"},
		"tos_agree":        {"on"},
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/register",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()

	router := tests.Run(tests.DB)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPasswordTooShort(t *testing.T) {
	form := url.Values{
		"username":         {"bob"},
		"password":         {"foo-bar"},
		"password_confirm": {"foo-bar"},
		"tos_agree":        {"on"},
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/register",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()

	router := tests.Run(tests.DB)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPasswordTooLong(t *testing.T) {
	form := url.Values{
		"username":         {"bob"},
		"password":         {"foo-bar-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		"password_confirm": {"foo-bar-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		"tos_agree":        {"on"},
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/register",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()

	router := tests.Run(tests.DB)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPasswordMismatch(t *testing.T) {
	form := url.Values{
		"username":         {"bob"},
		"password":         {"foo-bar-baz"},
		"password_confirm": {"foo-bar-qux"},
		"tos_agree":        {"on"},
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/register",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()

	router := tests.Run(tests.DB)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestMain(m *testing.M) {
	if err := os.Chdir(".."); err != nil {
		panic(err)
	}

	if err := tests.Init(); err != nil {
		panic(err)
	}

	code := m.Run()

	tests.Cleanup()

	os.Exit(code)
}
