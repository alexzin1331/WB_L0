package main

import (
	_ "WB_LVL0/docs"
	"WB_LVL0/server/internal/auth"
	"WB_LVL0/server/internal/observability"
	"WB_LVL0/server/internal/service"
	"WB_LVL0/server/internal/storage"
	k "WB_LVL0/server/kafka"
	"WB_LVL0/server/models"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

const (
	configPath = "config.yaml"
)

// @title WB_LVL0 API
// @version 1.0
// @description API для работы с заказами
// @host localhost:8080
// @BasePath /
func main() {
	//init config
	cfg := models.MustLoad(configPath)
	//init PostrgeSQL
	db, err := storage.New(*cfg)
	if err != nil {
		log.Fatalf("can't set connection to postgres: %v", err)
	}
	//init kafka
	reader := k.NewReader()
	defer reader.Close()
	//init service
	authManager := auth.NewManager(cfg.AuthConf.JWTSecret, cfg.AuthConf.TokenTTL)
	serv := service.NewService(db, authManager)
	//init router
	metrics := observability.NewMetrics()
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), metrics.Middleware())
	setupPprof(router)
	router.GET("/", func(c *gin.Context) {
		//c.File("./server/static/index.html") -- local
		c.File("./static/index.html")

	})
	router.GET("/metrics", metrics.Handler)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/order/:order_uid", serv.GetOrder)

	router.POST("/api/auth/register", serv.Register)
	router.POST("/api/auth/login", serv.Login)
	api := router.Group("/api", serv.AuthRequired())
	{
		api.GET("/auth/me", serv.Me)
		api.GET("/orders", serv.ListOrders)
		api.GET("/orders/aggregate", serv.AggregateOrders)
		api.GET("/orders/filter-values", serv.GetFilterValues)
		api.GET("/order/:order_uid", serv.GetOrder)
	}

	router.Static("/static", "./static")
	//router.Static("/server/static", "./server/static")

	//server start
	go func() {
		if err := router.Run(cfg.ServConf.Host); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()
	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Processing message
	go func() {
		k.ReadMSG(db, reader)
	}()

	fmt.Println("Consumer started. Waiting for messages...")
	<-quit
	fmt.Println("Shutting down consumer...")
}

// setupPprof настраивает эндпоинты для профилирования
func setupPprof(router *gin.Engine) {
	// Группа роутов для pprof
	pprofGroup := router.Group("/debug/pprof")
	{
		pprofGroup.GET("/", gin.WrapH(http.DefaultServeMux))
		pprofGroup.GET("/cmdline", gin.WrapH(http.DefaultServeMux))
		pprofGroup.GET("/profile", gin.WrapH(http.DefaultServeMux))
		pprofGroup.POST("/symbol", gin.WrapH(http.DefaultServeMux))
		pprofGroup.GET("/symbol", gin.WrapH(http.DefaultServeMux))
		pprofGroup.GET("/trace", gin.WrapH(http.DefaultServeMux))
		pprofGroup.GET("/allocs", gin.WrapH(http.DefaultServeMux))
		pprofGroup.GET("/block", gin.WrapH(http.DefaultServeMux))
		pprofGroup.GET("/goroutine", gin.WrapH(http.DefaultServeMux))
		pprofGroup.GET("/heap", gin.WrapH(http.DefaultServeMux))
		pprofGroup.GET("/mutex", gin.WrapH(http.DefaultServeMux))
		pprofGroup.GET("/threadcreate", gin.WrapH(http.DefaultServeMux))
	}
}
