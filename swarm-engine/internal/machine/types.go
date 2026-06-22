package machine

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

const StationMemoryTTL = 10 * time.Second
const HeartbeatMemoryTTL = 4 * time.Second

type State string

const (
	StatePatrol     State = "PATROL"
	StateCharging   State = "CHARGING"
	StateLowPower   State = "LOW_POWER"
	StateDead       State = "DEAD"
	StateIdle       State = "IDLE"
	StateRecharging State = "RECHARGING"
	StateRescuing   State = "RESCUING"
)

type StateBlueprint struct {
	InitialState     State   `json:"initial_state"`
	AdditionalStates []State `json:"additional_states"`
}

// BatteryBlueprint defines power characteristics for a machine class.
type BatteryBlueprint struct {
	InitialLevel        float64 `json:"initial_level"`
	DrainRate           float64 `json:"drain_rate"`
	ChargeRate          float64 `json:"charge_rate"`
	LowPowerDrain       float64 `json:"low_power_drain"`
	RechargingDrainRate float64 `json:"recharging_drain_rate"`
	TransferEfficiency  float64 `json:"transfer_efficiency"`
}

// LocomotionBlueprint defines kinematic speed constraints.
type LocomotionBlueprint struct {
	Speed            float64 `json:"speed"`
	LowPowerSpeed    float64 `json:"low_power_speed"`
	EvasionThreshold float64 `json:"evasion_threshold"`
}

// VisionBlueprint models the physical hardware sector depth and arc.
type VisionBlueprint struct {
	Range        float64 `json:"range"`
	AngleDegrees float64 `json:"angle_degrees"`
}

// CommunicationBlueprint dictates the mesh routing distances.
type CommunicationBlueprint struct {
	Range         float64 `json:"range"`
	LowPowerRange float64 `json:"low_power_range"`
}

// MachineBlueprint represents the unified specification schema.
type MachineBlueprint struct {
	States        StateBlueprint         `json:"states"`
	Battery       BatteryBlueprint       `json:"battery"`
	Locomotion    LocomotionBlueprint    `json:"locomotion"`
	Vision        VisionBlueprint        `json:"vision"`
	Communication CommunicationBlueprint `json:"communication"`
}

// CauldronRegistry maps machine names to their static blueprints.
type CauldronRegistry map[string]MachineBlueprint

type GossipPacket struct {
	ID       string
	OriginId string
	Payload  PacketPayload
	TTL      int
}

type PayloadType string

const (
	PayloadResource  PayloadType = "RESOURCE"
	PayloadHeartbeat PayloadType = "HEARTBEAT"
)

type ResourceType string

const (
	ResourceStation ResourceType = "STATION"
)

type HeartbeatType string

const (
	HeartbeatRequest  HeartbeatType = "HEARTBEAT REQUEST"
	HeartbeatResponse HeartbeatType = "HEARTBEAT RESPONSE"
)

type ResourceData struct {
	ID   string       `json:"id"`
	Type ResourceType `json:"type"`
	X    float64      `json:"x"`
	Y    float64      `json:"y"`
}

type HeartbeatData struct {
	ID           string        `json:"id"`
	Type         HeartbeatType `json:"type"`
	X            float64       `json:"x"`
	Y            float64       `json:"y"`
	BatteryLevel float64       `json:"battery_level"`
}

type PacketPayload struct {
	Type      PayloadType    `json:"type"`
	Resource  *ResourceData  `json:"resource,omitempty"`
	Heartbeat *HeartbeatData `json:"heartbeat,omitempty"`
}

type StationData struct {
	DiscoveredAt time.Time `json:"discovered_at"`
	X            float64   `json:"x"`
	Y            float64   `json:"y"`
}

type PendingPing struct {
	LastPingedAt  time.Time `json:"last_pinged_at"`
	LastKnownX    float64   `json:"last_known_x"`
	LastKnownY    float64   `json:"last_known_y"`
	HeardResponse bool      `json:"heard_response"`
}

type MachineState struct {
	mu                sync.Mutex
	Inbox             chan GossipPacket
	Outbox            chan GossipPacket
	ID                string                  `json:"id"`
	Name              string                  `json:"name"`
	State             State                   `json:"state"`
	X                 float64                 `json:"x"`
	Y                 float64                 `json:"y"`
	TargetX           float64                 `json:"target_x"`
	TargetY           float64                 `json:"target_y"`
	Vx                float64                 `json:"v_x"`
	Vy                float64                 `json:"v_y"`
	BatteryLevel      float64                 `json:"battery_level"`
	Blueprint         MachineBlueprint        `json:"blueprint"`
	TickerInterval    time.Duration           `json:"ticker_interval"`
	SeenPackets       map[string]bool         `json:"seen_packets"`
	Env               Environment             `json:"environment"`
	KnownStations     map[string]*StationData `json:"known_stations"`
	PendingHeartbeats map[string]PendingPing  `json:"pending_heartbeats"`
	StationTargeted   bool                    `json:"station_targeted"`
}

type WorldItem struct {
	ID   string       `json:"id"`
	Type ResourceType `json:"type"`
	X    float64      `json:"x"`
	Y    float64      `json:"y"`
}

type Environment interface {
	GetVisibleItems(x, y, visionRange float64) []WorldItem
	GetVisibleMachines(machineId string, x, y, visionRange float64) []*MachineState
}

func (m *MachineState) RunLifecycle(ctx context.Context) {
	ticker := time.NewTicker(m.GetTickerInterval())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			batteryLevel := m.GetBatteryLevel()
			blueprint := m.GetBlueprint()

			if batteryLevel <= 0 {
				if m.GetState() != StateDead {
					fmt.Printf("[%s] 💀 CRITICAL: Battery fully depleted. Unit deactivated.\n", m.GetName())
					m.SetBatteryLevel(0.0)
					m.SetState(StateDead)
				}
				continue
			}

			switch m.GetState() {
			case StateCharging:
				m.SetBatteryLevel(batteryLevel + blueprint.Battery.ChargeRate)
			case StateLowPower:
				m.SetBatteryLevel(batteryLevel - blueprint.Battery.LowPowerDrain)
			case StateRecharging:
				m.SetBatteryLevel(batteryLevel - blueprint.Battery.RechargingDrainRate)
			case StateIdle:
			default:
				m.SetBatteryLevel(batteryLevel - blueprint.Battery.DrainRate)
			}

			batteryLevel = m.GetBatteryLevel()
			fmt.Printf("Machine %s battery: %f\n", m.GetName(), batteryLevel)

			seenItems := []WorldItem{}
			if m.Env != nil {
				seenItems = m.Env.GetVisibleItems(m.X, m.Y, blueprint.Vision.Range)
			}

			knownStations := m.GetKnownStations()
			for _, item := range seenItems {
				now := time.Now()
				if _, ok := knownStations[item.ID]; !ok {
					fmt.Printf("[%s] has seen world item %s!\n", m.GetName(), item.Type)
					m.AddToKnownStations(item)
					m.Outbox <- GossipPacket{
						ID:       fmt.Sprintf("%s:%s:%s", m.GetId(), item.ID, time.Now()),
						OriginId: m.ID,
						Payload: PacketPayload{
							Type: PayloadResource,
							Resource: &ResourceData{
								ID:   item.ID,
								Type: item.Type,
								X:    item.X,
								Y:    item.Y,
							},
						},
						TTL: 2,
					}
				} else {
					m.UpdateKnownStationsTime(item.ID, now)
				}
			}

			seenMachines := []*MachineState{}
			if m.Env != nil {
				seenMachines = m.Env.GetVisibleMachines(m.ID, m.X, m.Y, blueprint.Vision.Range)
			}

			mX, mY := m.GetPosition()
			for _, targetMachine := range seenMachines {
				if _, ok := m.GetPendingHeartbeats()[targetMachine.ID]; !ok {
					heartbeatRequest := PendingPing{
						LastPingedAt:  time.Now(),
						LastKnownX:    mX,
						LastKnownY:    mY,
						HeardResponse: false,
					}
					m.SetPendingHeartbeat(targetMachine.ID, heartbeatRequest)
					mId := m.GetId()
					fmt.Printf("[%s] Sending heartbeat request for peer: %s\n", m.GetName(), targetMachine.GetId())
					m.Outbox <- GossipPacket{
						ID:       fmt.Sprintf("%s:%s:%s", mId, targetMachine.GetId(), time.Now()),
						OriginId: mId,
						Payload: PacketPayload{
							Type: PayloadHeartbeat,
							Heartbeat: &HeartbeatData{
								ID:           mId,
								Type:         HeartbeatRequest,
								X:            mX,
								Y:            mY,
								BatteryLevel: batteryLevel,
							},
						},
						TTL: 1,
					}
				}
			}

			m.GcPendingHeartbeats()

		case packet := <-m.Inbox:
			if !m.HasSeenPacket(packet.ID) {
				m.AddSeenPacket(packet.ID)

				switch packet.Payload.Type {
				case PayloadResource:
					if packet.Payload.Resource != nil && packet.Payload.Resource.Type == ResourceStation {
						knownStations := m.GetKnownStations()
						resource := packet.Payload.Resource
						if _, ok := knownStations[resource.ID]; !ok {
							fmt.Printf("[%s] Received Gossip Packet: Saw %s from %s\n", m.GetName(), packet.Payload.Resource.Type, packet.OriginId)
							m.AddToKnownStations(WorldItem{
								ID:   resource.ID,
								Type: resource.Type,
								X:    resource.X,
								Y:    resource.Y,
							})
						}
					}
				case PayloadHeartbeat:
					if packet.Payload.Heartbeat != nil {
						heartbeat := packet.Payload.Heartbeat
						mId := m.GetId()
						mX, mY := m.GetPosition()
						batteryLevel := m.GetBatteryLevel()

						switch heartbeat.Type {
						// Another machine saw this machine, and is requesting a heartbeat response
						case HeartbeatRequest:
							fmt.Printf("[%s] Heartbeat request receieved from peer: %s\n", m.Name, packet.OriginId)
							m.Outbox <- GossipPacket{
								ID:       fmt.Sprintf("%s:%s:%s", mId, packet.OriginId, time.Now()),
								OriginId: mId,
								Payload: PacketPayload{
									Type: PayloadHeartbeat,
									Heartbeat: &HeartbeatData{
										ID:           mId,
										Type:         HeartbeatResponse,
										X:            mX,
										Y:            mY,
										BatteryLevel: batteryLevel,
									},
								},
								TTL: 1,
							}
						// We are getting back a heartbeat response, so just update the pending request to indicate
						// that we heard back from the machine.
						case HeartbeatResponse:
							m.UpdatePendingHeartbeat(packet.OriginId, true)
							fmt.Printf("[%s] Heartbeat confirmed for peer: %s\n", m.Name, packet.OriginId)
						}
					}
				}
				packet.TTL -= 1
				if packet.TTL > 0 {
					m.QueuePacketToSend(packet)
				}
			}
		}
	}
}

func (m *MachineState) GetPosition() (float64, float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.X, m.Y
}

func (m *MachineState) SetPosition(newX, newY float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.X = newX
	m.Y = newY
}

func (m *MachineState) GetHeading() (float64, float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.Vx, m.Vy
}

func (m *MachineState) SetHeading(newVx, newVy float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Vx = newVx
	m.Vy = newVy
}

func (m *MachineState) GetTarget() (float64, float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.TargetX, m.TargetY
}

func (m *MachineState) SetTarget(newTargetX, newTargetY float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TargetX = newTargetX
	m.TargetY = newTargetY
}

func (m *MachineState) GetBatteryLevel() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.BatteryLevel
}

func (m *MachineState) SetBatteryLevel(new_level float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.BatteryLevel = new_level
}

func (m *MachineState) GetState() State {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.State
}

func (m *MachineState) SetState(newState State) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.State = newState
}

func (m *MachineState) GetName() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.Name
}

func (m *MachineState) GetId() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.ID
}

func (m *MachineState) GetBlueprint() MachineBlueprint {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.Blueprint
}

func (m *MachineState) GetTickerInterval() time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.TickerInterval
}

func (m *MachineState) HasSeenPacket(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.SeenPackets[id]
}

func (m *MachineState) AddSeenPacket(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SeenPackets[id] = true
}

func (m *MachineState) GetSeenPackets() map[string]bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.SeenPackets
}

func (m *MachineState) QueuePacketToSend(packet GossipPacket) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Outbox <- packet
}

func (m *MachineState) GetKnownStations() map[string]*StationData {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Janitor phase
	for id, data := range m.KnownStations {
		if time.Since(data.DiscoveredAt) > StationMemoryTTL {
			fmt.Printf("[%s] Station has expired out of memory.\n", m.Name)
			delete(m.KnownStations, id)
		}
	}

	return m.KnownStations
}

func (m *MachineState) AddToKnownStations(item WorldItem) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.KnownStations[item.ID] = &StationData{
		DiscoveredAt: time.Now(),
		X:            item.X,
		Y:            item.Y,
	}
}

func (m *MachineState) UpdateKnownStationsTime(id string, t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	station := m.KnownStations[id]
	station.DiscoveredAt = t

	m.KnownStations[id] = station
}

func (m *MachineState) SetStationTargeted(targeted bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.StationTargeted = targeted
}

func (m *MachineState) GetStationTargeted() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.StationTargeted
}

func (m *MachineState) GetActiveBeacons() []GossipPacket {
	m.mu.Lock()
	defer m.mu.Unlock()

	var packets []GossipPacket
	for stationId, stationData := range m.KnownStations {
		if time.Since(stationData.DiscoveredAt) < StationMemoryTTL {
			packets = append(packets, GossipPacket{
				ID:       fmt.Sprintf("%s:%s:%s", m.ID, stationId, stationData.DiscoveredAt),
				OriginId: m.ID,
				Payload: PacketPayload{
					Type: PayloadResource,
					Resource: &ResourceData{
						ID:   stationId,
						Type: ResourceStation,
						X:    stationData.X,
						Y:    stationData.Y,
					},
				},
				TTL: 2,
			})
		}
	}

	return packets
}

func (m *MachineState) GetPendingHeartbeats() map[string]PendingPing {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.PendingHeartbeats
}

func (m *MachineState) SetPendingHeartbeat(targetMachineId string, pendingHeartbeat PendingPing) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.PendingHeartbeats[targetMachineId] = pendingHeartbeat
}

func (m *MachineState) UpdatePendingHeartbeat(targetMachineId string, heardResponse bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ping, ok := m.PendingHeartbeats[targetMachineId]; ok {
		ping.HeardResponse = heardResponse
		m.PendingHeartbeats[targetMachineId] = ping
	}
}

func (m *MachineState) GcPendingHeartbeats() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Janitor phase
	for id, ping := range m.PendingHeartbeats {
		if time.Since(ping.LastPingedAt) > HeartbeatMemoryTTL {
			fmt.Printf("[%s] Heartbeat has expired out of memory for %s.\nResponded? %t\n", m.Name, id, ping.HeardResponse)
			delete(m.PendingHeartbeats, id)
		}
	}

}

func getEuclDistance(x1, y1, x2, y2 float64) float64 {
	return math.Sqrt(math.Pow(x2-x1, 2) + math.Pow(y2-y1, 2))
}
