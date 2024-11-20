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

type (
	Config struct {
		SlowThreshold             time.Duration
		IgnoreRecordNotFoundError bool
		ParameterizedQueries      bool
		LogLevel                  gormlogger.LogLevel
	}
	Logger struct {
		Config
		*slog.Logger
	}
)

func New(log *slog.Logger, cfg *Config) *Logger { return &Logger{Logger: log, Config: *cfg} }

var (
	Default = New(nil, &Config{
		SlowThreshold:             time.Second,
		LogLevel:                  gormlogger.Info,
		IgnoreRecordNotFoundError: true,
	})
	Discard = Default.LogMode(gormlogger.Silent)
)

func (l *Logger) logger() *slog.Logger {
	if l.Logger == nil {
		return slog.Default()
	}
	return l.Logger
}

func (l *Logger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	n := new(Logger)
	n.Config = l.Config
	n.Logger = l.Logger
	n.LogLevel = level
	return n
}

func (l *Logger) Error(ctx context.Context, msg string, args ...interface{}) {
	if l.Config.LogLevel >= gormlogger.Error {
		l.logger().ErrorContext(ctx, fmt.Sprintf(msg, args...))
	}
}

func (l *Logger) Warn(ctx context.Context, msg string, args ...interface{}) {
	if l.Config.LogLevel >= gormlogger.Warn {
		l.logger().WarnContext(ctx, fmt.Sprintf(msg, args...))
	}
}

func (l *Logger) Info(ctx context.Context, msg string, args ...interface{}) {
	if l.Config.LogLevel >= gormlogger.Info {
		l.logger().InfoContext(ctx, fmt.Sprintf(msg, args...))
	}
}

func (l *Logger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.Config.LogLevel <= gormlogger.Silent {
		return
	}
	latency := time.Since(begin)
	if err != nil && (!l.Config.IgnoreRecordNotFoundError || !errors.Is(err, gorm.ErrRecordNotFound)) && l.Config.LogLevel >= gormlogger.Warn {
		if sql, rows := fc(); rows == -1 {
			l.logger().WarnContext(ctx, "Failed", slog.String("error", err.Error()), slog.Duration("latency", latency), slog.String("sql", sql))
		} else {
			l.logger().WarnContext(ctx, "Failed", slog.String("error", err.Error()), slog.Duration("latency", latency), slog.Int64("rows", rows), slog.String("sql", sql))
		}
	} else if l.Config.SlowThreshold != 0 && latency > l.Config.SlowThreshold && l.Config.LogLevel >= gormlogger.Info {
		if sql, rows := fc(); rows == -1 {
			l.logger().InfoContext(ctx, "Slowed", slog.Duration("latency", latency), slog.String("sql", sql))
		} else {
			l.logger().InfoContext(ctx, "Slowed", slog.Duration("latency", latency), slog.Int64("rows", rows), slog.String("sql", sql))
		}
	} else if l.Config.LogLevel >= gormlogger.Info {
		if sql, rows := fc(); rows == -1 {
			l.logger().DebugContext(ctx, "Traced", slog.Duration("latency", latency), slog.String("sql", sql))
		} else {
			l.logger().DebugContext(ctx, "Traced", slog.Duration("latency", latency), slog.Int64("rows", rows), slog.String("sql", sql))
		}
	}
}

var _ gormlogger.Interface = &Logger{}

func (l *Logger) ParamsFilter(ctx context.Context, sql string, params ...interface{}) (string, []interface{}) {
	if l.Config.ParameterizedQueries {
		return sql, nil
	}
	return sql, params
}

var _ gorm.ParamsFilter = &Logger{}
