package machine

import (
	"fmt"
	"time"
)

type PayloadType string
type ResourceType string
type HeartbeatType string

const (
	PayloadResource  PayloadType = "RESOURCE"
	PayloadHeartbeat PayloadType = "HEARTBEAT"
	PayloadSos       PayloadType = "SOS"

	ResourceStation ResourceType = "STATION"

	HeartbeatRequest  HeartbeatType = "HEARTBEAT REQUEST"
	HeartbeatResponse HeartbeatType = "HEARTBEAT RESPONSE"
)

type GossipPacket struct {
	ID       string
	OriginId string
	Payload  PacketPayload
	TTL      int
}

type PacketPayload struct {
	Type      PayloadType    `json:"type"`
	Resource  *ResourceData  `json:"resource,omitempty"`
	Heartbeat *HeartbeatData `json:"heartbeat,omitempty"`
	Sos       *SosData       `json:"sos,omitempty"`
}

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

type SosData struct {
	ID              string    `json:"id"`
	Type            string    `json:"type"`
	TargetMachineId string    `json:"target_machine_id"`
	OriginName      string    `json:"origin_name"`
	OriginX         float64   `json:"origin_x"`
	OriginY         float64   `json:"origin_y"`
	DetectedAt      time.Time `json:"detected_at"`
}

// Generates a resource discovery notification packet
func (m *MachineState) NewResourcePacket(item WorldItem) GossipPacket {
	return GossipPacket{
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
}

// Generates a ping request intended for a visible peer
func (m *MachineState) NewHeartbeatRequestPacket(targetMachineID string, currentBattery float64) GossipPacket {
	mId := m.GetId()
	mX, mY := m.GetPosition()
	return GossipPacket{
		ID:       fmt.Sprintf("%s:%s:%s", mId, targetMachineID, time.Now()),
		OriginId: mId,
		Payload: PacketPayload{
			Type: PayloadHeartbeat,
			Heartbeat: &HeartbeatData{
				ID:           mId,
				Type:         HeartbeatRequest,
				X:            mX,
				Y:            mY,
				BatteryLevel: currentBattery,
			},
		},
		TTL: 1,
	}
}

// Generates a reactive ping confirmation message back to an inquirer
func (m *MachineState) NewHeartbeatResponsePacket(targetMachineID string, currentBattery float64) GossipPacket {
	mId := m.GetId()
	mX, mY := m.GetPosition()
	return GossipPacket{
		ID:       fmt.Sprintf("%s:%s:%s", mId, targetMachineID, time.Now()),
		OriginId: mId,
		Payload: PacketPayload{
			Type: PayloadHeartbeat,
			Heartbeat: &HeartbeatData{
				ID:           mId,
				Type:         HeartbeatResponse,
				X:            mX,
				Y:            mY,
				BatteryLevel: currentBattery,
			},
		},
		TTL: 1,
	}
}
