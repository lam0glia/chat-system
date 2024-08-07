package bootstrap

import (
	"time"

	"github.com/sony/sonyflake"
)

func newUIDGenerator(startTime string) (*sonyflake.Sonyflake, error) {
	start, err := time.Parse("2006-01-02", startTime)
	if err != nil {
		return nil, err
	}

	settings := sonyflake.Settings{
		StartTime: start,
	}

	return sonyflake.New(settings)
}
