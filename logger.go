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
type Logger struct {
	Config
	Logger *slog.Logger
}

func New(conf *Config) *Logger {
	return &Logger{Config: *conf}
}

func (l *Logger) getLogger() *slog.Logger {
	if l.Logger == nil {
		return slog.Default()
	}
	return l.Logger
}

func (l *Logger) With(log *slog.Logger) *Logger {
	c := new(Logger)
	c.Config = l.Config
	c.Logger = log
	return c
}

func (l *Logger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	c := new(Logger)
	c.Config = l.Config
	c.Logger = l.Logger
	c.LogLevel = level
	return c
}

func (l *Logger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	sql, rows := fc()
	latency := time.Since(begin)

	if err != nil && (!l.IgnoreRecordNotFoundError || !errors.Is(err, gorm.ErrRecordNotFound)) {
		if l.LogLevel >= gormlogger.Error {
			l.getLogger().WarnContext(ctx, "Traced",
				slog.String("error", err.Error()),
				slog.Duration("latency", latency),
				slog.Int64("rows", rows),
				slog.String("sql", sql),
			)
		}
	} else if l.SlowThreshold != 0 && latency > l.SlowThreshold {
		if l.LogLevel >= gormlogger.Warn {
			l.getLogger().InfoContext(ctx, "Traced",
				slog.Duration("latency", latency),
				slog.Int64("rows", rows),
				slog.String("sql", sql),
			)
		}
	} else {
		if l.LogLevel >= gormlogger.Info {
			l.getLogger().DebugContext(ctx, "Traced",
				slog.Duration("latency", latency),
				slog.Int64("rows", rows),
				slog.String("sql", sql),
			)
		}
	}
}

func (l *Logger) Error(ctx context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= gormlogger.Error {
		l.getLogger().WarnContext(ctx, fmt.Sprintf(msg, args...))
	}
}

func (l *Logger) Warn(ctx context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= gormlogger.Warn {
		l.getLogger().InfoContext(ctx, fmt.Sprintf(msg, args...))
	}
}

func (l *Logger) Info(ctx context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= gormlogger.Info {
		l.getLogger().DebugContext(ctx, fmt.Sprintf(msg, args...))
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
