package machine

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateDistance(t *testing.T) {
	var testCases = []struct {
		description   string
		x1            float64
		x2            float64
		y1            float64
		y2            float64
		expected_dist float64
	}{
		{
			description:   "Returns 0.0 if already at target location",
			x1:            2.0,
			y1:            4.6,
			x2:            2.0,
			y2:            4.6,
			expected_dist: 0.0,
		},
		{
			description:   "Calculates appropriate distance if not already at target location",
			x1:            1.0,
			y1:            2.0,
			x2:            4.0,
			y2:            6.0,
			expected_dist: 5.0,
		},
		{
			description:   "Calculates distance with negative coordinates properly",
			x1:            -5.0,
			y1:            2.0,
			x2:            -2.0,
			y2:            6.0,
			expected_dist: 5.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			actual_dist := CalculateDistance(tc.x1, tc.y1, tc.x2, tc.y2)
			assert.LessOrEqual(t, math.Abs(actual_dist-tc.expected_dist), 0.000001)
		})
	}
}

func TestCalculateHeading(t *testing.T) {
	testCases := []struct {
		description string
		currentX    float64
		currentY    float64
		targetX     float64
		targetY     float64
		speed       float64
		expectedVx  float64
		expectedVy  float64
	}{
		{
			description: "Returns 0.0 for both vx and vy if the target is the current position",
			currentX:    1.0,
			currentY:    3.0,
			targetX:     1.0,
			targetY:     3.0,
			speed:       5.0,
			expectedVx:  0.0,
			expectedVy:  0.0,
		},
		{
			description: "Returns proper speed components for pure horizontal movement",
			currentX:    1.0,
			currentY:    3.0,
			targetX:     8.0,
			targetY:     3.0,
			speed:       5.0,
			expectedVx:  5.0,
			expectedVy:  0.0,
		},
		{
			description: "Returns proper speed components for pure vertical movement",
			currentX:    1.0,
			currentY:    3.0,
			targetX:     1.0,
			targetY:     9.0,
			speed:       5.0,
			expectedVx:  0.0,
			expectedVy:  5.0,
		},
		{
			description: "Returns proper speed components for diagnonal movement",
			currentX:    1.0,
			currentY:    3.0,
			targetX:     5.0,
			targetY:     7.0,
			speed:       5.0,
			expectedVx:  5.0 * math.Cos(45.0*(math.Pi/180)),
			expectedVy:  5.0 * math.Sin(45.0*(math.Pi/180)),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			vx, vy := CalculateHeading(tc.currentX, tc.currentY, tc.targetX, tc.targetY, tc.speed)
			assert.LessOrEqual(t, math.Abs(vx-tc.expectedVx), 0.000001)
			assert.LessOrEqual(t, math.Abs(vy-tc.expectedVy), 0.000001)
		})
	}
}
