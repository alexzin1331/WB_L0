package main

import (
	_ "WB_LVL0/docs"
	"WB_LVL0/server/internal/service"
	"WB_LVL0/server/internal/storage"
	k "WB_LVL0/server/kafka"
	"WB_LVL0/server/models"
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"log"
	"os"
	"os/signal"
	"syscall"
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
	serv := service.NewService(db)
	//init router
	router := gin.Default()
	router.GET("/", func(c *gin.Context) {
		//c.File("./server/static/index.html") -- local
		c.File("./static/index.html")

	})
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	router.GET("/order/:order_uid", serv.GetOrder)
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
