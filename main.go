package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/fanchunke/deeppick-ai/internal/config"
	"github.com/fanchunke/deeppick-ai/internal/otel"
	"github.com/fanchunke/deeppick-ai/internal/service"
	_ "github.com/go-sql-driver/mysql"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/panjf2000/ants/v2"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
)

var (
	conf string
)

func init() {
	flag.StringVar(&conf, "conf", "conf/online.toml", "配置文件")
}

func main() {

	flag.Parse()

	// load config
	cfg, err := config.NewConfig(conf)
	if err != nil {
		log.Fatalf("load config error: %v", err)
	}

	ctx := context.Background()
	// init opentelemetry
	shutdown := otel.InitOpenTelemetry(
		ctx,
		otel.WithServiceName(cfg.Otel.ServiceName),
		otel.WithServiceVersion(cfg.Otel.ServiceVersion),
		otel.WithDeployEnvironment(cfg.Otel.DeployEnvironment),
		otel.WithHTTPEndpoint(cfg.Otel.HTTPEndpoint),
		otel.WithHTTPUrlPath(cfg.Otel.HTTPUrlPath),
	)
	defer shutdown()

	// 初始化数据库
	db, err := sql.Open(cfg.Database.Driver, cfg.Database.DataSource)
	if err != nil {
		log.Fatalf("open database error: %v", err)
	}
	defer db.Close()

	// 初始化协程池
	pool, err := ants.NewPool(10, ants.WithPreAlloc(true))
	if err != nil {
		log.Fatalf("init ants pool error: %v", err)
	}
	defer pool.Release()

	e := echo.New()
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.Use(otelecho.Middleware(cfg.Otel.ServiceName))

	openaiClient := openai.NewClient(option.WithBaseURL(cfg.OpenAI.BaseUrl), option.WithAPIKey(cfg.OpenAI.ApiKey))
	detectionSrv := service.NewChatCompletionService(openaiClient, cfg, db, pool, e.Logger)
	resourceSrv := service.NewResourceService(cfg)
	e.POST("/api/image/detect", detectionSrv.DetectImage())
	e.POST("/api/image/upload", resourceSrv.Upload())
	e.GET("/api/task/result", detectionSrv.GetTask())

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	go func() {
		if err := e.Start(fmt.Sprintf(":%d", cfg.HTTP.Port)); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("shutting down the server")
		}
	}()

	<-ctx.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}
