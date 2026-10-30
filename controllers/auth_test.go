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

func TestRegisterOk(t *testing.T) {
	form := url.Values{
		"username":         {"alice"},
		"email":            {"alice@example.test"},
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

	router := tests.Run(tests.Data, tests.DB)
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

func TestRegisterUsernameTaken(t *testing.T) {
	{
		form := url.Values{
			"username":         {"eve"},
			"email":            {"eve@example.test"},
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

		router := tests.Run(tests.Data, tests.DB)
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)
	}
	{
		form := url.Values{
			"username":         {"eve"},
			"email":            {"eve-other@example.test"},
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

		router := tests.Run(tests.Data, tests.DB)
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusConflict, rec.Code)
	}
}

func TestRegisterUsernameTooShort(t *testing.T) {
	form := url.Values{
		"username":         {"al"},
		"email":            {"al@example.test"},
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

	router := tests.Run(tests.Data, tests.DB)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegisterUsernameTooLong(t *testing.T) {
	form := url.Values{
		"username":         {"alice123456789abcdef"},
		"email":            {"alice123456789abcdef@example.test"},
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

	router := tests.Run(tests.Data, tests.DB)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegisterUsernameInvalid(t *testing.T) {
	form := url.Values{
		"username":         {"alice$"},
		"email":            {"alice$@example.test"},
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

	router := tests.Run(tests.Data, tests.DB)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegisterPasswordTooShort(t *testing.T) {
	form := url.Values{
		"username":         {"bob"},
		"email":            {"bob@example.test"},
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

	router := tests.Run(tests.Data, tests.DB)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegisterPasswordTooLong(t *testing.T) {
	form := url.Values{
		"username":         {"bob"},
		"email":            {"bob@example.test"},
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

	router := tests.Run(tests.Data, tests.DB)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegisterPasswordMismatch(t *testing.T) {
	form := url.Values{
		"username":         {"bob"},
		"email":            {"bob@example.test"},
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

	router := tests.Run(tests.Data, tests.DB)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegisterLoginOk(t *testing.T) {
	{
		form := url.Values{
			"username":         {"alice2"},
			"email":            {"alice2@example.test"},
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

		router := tests.Run(tests.Data, tests.DB)
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)
	}

	form := url.Values{
		"email":      {"alice2@example.test"},
		"login_type": {"email"},
		"password":   {"foo-bar-qux"},
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/login",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()

	router := tests.Run(tests.Data, tests.DB)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestRegisterBadPassword(t *testing.T) {
	{
		form := url.Values{
			"username":         {"alice3"},
			"email":            {"alice3@example.test"},
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

		router := tests.Run(tests.Data, tests.DB)
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)
	}

	form := url.Values{
		"username": {"alice3"},
		"password": {"foo-bar-baz"},
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/login",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()

	router := tests.Run(tests.Data, tests.DB)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestRegisterBadUsername(t *testing.T) {
	{
		form := url.Values{
			"username":         {"alice4"},
			"email":            {"alice4@example.test"},
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

		router := tests.Run(tests.Data, tests.DB)
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusCreated, rec.Code)
	}

	form := url.Values{
		"username": {"bob4"},
		"password": {"foo-bar-qux"},
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/login",
		strings.NewReader(form.Encode()),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rec := httptest.NewRecorder()

	router := tests.Run(tests.Data, tests.DB)
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
