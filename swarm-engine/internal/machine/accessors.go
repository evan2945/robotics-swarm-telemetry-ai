package machine

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"time"
)

// Position/Kinematics

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

// Basic properties

func (m *MachineState) GetBatteryLevel() float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.BatteryLevel
}

func (m *MachineState) SetBatteryLevel(newLevel float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.BatteryLevel = newLevel
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

// Packet/Network

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

	copied := make(map[string]bool, len(m.SeenPackets))
	maps.Copy(copied, m.SeenPackets)
	return copied
}

func (m *MachineState) QueuePacketToSend(packet GossipPacket) {
	select {
	case m.Outbox <- packet:
	default:
		fmt.Printf("[%s] Network congestion: Outbox full, dropping packet.\n", m.GetName())
	}
}

// Station

func (m *MachineState) GetKnownStations() map[string]*StationData {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, data := range m.KnownStations {
		if data.ExpiresAfter != -1 && time.Since(data.DiscoveredAt) > time.Duration(data.ExpiresAfter)*time.Second {
			fmt.Printf("[%s] Station has expired out of memory.\n", m.Name)
			delete(m.KnownStations, id)
		}
	}

	copied := make(map[string]*StationData, len(m.KnownStations))
	maps.Copy(copied, m.KnownStations)
	return copied
}

func (m *MachineState) AddToKnownStations(item WorldItem, expiresAfter float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.KnownStations[item.ID] = &StationData{
		DiscoveredAt: time.Now(),
		X:            item.X,
		Y:            item.Y,
		ExpiresAfter: expiresAfter,
	}
}

func (m *MachineState) UpdateKnownStationsTime(id string, t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if station, ok := m.KnownStations[id]; ok {
		m.KnownStations[id] = &StationData{
			DiscoveredAt: t,
			X:            station.X,
			Y:            station.Y,
			ExpiresAfter: station.ExpiresAfter,
		}
	}
}

// Rescue

func (m *MachineState) GetRescueInfo() *RescueData {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.RescueInfo
}

func (m *MachineState) SetRescueInfo(data *RescueData) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.RescueInfo = data
}

func (m *MachineState) GetSosMessages() []*SosData {
	m.mu.Lock()
	defer m.mu.Unlock()

	copied := make([]*SosData, len(m.SosMessages))
	copy(copied, m.SosMessages)
	return copied
}

func (m *MachineState) AddToSosMessages(data *SosData) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SosMessages = append(m.SosMessages, data)
}

func (m *MachineState) GetActiveBeacons() []GossipPacket {
	m.mu.Lock()
	defer m.mu.Unlock()

	var packets []GossipPacket

	// Collect valid station beacons
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

	// Collect valid SOS beacons
	for _, sos := range m.SosMessages {
		if time.Since(sos.DetectedAt) < 20*time.Second {
			packets = append(packets, GossipPacket{
				ID:       fmt.Sprintf("%s:%s", m.ID, time.Now()),
				OriginId: m.ID,
				Payload: PacketPayload{
					Type: PayloadSos,
					Sos:  sos,
				},
				TTL: 2,
			})
		} else {
			fmt.Printf("[%s] Sos message expired and removed.\n", m.Name)
		}
	}

	m.SosMessages = slices.DeleteFunc(m.SosMessages, func(sos *SosData) bool {
		return time.Since(sos.DetectedAt) >= 20*time.Second
	})

	return packets
}

// Heartbeat

func (m *MachineState) GetPendingHeartbeats() map[string]PendingPing {
	m.mu.Lock()
	defer m.mu.Unlock()

	copied := make(map[string]PendingPing, len(m.PendingHeartbeats))
	maps.Copy(copied, m.PendingHeartbeats)
	return copied
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

func (m *MachineState) GetStationTargeted() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.StationTargeted
}

func (m *MachineState) SetStationTargeted(targeted bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.StationTargeted = targeted
}

func (m *MachineState) GcPendingHeartbeats() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, ping := range m.PendingHeartbeats {
		if time.Since(ping.LastPingedAt) > HeartbeatMemoryTTL {
			fmt.Printf("[%s] Heartbeat has expired out of memory for %s.\nResponded? %t\n", m.Name, id, ping.HeardResponse)

			if !ping.HeardResponse {
				fmt.Println("No heartbeat response heard, sending SOS message.")
				sosData := &SosData{
					ID:              fmt.Sprintf("SOS-%s", m.ID),
					Type:            "SOS",
					TargetMachineId: id,
					OriginName:      m.ID,
					OriginX:         ping.TargetX,
					OriginY:         ping.TargetY,
					DetectedAt:      time.Now(),
				}
				m.SosMessages = append(m.SosMessages, sosData)

				m.QueuePacketToSend(GossipPacket{
					ID:       fmt.Sprintf("%s:%s", m.ID, time.Now()),
					OriginId: m.ID,
					Payload: PacketPayload{
						Type: PayloadSos,
						Sos:  sosData,
					},
					TTL: 3,
				})
			}
			delete(m.PendingHeartbeats, id)
		}
	}
}

func getEuclDistance(x1, y1, x2, y2 float64) float64 {
	return math.Sqrt(math.Pow(x2-x1, 2) + math.Pow(y2-y1, 2))
}
