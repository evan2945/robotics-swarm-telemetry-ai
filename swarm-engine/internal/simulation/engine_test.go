package simulation

import (
	"swarm-engine/internal/machine"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEngineTick(t *testing.T) {
	testCases := []struct {
		description string
		startX      float64
		startY      float64
		targetX     float64
		targetY     float64
		speed       float64
		dt          float64

		expectedX       float64
		expectedY       float64
		expectedVx      float64
		expectedVy      float64
		expectedTargetX float64
		expectedTargetY float64
	}{
		{
			description: "Calculates proper values for standard movement",
			startX:      100.0,
			startY:      100.0,
			targetX:     200.0,
			targetY:     100.0,
			speed:       10.0,
			dt:          1.0,

			expectedX:       110.0,
			expectedY:       100.0,
			expectedVx:      10.0,
			expectedVy:      0.0,
			expectedTargetX: 200.0,
			expectedTargetY: 100.0,
		},
		{
			description: "Ensures proper state updates in the case of an encounter with a left wall boundary",
			startX:      5.0,
			startY:      100.0,
			targetX:     -50.0,
			targetY:     100.0,
			speed:       -10.0,
			dt:          1.0,

			expectedX:       0.0,
			expectedY:       100.0,
			expectedVx:      10.0,
			expectedVy:      0.0,
			expectedTargetX: 50.0,
			expectedTargetY: 100.0,
		},
		{
			description: "Ensures proper state updates in the case of an encounter with a right wall boundary",
			startX:      795.0,
			startY:      100.0,
			targetX:     850.0,
			targetY:     100.0,
			speed:       10.0,
			dt:          1.0,

			expectedX:       800.0,
			expectedY:       100.0,
			expectedVx:      -10.0,
			expectedVy:      0.0,
			expectedTargetX: 750.0,
			expectedTargetY: 100.0,
		},
		{
			description: "Ensures proper state updates in the case of an encounter with an lower wall boundary",
			startX:      100.0,
			startY:      5.0,
			targetX:     100.0,
			targetY:     -50.0,
			speed:       -10.0,
			dt:          1.0,

			expectedX:       100.0,
			expectedY:       0.0,
			expectedVx:      0.0,
			expectedVy:      10.0,
			expectedTargetX: 100.0,
			expectedTargetY: 50.0,
		},
		{
			description: "Ensures proper state updates in the case of an encounter with an upper wall boundary",
			startX:      100.0,
			startY:      595.0,
			targetX:     100.0,
			targetY:     650.0,
			speed:       10.0,
			dt:          1.0,

			expectedX:       100.0,
			expectedY:       600.0,
			expectedVx:      0.0,
			expectedVy:      -10.0,
			expectedTargetX: 100.0,
			expectedTargetY: 550.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			mock_machine := &machine.MachineState{
				ID:           "machine-01",
				Name:         "mock machine",
				State:        "PATROL",
				X:            tc.startX,
				Y:            tc.startY,
				TargetX:      tc.targetX,
				TargetY:      tc.targetY,
				BatteryLevel: 100,
				Blueprint: machine.MachineBlueprint{
					Locomotion: machine.LocomotionBlueprint{
						Speed: 10,
					},
				},
			}

			engine := NewEngine([]*machine.MachineState{mock_machine})
			engine.Tick(tc.dt)

			assert.Equal(t, tc.expectedX, mock_machine.X)
			assert.Equal(t, tc.expectedY, mock_machine.Y)
			assert.Equal(t, tc.expectedVx, mock_machine.Vx)
			assert.Equal(t, tc.expectedVy, mock_machine.Vy)
			assert.Equal(t, tc.expectedTargetX, mock_machine.TargetX)
			assert.Equal(t, tc.expectedTargetY, mock_machine.TargetY)
		})
	}
}
