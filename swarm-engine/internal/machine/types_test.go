package machine

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRunLifecycle(t *testing.T) {
	mockMachine := &MachineState{
		ID:             "machine-01",
		Name:           "mock machine",
		State:          "PATROL",
		X:              100,
		Y:              100,
		TargetX:        100,
		TargetY:        100,
		BatteryLevel:   100,
		TickerInterval: time.Millisecond,
		Blueprint: MachineBlueprint{
			Battery: BatteryBlueprint{
				DrainRate: 1.0,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	go mockMachine.RunLifecycle(ctx)

	<-ctx.Done()

	assert.Equal(t, mockMachine.BatteryLevel, 95.0)
}
