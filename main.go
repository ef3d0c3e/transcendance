package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"transcendance/config"
	"transcendance/controllers"
	"transcendance/data"
	"transcendance/localization"
	"transcendance/metrics"
	"transcendance/routes"
	"transcendance/views"
)

func run(data *data.Data, db *gorm.DB) (*gin.Engine, *metrics.Metrics) {
	if err := config.MigrateDatabase(db); err != nil {
		log.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	m := metrics.New(sqlDB)

	router := gin.Default()
	// Only the gateway names the client, in X-Real-IP, which it always overwrites
	router.RemoteIPHeaders = []string{"X-Real-IP"}
	if err := router.SetTrustedProxies(trustedProxies()); err != nil {
		log.Fatal(err)
	}
	router.Use(m.Middleware())

	l10n, err := localization.New()
	if err != nil {
		log.Fatal(err)
	}
	themer := views.NewThemer()

	router.Use(l10n.Middleware())
	router.Use(themer.Middleware())
	router.Use(controllers.AuthMiddleware(db))

	renderer := views.NewRenderer(l10n, themer)
	routes.RegisterRoutes(router, data, db, renderer)

	return router, m
}

// Addresses allowed to name the client, from TRUSTED_PROXIES. None when unset
func trustedProxies() []string {
	var proxies []string
	for _, proxy := range strings.Split(os.Getenv("TRUSTED_PROXIES"), ",") {
		if proxy = strings.TrimSpace(proxy); proxy != "" {
			proxies = append(proxies, proxy)
		}
	}
	return proxies
}

func main() {
	data, err := data.LoadCards()
	if err != nil {
		log.Fatal(err)
		return
	}

	db, err := config.ConnectDatabase()
	if err != nil {
		log.Fatal(err)
	}

	router, m := run(data, db)

	// Prometheus scrapes this address, keep it off the public network
	if addr := os.Getenv("METRICS_ADDR"); addr != "" {
		server := &http.Server{Addr: addr, Handler: m.Handler(), ReadHeaderTimeout: 5 * time.Second}
		go func() {
			log.Printf("Metrics on http://%s/metrics", addr)
			if err := server.ListenAndServe(); err != nil {
				log.Fatal(err)
			}
		}()
	}

	log.Println("Server running on http://localhost:8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
