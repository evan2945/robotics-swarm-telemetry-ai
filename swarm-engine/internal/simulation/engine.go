package simulation

import (
	"math"
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
		mVx, mVy := m.GetHeading()
		mX, mY := m.GetPosition()
		mTargetX, mTargetY := m.GetTarget()
		mBlueprint := m.GetBlueprint()

		mVx, mVy = machine.CalculateHeading(mX, mY, mTargetX, mTargetY, mBlueprint.Locomotion.Speed)
		mX = mX + (mVx * dt)
		mY = mY + (mVy * dt)

		if mX <= 0 {
			mX = 0
			mVx = -mVx
			mTargetX = -mTargetX
		} else if mX >= e.Width {
			mX = e.Width
			mVx = -mVx
			mTargetX = e.Width - (mTargetX - e.Width)
		}

		if mY <= 0 {
			mY = 0
			mVy = -mVy
			mTargetY = -mTargetY
		} else if mY >= e.Height {
			mY = e.Height
			mVy = -mVy
			mTargetY = e.Height - (mTargetY - e.Height)
		}

		m.SetPosition(mX, mY)
		m.SetHeading(mVx, mVy)
		m.SetTarget(mTargetX, mTargetY)

	DrainingOutbox:
		for {
			select {
			case packet := <-m.Outbox:
				e.broadcastPacket(m, packet)
			default:
				break DrainingOutbox
			}
		}
	}
}

func (e *Engine) broadcastPacket(sender *machine.MachineState, packet machine.GossipPacket) {
	senderX, senderY := sender.GetPosition()
	senderRange := sender.GetBlueprint().Communication.Range

	for _, receiver := range e.Machines {
		if receiver.GetId() == sender.GetId() {
			continue
		}

		if receiver.HasSeenPacket(packet.ID) {
			continue
		}

		receiverX, receiverY := receiver.GetPosition()
		euclideanDist := math.Sqrt(math.Pow(receiverX-senderX, 2) + math.Pow(receiverY-senderY, 2))
		if euclideanDist <= senderRange {
			select {
			case receiver.Inbox <- packet:
			default:
			}
		}
	}
}
