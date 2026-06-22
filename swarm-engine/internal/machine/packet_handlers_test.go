package machine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Test resource packets
func TestHandleResourcePacket_DiscoversNewStation(t *testing.T) {
	m := &MachineState{
		ID:            "test-machine",
		Name:          "Scout",
		KnownStations: make(map[string]*StationData),
		SeenPackets:   make(map[string]bool),
	}

	packet := GossipPacket{
		ID:       "pkg-123",
		OriginId: "peer-01",
		Payload: PacketPayload{
			Type: PayloadResource,
			Resource: &ResourceData{
				ID:   "station-alpha",
				Type: ResourceStation,
				X:    45.0,
				Y:    90.0,
			},
		},
		TTL: 3,
	}

	m.handleIncomingPacket(&packet)

	assert.True(t, m.HasSeenPacket("pkg-123"), "Should mark packet ID as seen")
	assert.Contains(t, m.GetKnownStations(), "station-alpha", "Should add new station to memory")

	station := m.GetKnownStations()["station-alpha"]
	assert.Equal(t, 45.0, station.X)
	assert.Equal(t, 90.0, station.Y)
}

// Test heartbeat packets
func TestHandleHeartbeatPacket_SendsResponseOnRequest(t *testing.T) {
	m := &MachineState{
		ID:           "my-id",
		Name:         "Scrappy",
		BatteryLevel: 85.5,
		Outbox:       make(chan GossipPacket, 5),
		SeenPackets:  make(map[string]bool),
	}

	packet := GossipPacket{
		ID:       "hb-req-01",
		OriginId: "peer-id-99",
		Payload: PacketPayload{
			Type: PayloadHeartbeat,
			Heartbeat: &HeartbeatData{
				ID:   "peer-id-99",
				Type: HeartbeatRequest,
			},
		},
		TTL: 2, // TTL at 2 to verify routing propagation
	}

	m.handleIncomingPacket(&packet)

	assert.Len(t, m.Outbox, 2)

	// The first packet must be the generated Heartbeat Response
	responsePacket := <-m.Outbox
	assert.Equal(t, PayloadHeartbeat, responsePacket.Payload.Type)
	assert.Equal(t, HeartbeatResponse, responsePacket.Payload.Heartbeat.Type)
	assert.Equal(t, 85.5, responsePacket.Payload.Heartbeat.BatteryLevel)

	// The second packet must be the forwarded original Request with a decremented TTL
	forwardedPacket := <-m.Outbox
	assert.Equal(t, "hb-req-01", forwardedPacket.ID)
	assert.Equal(t, 1, forwardedPacket.TTL, "Mesh network should decrement packet TTL by 1")
}

// Test SOS packets
func TestHandleSosPacket_SparkerDispatchesRescue(t *testing.T) {
	m := &MachineState{
		ID:           "sparker-unit-01",
		Name:         "SPARKER", // Must be "SPARKER" to activate rescue paths
		State:        StatePatrol,
		BatteryLevel: 100.0,
		SeenPackets:  make(map[string]bool),
	}

	packet := GossipPacket{
		ID:       "sos-msg-777",
		OriginId: "stranded-bot-abc",
		Payload: PacketPayload{
			Type: PayloadSos,
			Sos: &SosData{
				ID:              "SOS-stranded-bot-abc",
				TargetMachineId: "stranded-bot-abc",
				OriginName:      "StrandedBot",
				OriginX:         500.0,
				OriginY:         600.0,
				DetectedAt:      time.Now(),
			},
		},
		TTL: 3,
	}

	m.handleIncomingPacket(&packet)

	assert.Equal(t, StateRescuing, m.GetState(), "Machine should switch to RESCUING state")

	targetX, targetY := m.GetTarget()
	assert.Equal(t, 500.0, targetX, "Target kinematics should lock onto distress coordinates")
	assert.Equal(t, 600.0, targetY)

	rescueInfo := m.GetRescueInfo()
	assert.NotNil(t, rescueInfo)
	assert.True(t, rescueInfo.HasRescueTarget)
	assert.Equal(t, "stranded-bot-abc", rescueInfo.MachineIdToRescue)
}

// Test self-rescue guard
func TestHandleSosPacket_IgnoresSelfRescue(t *testing.T) {
	m := &MachineState{
		ID:           "sparker-01",
		Name:         "SPARKER",
		State:        StatePatrol,
		BatteryLevel: 100.0,
		SeenPackets:  make(map[string]bool),
	}

	packet := GossipPacket{
		ID:       "sos-self",
		OriginId: "sparker-01",
		Payload: PacketPayload{
			Type: PayloadSos,
			Sos: &SosData{
				ID:              "SOS-sparker-01",
				TargetMachineId: "sparker-01", // Target matches this machine's own ID
				OriginName:      "SPARKER",
				OriginX:         999.0,
				OriginY:         999.0,
				DetectedAt:      time.Now(),
			},
		},
		TTL: 3,
	}

	m.handleIncomingPacket(&packet)
}

// Low power level guard
func TestHandleSosPacket_IgnoresWhenInLowPower(t *testing.T) {
	m := &MachineState{
		ID:           "sparker-01",
		Name:         "SPARKER",
		State:        StateLowPower,
		BatteryLevel: 15.0,
		SeenPackets:  make(map[string]bool),
	}

	packet := GossipPacket{
		ID:       "sos-lowpower",
		OriginId: "stranded-bot",
		Payload: PacketPayload{
			Type: PayloadSos,
			Sos: &SosData{
				ID:              "SOS-stranded-bot",
				TargetMachineId: "stranded-bot",
				OriginName:      "StrandedBot",
				OriginX:         500.0,
				OriginY:         600.0,
				DetectedAt:      time.Now(),
			},
		},
		TTL: 3,
	}

	m.handleIncomingPacket(&packet)

	assert.Equal(t, StateLowPower, m.GetState(), "Should NOT shift out of LOW_POWER state to perform a rescue")
	assert.Nil(t, m.GetRescueInfo(), "Rescue data processing must be completely bypassed")
}

// Dedupe guard
func TestHandleSosPacket_IgnoresDuplicateRescueTargets(t *testing.T) {
	m := &MachineState{
		ID:           "sparker-01",
		Name:         "SPARKER",
		State:        StateRescuing,
		BatteryLevel: 90.0,
		SeenPackets:  make(map[string]bool),
		RescueInfo: &RescueData{
			HasRescueTarget:   true,
			MachineIdToRescue: "bot-A",
		},
		TargetX: 100.0,
		TargetY: 100.0,
	}

	packet := GossipPacket{
		ID:       "sos-duplicate",
		OriginId: "bot-A",
		Payload: PacketPayload{
			Type: PayloadSos,
			Sos: &SosData{
				ID:              "SOS-bot-A-again",
				TargetMachineId: "bot-A",
				OriginName:      "bot-A",
				OriginX:         500.0,
				OriginY:         600.0,
				DetectedAt:      time.Now(),
			},
		},
		TTL: 3,
	}

	m.handleIncomingPacket(&packet)

	targetX, targetY := m.GetTarget()
	assert.Equal(t, 100.0, targetX, "Should not alter target tracking path for ongoing missions")
	assert.Equal(t, 100.0, targetY)
}
