package main

import (
	"context"
	"flag"
	"os"
	"strconv"
	"time"

	"github.com/igredk/greenlight/internal/data"
	"github.com/igredk/greenlight/internal/jsonlog"
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
}

type application struct {
	config config
	logger *jsonlog.Logger
	models data.Models
}

func main() {
	var cfg config
	// HTTP server
	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
	// Database
	flag.StringVar(&cfg.db.dsn, "pg-dsn", "postgres://biba:boba@localhost:5432/greenlight", "PostgreSQL connection URL")
	flag.IntVar(&cfg.db.maxOpenConns, "pg-max-open-conns", 25, "PostgreSQL max open connections")
	flag.StringVar(&cfg.db.maxIdleTime, "pg-max-idle-time", "15m", "PostgreSQL max connection idle time")
	// Rate limiter
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")

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

	app := &application{
		config: cfg,
		logger: logger,
		models: data.NewModels(dbPool),
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
