package metrics_test

import (
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"transcendance/metrics"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

func TestRequestsAreCountedByRoute(t *testing.T) {
	// Never connects, only its pool statistics are read
	db, err := sql.Open("mysql", "user:password@tcp(127.0.0.1:1)/none")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	m := metrics.New(db)
	router := gin.New()
	router.Use(m.Middleware())
	router.GET("/cards/:id", func(c *gin.Context) { c.Status(http.StatusOK) })

	for _, target := range []string{"/cards/1", "/cards/2", "/missing"} {
		router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, target, nil))
	}

	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body, err := io.ReadAll(rec.Body)
	require.NoError(t, err)
	page := string(body)

	require.Contains(t, page, `http_requests_total{method="GET",route="/cards/:id",status="200"} 2`)
	require.Contains(t, page, `http_requests_total{method="GET",route="unmatched",status="404"} 1`)
	require.Contains(t, page, `http_request_duration_seconds_count{method="GET",route="/cards/:id"} 2`)
	require.Contains(t, page, `go_sql_open_connections{db_name="main"}`)
	require.Contains(t, page, "go_goroutines")
}
