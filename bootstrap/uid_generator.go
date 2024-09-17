package bootstrap

import (
	"time"

	"github.com/sony/sonyflake"
)

func newUIDGenerator(startTime string, machineID uint16) (*sonyflake.Sonyflake, error) {
	start, err := time.Parse("2006-01-02", startTime)
	if err != nil {
		return nil, err
	}

	settings := sonyflake.Settings{
		StartTime: start,
		MachineID: func() (uint16, error) {
			return machineID, nil
		},
	}

	return sonyflake.New(settings)
}
