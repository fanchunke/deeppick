package main

import (
	"context"
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
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
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
	shutdown := otel.InitOpenTelemetry(
		ctx,
		otel.WithServiceName(cfg.Otel.ServiceName),
		otel.WithServiceVersion(cfg.Otel.ServiceVersion),
		otel.WithDeployEnvironment(cfg.Otel.DeployEnvironment),
		otel.WithHTTPEndpoint(cfg.Otel.HTTPEndpoint),
		otel.WithHTTPUrlPath(cfg.Otel.HTTPUrlPath),
	)
	defer shutdown()

	e := echo.New()
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.Use(otelecho.Middleware(cfg.Otel.ServiceName))

	openaiClient := openai.NewClient(option.WithBaseURL(cfg.OpenAI.BaseUrl), option.WithAPIKey(cfg.OpenAI.ApiKey))
	detectionSrv := service.NewChatCompletionService(openaiClient, cfg)
	resourceSrv := service.NewResourceService(cfg)
	e.POST("/api/image/detect", detectionSrv.DetectImage())
	e.POST("/api/image/upload", resourceSrv.Upload())

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
