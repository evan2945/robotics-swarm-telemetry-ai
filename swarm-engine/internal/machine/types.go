package machine

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// BatteryBlueprint defines power characteristics for a machine class.
type BatteryBlueprint struct {
	InitialLevel  float64 `json:"initial_level"`
	DrainRate     float64 `json:"drain_rate"`
	ChargeRate    float64 `json:"charge_rate"`
	LowPowerDrain float64 `json:"low_power_drain"`
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
	PayloadResource PayloadType = "RESOURCE"
)

type ResourceType string

const (
	ResourceStation ResourceType = "STATION"
)

type ResourceData struct {
	ID   string       `json:"id"`
	Type ResourceType `json:"type"`
	X    float64      `json:"x"`
	Y    float64      `json:"y"`
}

type PacketPayload struct {
	Type     PayloadType   `json:"type"`
	Resource *ResourceData `json:"resource,omitempty"`
}

type StationData struct {
	DiscoveredAt time.Time `json:"discovered_at"`
	X            float64   `json:"x"`
	Y            float64   `json:"y"`
}

type MachineState struct {
	mu             sync.Mutex
	Inbox          chan GossipPacket
	Outbox         chan GossipPacket
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	State          string                 `json:"state"`
	X              float64                `json:"x"`
	Y              float64                `json:"y"`
	TargetX        float64                `json:"target_x"`
	TargetY        float64                `json:"target_y"`
	Vx             float64                `json:"v_x"`
	Vy             float64                `json:"v_y"`
	BatteryLevel   float64                `json:"battery_level"`
	Blueprint      MachineBlueprint       `json:"blueprint"`
	TickerInterval time.Duration          `json:"ticker_interval"`
	SeenPackets    map[string]bool        `json:"seen_packets"`
	Env            Environment            `json:"environment"`
	KnownStations  map[string]StationData `json:"known_stations"`
}

type WorldItem struct {
	ID   string  `json:"id"`
	Type string  `json:"type"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
}

type Environment interface {
	GetVisibleItems(x, y float64, visionRange float64) []WorldItem
}

func (m *MachineState) RunLifecycle(ctx context.Context) {
	ticker := time.NewTicker(m.GetTickerInterval())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			battery_level := m.GetBatteryLevel()
			blueprint := m.GetBlueprint()
			m.SetBatteryLevel(battery_level - blueprint.Battery.DrainRate)
			fmt.Printf("Machine %s battery: %f\n", m.GetName(), m.GetBatteryLevel())

			seenItems := []WorldItem{}
			if m.Env != nil {
				seenItems = m.Env.GetVisibleItems(m.X, m.Y, blueprint.Vision.Range)
			}

			knownStations := m.GetKnownStations()
			for _, item := range seenItems {
				now := time.Now()
				if _, ok := knownStations[item.ID]; !ok {
					m.AddToKnownStations(item)
					m.Outbox <- GossipPacket{
						ID: fmt.Sprintf("%s:%s:%s", item.ID, m.GetId(), now),
					}
				} else {
					m.UpdateKnownStationsTime(item.ID, now)
				}
			}

		case packet := <-m.Inbox:
			fmt.Printf("[%s] Received Gossip Packet: %s from %s\n", m.GetName(), packet.ID, packet.OriginId)

			if !m.HasSeenPacket(packet.ID) {
				m.AddSeenPacket(packet.ID)

				if packet.Payload.Resource.Type == ResourceStation {
					knownStations := m.GetKnownStations()
					resource := packet.Payload.Resource
					if _, ok := knownStations[resource.ID]; !ok {
						m.AddToKnownStations(WorldItem{
							ID:   resource.ID,
							Type: string(resource.Type),
							X:    resource.X,
							Y:    resource.Y,
						})
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

func (m *MachineState) GetState() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.State
}

func (m *MachineState) SetState(newState string) {
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

func (m *MachineState) GetKnownStations() map[string]StationData {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Janitor phase
	for id, data := range m.KnownStations {
		if time.Since(data.DiscoveredAt) > 10*time.Second {
			delete(m.KnownStations, id)
		}
	}

	return m.KnownStations
}

func (m *MachineState) AddToKnownStations(item WorldItem) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.KnownStations[item.ID] = StationData{
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
}

func (m *MachineState) GetActiveBeacons() []GossipPacket {
	m.mu.Lock()
	defer m.mu.Unlock()

	var packets []GossipPacket
	for stationId, stationData := range m.KnownStations {
		if time.Since(stationData.DiscoveredAt) < 10*time.Second {
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
