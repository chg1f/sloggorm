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
	SlowThreshold             time.Duration
	IgnoreRecordNotFoundError bool
	ParameterizedQueries      bool
	LogLevel                  gormlogger.LogLevel
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
	if l.LogLevel <= gormlogger.Silent {
		return
	}
	latency := time.Since(begin)
	sql, rows := fc()
	if l.LogLevel >= gormlogger.Error &&
		err != nil && (!l.IgnoreRecordNotFoundError || !errors.Is(err, gorm.ErrRecordNotFound)) {
		if l.LogLevel < gormlogger.Error {
			return
		}
		l.Logger.LogAttrs(ctx, slog.LevelError, "gorm trace",
			slog.String("error", err.Error()),
			slog.Duration("latency", latency),
			slog.Int64("rows", rows),
			slog.String("sql", sql),
		)
	} else if l.LogLevel >= gormlogger.Warn &&
		l.SlowThreshold != 0 && latency > l.SlowThreshold {
		l.Logger.LogAttrs(ctx, slog.LevelWarn, "gorm trace",
			slog.Duration("latency", latency),
			slog.Int64("rows", rows),
			slog.String("sql", sql),
		)
	} else if l.LogLevel >= gormlogger.Info {
		l.Logger.LogAttrs(ctx, slog.LevelDebug, "gorm trace",
			slog.Duration("latency", latency),
			slog.Int64("rows", rows),
			slog.String("sql", sql),
		)
	}
}
func (l *Logger) Info(ctx context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= gormlogger.Info {
		l.Logger.LogAttrs(ctx, slog.LevelInfo, fmt.Sprintf(msg, args...))
	}
}
func (l *Logger) Warn(ctx context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= gormlogger.Warn {
		l.Logger.LogAttrs(ctx, slog.LevelWarn, fmt.Sprintf(msg, args...))
	}
}
func (l *Logger) Error(ctx context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= gormlogger.Error {
		l.Logger.LogAttrs(ctx, slog.LevelWarn, fmt.Sprintf(msg, args...))
	}
}
func (l *Logger) ParamsFilter(ctx context.Context, sql string, params ...interface{}) (string, []interface{}) {
	if l.ParameterizedQueries {
		return sql, nil
	}
	return sql, params
}

var _ gormlogger.Interface = &Logger{}
var _ gorm.ParamsFilter = &Logger{}
