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
		LogLevel             gormlogger.LogLevel
		ParameterizedQueries bool

		ErrorLevel                slog.Level
		WarnLevel                 slog.Level
		InfoLevel                 slog.Level
		IgnoreRecordNotFoundError bool
		FailedLevel               slog.Level
		SlowThreshold             time.Duration
		SlowedLevel               slog.Level
		TracedLevel               slog.Level
	}
	Logger struct {
		Config
		*slog.Logger
	}
)

func New(log *slog.Logger, cfg *Config) *Logger { return &Logger{Logger: log, Config: *cfg} }

var (
	Default = New(nil, &Config{
		LogLevel:                  gormlogger.Info,
		ParameterizedQueries:      false,
		IgnoreRecordNotFoundError: true,
		SlowThreshold:             time.Second,

		ErrorLevel:  slog.LevelError,
		WarnLevel:   slog.LevelWarn,
		InfoLevel:   slog.LevelInfo,
		FailedLevel: slog.LevelWarn,
		SlowedLevel: slog.LevelDebug,
		TracedLevel: slog.LevelDebug,
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
		l.logger().LogAttrs(ctx, l.Config.ErrorLevel, fmt.Sprintf(msg, args...))
	}
}

func (l *Logger) Warn(ctx context.Context, msg string, args ...interface{}) {
	if l.Config.LogLevel >= gormlogger.Warn {
		l.logger().LogAttrs(ctx, l.Config.WarnLevel, fmt.Sprintf(msg, args...))
	}
}

func (l *Logger) Info(ctx context.Context, msg string, args ...interface{}) {
	if l.Config.LogLevel >= gormlogger.Info {
		l.logger().LogAttrs(ctx, l.Config.InfoLevel, fmt.Sprintf(msg, args...))
	}
}

func (l *Logger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.Config.LogLevel <= gormlogger.Silent {
		return
	}
	latency := time.Since(begin)
	if err != nil && (!l.Config.IgnoreRecordNotFoundError || !errors.Is(err, gorm.ErrRecordNotFound)) && l.Config.LogLevel >= gormlogger.Error {
		if sql, rows := fc(); rows == -1 {
			l.logger().LogAttrs(ctx, l.Config.FailedLevel, "Failed", slog.String("error", err.Error()), slog.Duration("latency", latency), slog.String("sql", sql))
		} else {
			l.logger().LogAttrs(ctx, l.Config.FailedLevel, "Failed", slog.String("error", err.Error()), slog.Duration("latency", latency), slog.Int64("rows", rows), slog.String("sql", sql))
		}
	} else if l.Config.SlowThreshold != 0 && latency > l.Config.SlowThreshold && l.Config.LogLevel >= gormlogger.Warn {
		if sql, rows := fc(); rows == -1 {
			l.logger().LogAttrs(ctx, l.Config.SlowedLevel, "Slowed", slog.Duration("latency", latency), slog.String("sql", sql))
		} else {
			l.logger().LogAttrs(ctx, l.Config.SlowedLevel, "Slowed", slog.Duration("latency", latency), slog.Int64("rows", rows), slog.String("sql", sql))
		}
	} else if l.Config.LogLevel >= gormlogger.Info {
		if sql, rows := fc(); rows == -1 {
			l.logger().LogAttrs(ctx, l.Config.TracedLevel, "Traced", slog.Duration("latency", latency), slog.String("sql", sql))
		} else {
			l.logger().LogAttrs(ctx, l.Config.TracedLevel, "Traced", slog.Duration("latency", latency), slog.Int64("rows", rows), slog.String("sql", sql))
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
