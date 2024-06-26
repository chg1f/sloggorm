package sloggorm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sagikazarmark/slog-shim"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type Config struct {
	SlowThreshold             time.Duration       `json:"slow_threshold"`
	IgnoreRecordNotFoundError bool                `json:"ignore_record_not_found_error"`
	ParameterizedQueries      bool                `json:"parameterized_queries"`
	LogLevel                  gormlogger.LogLevel `json:"log_level"`
}

func NewConfig() *Config {
	return &Config{
		SlowThreshold:             time.Second,
		IgnoreRecordNotFoundError: true,
		ParameterizedQueries:      false,
		LogLevel:                  gormlogger.Warn,
	}
}

type Logger struct {
	Config
	*slog.Logger
}

func NewLogger(logger *slog.Logger, conf *Config) *Logger {
	l := new(Logger)
	l.Config = *conf
	l.Logger = logger
	return l
}

func (l *Logger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	clone := new(Logger)
	clone.Logger = l.Logger
	clone.Config = l.Config
	clone.LogLevel = level
	return clone
}

func (l *Logger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	sql, rows := fc()
	latency := time.Since(begin)

	if err != nil && (!l.IgnoreRecordNotFoundError || !errors.Is(err, gorm.ErrRecordNotFound)) {
		if l.LogLevel >= gormlogger.Error {
			l.WarnContext(ctx, "Traced",
				slog.String("error", err.Error()),
				slog.Duration("latency", latency),
				slog.Int64("rows", rows),
				slog.String("sql", sql),
			)
		}
	} else if l.SlowThreshold != 0 && latency > l.SlowThreshold {
		if l.LogLevel >= gormlogger.Warn {
			l.InfoContext(ctx, "Traced",
				slog.Duration("latency", latency),
				slog.Int64("rows", rows),
				slog.String("sql", sql),
			)
		}
	} else {
		if l.LogLevel >= gormlogger.Info {
			l.DebugContext(ctx, "Traced",
				slog.Duration("latency", latency),
				slog.Int64("rows", rows),
				slog.String("sql", sql),
			)
		}
	}
}

func (l *Logger) Error(ctx context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= gormlogger.Error {
		l.WarnContext(ctx, fmt.Sprintf(msg, args...))
	}
}

func (l *Logger) Warn(ctx context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= gormlogger.Warn {
		l.InfoContext(ctx, fmt.Sprintf(msg, args...))
	}
}

func (l *Logger) Info(ctx context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= gormlogger.Info {
		l.DebugContext(ctx, fmt.Sprintf(msg, args...))
	}
}

func (l *Logger) ParamsFilter(ctx context.Context, sql string, params ...interface{}) (string, []interface{}) {
	if l.ParameterizedQueries {
		return sql, nil
	}
	return sql, params
}

var (
	_ gormlogger.Interface = &Logger{}
	_ gorm.ParamsFilter    = &Logger{}
)
