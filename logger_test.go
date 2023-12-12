package sloggorm

import (
	"github.com/sagikazarmark/slog-shim"
	"gorm.io/gorm"
)

func Example() {
	tx, _ := gorm.Open(nil, &gorm.Config{Logger: NewLogger(slog.Default(), NewConfig())})
	// do something with tx
	_ = tx
}
