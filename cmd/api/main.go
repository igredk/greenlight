package main

import (
	"context"
	"expvar"
	"flag"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/igredk/greenlight/internal/data"
	"github.com/igredk/greenlight/internal/jsonlog"
	"github.com/igredk/greenlight/internal/mailer"
	"github.com/jackc/pgx/v5/pgxpool"
)

const version string = "1.0.0"

type config struct {
	port int
	env  string
	db   struct {
		dsn          string
		maxOpenConns int
		maxIdleTime  string
	}
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
	cors struct {
		trustedOrigins []string
	}
}

type application struct {
	config config
	logger *jsonlog.Logger
	models data.Models
	mailer mailer.Mailer
	wg     sync.WaitGroup
}

func main() {
	var cfg config
	// HTTP server
	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
	// Database
	flag.StringVar(&cfg.db.dsn, "pg-dsn", "", "PostgreSQL connection URL")
	flag.IntVar(&cfg.db.maxOpenConns, "pg-max-open-conns", 25, "PostgreSQL max open connections")
	flag.StringVar(&cfg.db.maxIdleTime, "pg-max-idle-time", "15m", "PostgreSQL max connection idle time")
	// Rate limiter
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")
	// SMTP server
	flag.StringVar(&cfg.smtp.host, "smtp-host", "sandbox.smtp.mailtrap.io", "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 25, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", "dc1a969cd7920e", "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", "68935db316fb19", "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "Greenlight <no-reply@greenlight.com>", "SMTP sender")
	// CORS
	flag.Func("cors-trusted-origins", "Trusted CORS origins (space separated)", func(val string) error {
		cfg.cors.trustedOrigins = strings.Fields(val)
		return nil
	})

	flag.Parse()

	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	dbPool, err := openDB(cfg)
	if err != nil {
		logger.PrintFatal(err, nil)
	}
	defer dbPool.Close()
	logger.PrintInfo(
		"database connection established",
		map[string]string{
			"max_conns":      strconv.Itoa(int((dbPool.Config().MaxConns))),
			"conn_idle_time": dbPool.Config().MaxConnIdleTime.String(),
		},
	)

	// Publish version in metrics.
	expvar.NewString("version").Set(version)
	// Publish the number of active goroutines.
	expvar.Publish("goroutines", expvar.Func(func() any {
		return runtime.NumGoroutine()
	}))
	// Publish the database connection pool statistics in a serializable format.
	expvar.Publish("database", expvar.Func(func() any {
		stats := dbPool.Stat()
		return map[string]interface{}{
			"acquire_count":              stats.AcquireCount(),
			"acquire_duration":           stats.AcquireDuration(),
			"acquired_conns":             stats.AcquiredConns(),
			"canceled_acquire":           stats.CanceledAcquireCount(),
			"constructing_conns":         stats.ConstructingConns(),
			"empty_acquire_count":        stats.EmptyAcquireCount(),
			"idle_conns":                 stats.IdleConns(),
			"max_conns":                  stats.MaxConns(),
			"max_idle_destroy_count":     stats.MaxIdleDestroyCount(),
			"max_lifetime_destroy_count": stats.MaxLifetimeDestroyCount(),
			"new_conns_count":            stats.NewConnsCount(),
			"total_conns":                stats.TotalConns(),
		}
	}))
	// Publish the current Unix timestamp.
	expvar.Publish("timestamp", expvar.Func(func() any {
		return time.Now().Unix()
	}))

	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(dbPool),
		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
	}

	err = app.serve() // start the HTTP server
	if err != nil {
		logger.PrintFatal(err, nil) // log the error and exit
	}
}

func openDB(cfg config) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config, err := pgxpool.ParseConfig(cfg.db.dsn)
	if err != nil {
		return nil, err
	}

	config.MaxConns = int32(cfg.db.maxOpenConns)
	duration, err := time.ParseDuration(cfg.db.maxIdleTime)
	if err != nil {
		return nil, err
	}
	config.MaxConnIdleTime = duration

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	if err = pool.Ping(ctx); err != nil {
		return nil, err
	}

	return pool, nil
}
