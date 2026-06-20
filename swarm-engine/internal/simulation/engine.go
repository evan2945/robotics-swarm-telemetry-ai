package simulation

import (
	"math"
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
		if m.GetBatteryLevel() <= 0 {
			continue
		}

		mState := m.GetState()

		if action, exists := machine.KinematicRegistry[mState]; exists {
			action(m, machine.KinematicActionParams{
				Dt:           dt,
				ScreenWidth:  e.Width,
				ScreenHeight: e.Height,
			})
		}

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
