package util

import (
	"time"
)

func Backoff(attempt int) time.Duration {
	if attempt < 0 { attempt = 0 }
	d := 500 * time.Millisecond
	for i := 0; i < attempt && i < 6; i++ {
		d *= 2
	}
	return d
}