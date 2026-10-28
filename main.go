package main

import (
	"log"
	"net/http"
	"os"
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
	router.Use(m.Middleware())

	l10n, err := localization.New()
	if err != nil {
		log.Fatal(err)
	}
	themer := views.NewThemer()

	router.Use(l10n.Middleware())
	router.Use(themer.Middleware())
	router.Use(controllers.AuthMiddleware(db))
	router.Use(controllers.DailyRewardMiddleware(db))

	renderer := views.NewRenderer(l10n, themer)
	routes.RegisterRoutes(router, data, db, renderer)

	return router, m
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
