package sloggorm

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type Config struct {
	LogLevel                  gormlogger.LogLevel
	ParameterizedQueries      bool
	IgnoreRecordNotFoundError bool
	FailedLevel               slog.Level
	SlowThreshold             time.Duration
}

type Logger struct {
	*slog.Logger
	Config
}

func New(log *slog.Logger, cfg *Config) *Logger {
	return &Logger{
		Logger: log,
		Config: *cfg,
	}
}

func (l *Logger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	n := new(Logger)
	n.Config = l.Config
	n.Logger = l.Logger
	n.LogLevel = level
	return n
}

func (l *Logger) Error(ctx context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= gormlogger.Error {
		l.ErrorContext(ctx, fmt.Sprintf(msg, args...))
	}
}

func (l *Logger) Warn(ctx context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= gormlogger.Warn {
		l.WarnContext(ctx, fmt.Sprintf(msg, args...))
	}
}

func (l *Logger) Info(ctx context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= gormlogger.Info {
		l.InfoContext(ctx, fmt.Sprintf(msg, args...))
	}
}

func (l *Logger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.LogLevel <= gormlogger.Silent {
		return
	}

	latency := time.Since(begin)

	log := l.With(slog.Duration("latency", latency))
	sql, rows := fc()
	slog.With(log, slog.String("sql", sql))
	if rows == -1 {
		log.With(slog.Int64("rows", rows))
	}

	if err != nil && (!l.IgnoreRecordNotFoundError || !errors.Is(err, gorm.ErrRecordNotFound)) {
		log.WarnContext(ctx, "Failed", slog.String("error", err.Error()))
	} else if l.SlowThreshold != 0 && latency > l.SlowThreshold {
		log.InfoContext(ctx, "Slowed")
	} else {
		log.DebugContext(ctx, "Traced")
	}
}

var _ gormlogger.Interface = &Logger{}

func (l *Logger) ParamsFilter(ctx context.Context, sql string, params ...interface{}) (string, []interface{}) {
	if l.ParameterizedQueries {
		return sql, nil
	}
	return sql, params
}

var _ gorm.ParamsFilter = &Logger{}
