package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"time"

	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/extra/bundebug"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zapio"

	"github.com/helpify-project/backend/internal/controllers"
)

func main() {
	ctx := context.Background()
	ctx, _ = signal.NotifyContext(ctx, os.Interrupt)

	app := &cli.App{
		Name: "helpify-api",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "debug",
				Value: false,
				EnvVars: []string{
					"HELPIFY_API_DEBUG",
				},
			},
			&cli.StringFlag{
				Name:  "http-listen-address",
				Value: "127.0.0.1:3009",
				EnvVars: []string{
					"HELPIFY_API_HTTP_LISTEN_ADDRESS",
				},
			},
			&cli.StringFlag{
				Name:     "postgres-uri",
				Required: true,
				EnvVars: []string{
					"HELPIFY_API_POSTGRES_URI",
				},
			},
			&cli.StringFlag{
				Name:  "session-secret",
				Value: `d+rOlDT4uH5foUDCDSPFpxKnY0tcrR0U8UWfZ6ng+sYAZinksr9G/bRxLV107ze9K2zFgoJj/zz8d542fRRgFQ==`,
			},
		},
		Before: func(cctx *cli.Context) (err error) {
			err = setupLogging(cctx.Bool("debug"))
			return
		},
		Action: entrypoint,
	}

	if err := app.RunContext(ctx, os.Args); err != nil {
		zap.L().Fatal("unhandled error", zap.Error(err))
	}
}

func setupLogging(debugMode bool) error {
	var cfg zap.Config

	if debugMode {
		cfg = zap.NewDevelopmentConfig()
		cfg.Level.SetLevel(zapcore.DebugLevel)
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		cfg.Development = false
	} else {
		cfg = zap.NewProductionConfig()
		cfg.Level.SetLevel(zapcore.InfoLevel)
	}

	cfg.OutputPaths = []string{
		"stdout",
	}

	logger, err := cfg.Build()
	if err != nil {
		return err
	}

	zap.ReplaceGlobals(logger)

	return nil
}

func entrypoint(cctx *cli.Context) (err error) {
	ctx := cctx.Context
	defer func() { _ = zap.L().Sync() }()

	var dbConfig *pgx.ConnConfig
	if dbConfig, err = pgx.ParseConfig(cctx.String("postgres-uri")); err != nil {
		err = fmt.Errorf("unable to parse postgres uri: %w", err)
		return
	}

	sqldb := stdlib.OpenDB(*dbConfig)
	db := bun.NewDB(sqldb, pgdialect.New())
	defer func() { _ = db.Close() }()

	if cctx.Bool("debug") {
		var dbLogger io.WriteCloser = &zapio.Writer{Log: zap.L().With(zap.String("section", "bun")), Level: zapcore.DebugLevel}
		defer func() { _ = dbLogger.Close() }()

		db.AddQueryHook(bundebug.NewQueryHook(
			bundebug.WithVerbose(true),
			bundebug.WithWriter(dbLogger),
		))
	}

	if _, err = db.ExecContext(ctx, "SELECT 1"); err != nil {
		err = fmt.Errorf("failed to test database connection: %w", err)
		return
	}

	// XXX: Render pls
	listenAddr := cctx.String("http-listen-address")
	if port := os.Getenv("PORT"); port != "" {
		listenAddr = ":" + port
	}

	router := mux.NewRouter()
	handler := gorillaHandlers.CORS(
		gorillaHandlers.AllowCredentials(),
		gorillaHandlers.AllowedHeaders([]string{
			"Authorization",
			"Content-Type",
			"Origin",
			"X-Requested-With",
		}),
		gorillaHandlers.AllowedMethods([]string{
			http.MethodDelete,
			http.MethodGet,
			http.MethodHead,
			http.MethodOptions,
			http.MethodPatch,
			http.MethodPost,
			http.MethodPut,
		}),
		gorillaHandlers.AllowedOrigins([]string{
			"http://localhost",
			"https://helpify-frontend.onrender.com",
		}),
		gorillaHandlers.MaxAge(86400),
	)

	srv := &http.Server{
		Addr:         listenAddr,
		Handler:      handler(router),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})

	if cctx.Bool("debug") {
		(&controllers.GoDebugController{}).Register(router)
	}
	(&controllers.ChatController{
		DB:            db,
		SessionSecret: cctx.String("session-secret"),
	}).Register(router)
	(&controllers.HealthController{}).Register(router)

	serverDone := make(chan interface{})
	go func() {
		zap.L().Info("serving requests", zap.String("addr", "http://"+srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			zap.L().Error("failed to listen for http requests", zap.Error(err))
		}
		close(serverDone)
	}()

	select {
	case <-serverDone:
	case <-cctx.Context.Done():
	}

	return
}
