package machine

import (
	"math"
)

func CalculateDistance(x1, y1, x2, y2 float64) float64 {
	dx := x2 - x1
	dy := y2 - y1

	dist := math.Sqrt((dx * dx) + (dy * dy))
	return dist
}

// Calculate the components of velocity (vx, vy) required to travel from a current position toward a
// target position at a given speed.
func CalculateHeading(currentX, currentY, targetX, targetY, speed float64) (vx, vy float64) {
	vx = 0.0
	vy = 0.0

	dist := CalculateDistance(currentX, currentY, targetX, targetY)

	if dist == 0.0 {
		return
	}

	dx := targetX - currentX
	dy := targetY - currentY

	vx = (dx / dist) * speed
	vy = (dy / dist) * speed

	return
}
