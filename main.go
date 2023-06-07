package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gofiber/contrib/fiberzap"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/nsqio/go-nsq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
	"go.uber.org/zap"
)

var (
	port      = flag.Int("port", 4252, "HTTP port")
	addr      = flag.String("nsqd-tcp-address", "localhost:4150", "nsqd TCP address")
	lookupd   = flag.String("lookupd-http-address", "", "nsqlookupd HTTP address")
	goMetrics = flag.Bool("gom", false, "Expose Go runtime metrics")
	logLevel  = zap.LevelFlag("log", zap.InfoLevel, "log level (debug, info, warn, error, dpanic, panic, fatal)")
)

func main() {
	var err error
	flag.Parse()

	// logger setup

	zapCfg := zap.NewProductionConfig()
	zapCfg.Level = zap.NewAtomicLevelAt(*logLevel)
	zapCfg.Encoding = "json"
	zapCfg.DisableCaller = true
	zapCfg.DisableStacktrace = true
	zapCfg.OutputPaths = []string{"stdout"}
	zapCfg.ErrorOutputPaths = []string{"stderr"}
	zapCfg.EncoderConfig = zap.NewProductionEncoderConfig()
	logger, err := zapCfg.Build(zap.Fields(zap.String("app", "http_to_nsq")))
	if err != nil {
		panic(err)
	}

	// metrics setup

	customRegistry := prometheus.NewRegistry()
	httpReqs := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Number of HTTP requests.",
			ConstLabels: prometheus.Labels{
				"app":  "http_to_nsq",
				"host": strings.Split(*addr, ":")[0],
			},
		},
		[]string{"status", "topic"},
	)
	customRegistry.MustRegister(httpReqs)
	if *goMetrics {
		customRegistry.MustRegister(collectors.NewGoCollector())
	}
	promHandler := promhttp.HandlerFor(customRegistry, promhttp.HandlerOpts{})

	// nsq setup

	config := nsq.NewConfig()
	var producer *nsq.Producer
	if *lookupd != "" {
		logger.Info("Connecting to NSQ", zap.String("via", "nsqlookupd"), zap.String("address", *lookupd))
		producer, err = nsq.NewProducer(*lookupd, config)
	} else {
		logger.Info("Connecting to NSQ", zap.String("via", "nsqd"), zap.String("address", *addr))
		producer, err = nsq.NewProducer(*addr, config)
	}
	if err != nil {
		logger.Fatal("failed to create NSQ producer", zap.Error(err))
	}
	var nsqLogLevel = nsq.LogLevelInfo
	switch *logLevel {
	case zap.DebugLevel:
		nsqLogLevel = nsq.LogLevelDebug
	case zap.InfoLevel:
		nsqLogLevel = nsq.LogLevelInfo
	case zap.WarnLevel:
		nsqLogLevel = nsq.LogLevelWarning
	case zap.ErrorLevel:
		nsqLogLevel = nsq.LogLevelError
	case zap.DPanicLevel:
		nsqLogLevel = nsq.LogLevelError
	case zap.PanicLevel:
		nsqLogLevel = nsq.LogLevelError
	case zap.FatalLevel:
		nsqLogLevel = nsq.LogLevelError
	}
	nsqZapLogger := &nsqZapLogger{logger: logger}
	producer.SetLogger(nsqZapLogger, nsqLogLevel)
	err = producer.Ping()
	if err != nil {
		logger.Fatal("failed to ping NSQ", zap.Error(err))
	}
	logger.Info("Connected to NSQ")
	defer producer.Stop()

	// server setup

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(fiberzap.New(fiberzap.Config{
		SkipBody:    func(c *fiber.Ctx) bool { return true },
		SkipResBody: func(c *fiber.Ctx) bool { return true },
		SkipURIs:    []string{"/metrics"},
		Logger:      logger,
	}))
	app.Get("/metrics", func(c *fiber.Ctx) error {
		fasthttpadaptor.NewFastHTTPHandler(promHandler)(c.Context())
		return nil
	})
	app.Post("/:topic", func(c *fiber.Ctx) error {
		topic := utils.CopyString(c.Params("topic"))
		if topic == "" {
			httpReqs.WithLabelValues("error", topic).Inc()
			return c.SendStatus(http.StatusBadRequest)
		}

		body := utils.CopyBytes(c.Body())
		if err := producer.Publish(topic, body); err != nil {
			httpReqs.WithLabelValues("error", topic).Inc()
			logger.Error("Failed to publish message", zap.String("topic", topic), zap.Error(err))
			return c.SendStatus(http.StatusInternalServerError)
		}
		httpReqs.WithLabelValues("ok", topic).Inc()
		logger.Info("Published message to topic", zap.String("topic", topic))
		return nil
	})
	go func() {
		logger.Info("Server listening", zap.Int("port", *port))
		if err := app.Listen(fmt.Sprintf(":%d", *port)); err != nil {
			logger.Fatal("Failed to listen", zap.Error(err))
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	if err := app.Shutdown(); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}
	logger.Info("Server exiting")
}

type nsqZapLogger struct {
	logger *zap.Logger
}

func (n *nsqZapLogger) Output(_ int, s string) error {
	// Split the log line into parts.
	parts := strings.Fields(s)
	if len(parts) < 2 {
		return fmt.Errorf("failed to parse NSQ log line: %s", s)
	}

	level := parts[0]
	message := strings.Join(parts[1:], " ")

	// Remove any connection or producer ID from the message.
	message = strings.TrimLeft(message, "0123456789 ")

	// Map the NSQ log level to a Zap log level.
	var logFunc func(string, ...zap.Field)
	switch level {
	case "DBG":
		logFunc = n.logger.Debug
	case "INF":
		logFunc = n.logger.Info
	case "WRN":
		logFunc = n.logger.Warn
	case "ERR":
		logFunc = n.logger.Error
	default:
		logFunc = n.logger.Info
	}

	logFunc(message)
	return nil
}
