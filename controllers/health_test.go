package controllers_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"transcendance/controllers"
	"transcendance/tests"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestHealthzOk(t *testing.T) {
	router := tests.Run(tests.Data, tests.DB)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"status":"ok"}`, rec.Body.String())
}

func TestHealthzDatabaseDown(t *testing.T) {
	// Nothing listens on port 1
	conn, err := sql.Open("mysql", "user:password@tcp(127.0.0.1:1)/none")
	require.NoError(t, err)
	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      conn,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{DisableAutomaticPing: true})
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	router := gin.New()
	health := controllers.Controller{DB: db}
	router.GET("/healthz", health.HealthzGet)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.JSONEq(t, `{"status":"unavailable"}`, rec.Body.String())
}
