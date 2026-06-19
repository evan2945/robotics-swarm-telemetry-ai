package simulation

import (
	"math"
	"math/rand"
	"swarm-engine/internal/machine"
	"time"
)

type Engine struct {
	Height                 float64
	Width                  float64
	Ticker                 *time.Ticker
	Machines               []*machine.MachineState
	TimeSinceLastBroadcast float64
	Items                  []machine.WorldItem
}

func NewEngine(machines []*machine.MachineState, secondsSinceLastBroadcast float64) *Engine {
	e := &Engine{
		Height:                 600.00,
		Width:                  800.00,
		Ticker:                 time.NewTicker(time.Second / 60),
		Machines:               machines,
		TimeSinceLastBroadcast: secondsSinceLastBroadcast,
		Items:                  []machine.WorldItem{},
	}

	for _, m := range e.Machines {
		m.Env = e
	}

	return e
}

func (e *Engine) Tick(dt float64) {
	e.TimeSinceLastBroadcast += dt

	isBeaconFrame := e.TimeSinceLastBroadcast >= 1.0
	if isBeaconFrame {
		e.TimeSinceLastBroadcast = 0.0
	}

	for _, m := range e.Machines {
		mVx, mVy := m.GetHeading()
		mX, mY := m.GetPosition()
		mTargetX, mTargetY := m.GetTarget()
		locBlueprint := m.GetBlueprint().Locomotion
		mState := m.GetState()
		mSpeed := locBlueprint.Speed

		if mState == "LOW_POWER" {
			mSpeed = locBlueprint.LowPowerSpeed

			if getEuclDistance(mX, mY, mTargetX, mTargetY) > 5.0 {
				// If the machine is in low power mode, it needs to head to
				// the closest charging station.
				knownStations := m.GetKnownStations()
				if len(knownStations) > 0 {
					dist := -1.0
					var targetStation *machine.StationData
					for _, station := range knownStations {
						newDist := getEuclDistance(mX, mY, station.X, station.Y)
						if dist == -1 || newDist < dist {
							dist = newDist
							targetStation = station
						}
					}

					m.SetTarget(targetStation.X, targetStation.Y)
				}
			}
		}

		mVx, mVy = machine.CalculateHeading(mX, mY, mTargetX, mTargetY, mSpeed)
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

		distance := getEuclDistance(mX, mY, mTargetX, mTargetY)

		if mState != "LOW_POWER" {
			if distance < 5.0 {
				m.SetTarget(rand.Float64()*e.Width, rand.Float64()*e.Height)
			} else {
				m.SetTarget(mTargetX, mTargetY)
			}
		}

		m.SetPosition(mX, mY)
		m.SetHeading(mVx, mVy)

	DrainingOutbox:
		for {
			select {
			case packet := <-m.Outbox:
				e.broadcastPacket(m, packet)
			default:
				break DrainingOutbox
			}
		}

		if isBeaconFrame {
			beacons := m.GetActiveBeacons()
			for _, packet := range beacons {
				e.broadcastPacket(m, packet)
			}
		}
	}
}

// broadcastPacket acts as the broadcaster of messages. Since in the simulation there really
// isn't a way for the machines to communicate via broadcast, the engine will act as the
// broadcaster and "medium" through which the broadcast travels to keep things simple.
func (e *Engine) broadcastPacket(sender *machine.MachineState, packet machine.GossipPacket) {
	senderX, senderY := sender.GetPosition()
	commBlueprint := sender.GetBlueprint().Communication
	senderState := sender.GetState()
	senderRange := commBlueprint.Range

	if senderState == "LOW_POWER" {
		senderRange = commBlueprint.LowPowerRange
	}

	for _, receiver := range e.Machines {
		if receiver.GetId() == sender.GetId() {
			continue
		}

		if receiver.HasSeenPacket(packet.ID) {
			continue
		}

		receiverX, receiverY := receiver.GetPosition()
		euclideanDist := getEuclDistance(senderX, senderY, receiverX, receiverY)
		if euclideanDist <= senderRange {
			select {
			case receiver.Inbox <- packet:
			default:
			}
		}
	}
}

// A machine will call this method to see if anything in the environment is within
// visual range.
func (e *Engine) GetVisibleItems(x, y, visionRange float64) []machine.WorldItem {
	visibleItems := []machine.WorldItem{}

	for _, item := range e.Items {
		eucl_distance := getEuclDistance(x, y, item.X, item.Y)

		if eucl_distance <= visionRange {
			visibleItems = append(visibleItems, item)
		}
	}

	return visibleItems
}

// Similar to the above method, this will be called to see if other machiens are within visual range.
func (e *Engine) GetVisibleMachines(machineId string, x, y, visionRange float64) []*machine.MachineState {
	visiblePeers := []*machine.MachineState{}

	for _, peer := range e.Machines {
		if peer.ID == machineId {
			continue
		}

		pX, pY := peer.GetPosition()
		if getEuclDistance(x, y, pX, pY) <= visionRange {
			visiblePeers = append(visiblePeers, peer)
		}
	}

	return visiblePeers
}

func getEuclDistance(x1, y1, x2, y2 float64) float64 {
	return math.Sqrt(math.Pow(x2-x1, 2) + math.Pow(y2-y1, 2))
}
