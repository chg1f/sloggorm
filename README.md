## sloggorm

## Intro

slog logging driver for gorm2

## Usage

```
import (
  sloggorm "github.com/chg1f/sloggorm"
  "gorm.io/gorm"
)

func main() {
	tx, _ := gorm.Open(nil, &gorm.Config{Logger: sloggorm.NewLogger(slog.Default(), sloggorm.NewConfig())})
}
```
