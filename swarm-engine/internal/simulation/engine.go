package simulation

import (
	"swarm-engine/internal/machine"
	"time"
)

type Engine struct {
	Height   float64
	Width    float64
	Ticker   *time.Ticker
	Machines []*machine.MachineState
}

func NewEngine(machines []*machine.MachineState) *Engine {
	return &Engine{
		Height:   600.00,
		Width:    800.00,
		Ticker:   time.NewTicker(time.Second / 60),
		Machines: machines,
	}
}

func (e *Engine) Tick(dt float64) {
	for _, m := range e.Machines {
		m.Vx, m.Vy = machine.CalculateHeading(m.X, m.Y, m.TargetX, m.TargetY, m.Blueprint.Locomotion.Speed)
		m.X = m.X + (m.Vx * dt)
		m.Y = m.Y + (m.Vy * dt)

		if m.X <= 0 {
			m.X = 0
			m.Vx = -m.Vx
			m.TargetX = -m.TargetX
		} else if m.X >= e.Width {
			m.X = e.Width
			m.Vx = -m.Vx
			m.TargetX = e.Width - (m.TargetX - e.Width)
		}

		if m.Y <= 0 {
			m.Y = 0
			m.Vy = -m.Vy
			m.TargetY = -m.TargetY
		} else if m.Y >= e.Height {
			m.Y = e.Height
			m.Vy = -m.Vy
			m.TargetY = e.Height - (m.TargetY - e.Height)
		}
	}
}
