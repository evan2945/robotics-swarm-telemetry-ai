package machine

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type MockEnvironment struct {
	ItemsToReturn []WorldItem
}

func (m *MockEnvironment) GetVisibleItems(x, y, visionRange float64) []WorldItem {
	items := []WorldItem{
		{
			ID:   "station-1",
			Type: "CHARGING_STATION",
			X:    105,
			Y:    105,
		},
	}

	for _, item := range items {
		x_dist := item.X - x
		y_dist := item.Y - y
		eucl_distance := math.Sqrt(math.Pow(x_dist, 2) + math.Pow(y_dist, 2))

		if eucl_distance <= visionRange {
			m.ItemsToReturn = append(m.ItemsToReturn, item)
		}
	}

	return m.ItemsToReturn
}

func TestRunLifecycle(t *testing.T) {
	mockMachine := &MachineState{
		ID:             "machine-01",
		Name:           "mock machine",
		State:          "PATROL",
		X:              100,
		Y:              100,
		TargetX:        100,
		TargetY:        100,
		BatteryLevel:   100,
		TickerInterval: time.Millisecond,
		Blueprint: MachineBlueprint{
			Battery: BatteryBlueprint{
				DrainRate: 1.0,
			},
		},
		Inbox:       make(chan GossipPacket, 10),
		Outbox:      make(chan GossipPacket, 10),
		SeenPackets: make(map[string]bool),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	go mockMachine.RunLifecycle(ctx)

	<-ctx.Done()

	assert.Equal(t, mockMachine.BatteryLevel, 96.0)
}

func TestRunLifecycle_StationDiscover(t *testing.T) {
	mockEnv := &MockEnvironment{}

	mockMachine := &MachineState{
		ID:             "machine-01",
		Name:           "mock machine",
		State:          "PATROL",
		X:              100,
		Y:              100,
		TargetX:        100,
		TargetY:        100,
		BatteryLevel:   100,
		Env:            mockEnv,
		TickerInterval: time.Millisecond,
		Blueprint: MachineBlueprint{
			Battery: BatteryBlueprint{
				DrainRate: 1.0,
			},
			Vision: VisionBlueprint{
				Range: 15,
			},
		},
		Inbox:         make(chan GossipPacket, 10),
		Outbox:        make(chan GossipPacket, 10),
		SeenPackets:   make(map[string]bool),
		KnownStations: make(map[string]*StationData),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	go mockMachine.RunLifecycle(ctx)

	<-ctx.Done()

	knownStations := mockMachine.GetKnownStations()
	assert.Equal(t, len(knownStations), 1)
	assert.Contains(t, knownStations, mockEnv.ItemsToReturn[0].ID)

	assert.True(t, len(mockMachine.Outbox) == 1)
}

func TestGossipNetwork_Transmission(t *testing.T) {
	mockMachineA := &MachineState{
		ID:             "machine-01",
		Name:           "mock machine",
		State:          "PATROL",
		X:              12,
		Y:              12,
		TargetX:        100,
		TargetY:        100,
		BatteryLevel:   100,
		TickerInterval: time.Millisecond,
		Blueprint: MachineBlueprint{
			Battery: BatteryBlueprint{
				DrainRate: 1.0,
			},
			Communication: CommunicationBlueprint{
				Range: 10,
			},
		},
		Inbox:         make(chan GossipPacket, 10),
		Outbox:        make(chan GossipPacket, 10),
		SeenPackets:   make(map[string]bool),
		KnownStations: make(map[string]*StationData),
	}

	packet := GossipPacket{
		ID:       "msg-001",
		OriginId: "machine-01",
		Payload: PacketPayload{
			Type: PayloadResource,
			Resource: &ResourceData{
				ID:   "station-01",
				Type: ResourceStation,
				X:    257,
				Y:    521,
			},
		},
		TTL: 3,
	}

	packet2 := GossipPacket{
		ID:       "msg-002",
		OriginId: "machine-01",
		Payload: PacketPayload{
			Type: PayloadResource,
			Resource: &ResourceData{
				ID:   "station-02",
				Type: ResourceStation,
				X:    351,
				Y:    428,
			},
		},
		TTL: 3,
	}

	testCases := []struct {
		description            string
		packets                []GossipPacket
		packetIdsInSeenPackets []string
		shouldCheckOutbox      bool
	}{
		{
			description:            "Receiving a single, new, packet adds its ID to SeenPackets and queues it in the outbox",
			packets:                []GossipPacket{packet},
			packetIdsInSeenPackets: []string{packet.ID},
			shouldCheckOutbox:      true,
		},
		{
			description:            "Receiving a repeated packet does not add to SeenPackets",
			packets:                []GossipPacket{packet, packet},
			packetIdsInSeenPackets: []string{packet.ID},
			shouldCheckOutbox:      false,
		},
		{
			description:            "Receiving a muliple, unique, packets adds them to SeenPackets",
			packets:                []GossipPacket{packet, packet2},
			packetIdsInSeenPackets: []string{packet.ID, packet2.ID},
			shouldCheckOutbox:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			for _, packet := range tc.packets {
				mockMachineA.Inbox <- packet
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
			defer cancel()

			go mockMachineA.RunLifecycle(ctx)

			<-ctx.Done()

			assert.Equal(t, len(tc.packetIdsInSeenPackets), len(mockMachineA.GetSeenPackets()))
			for _, packetId := range tc.packetIdsInSeenPackets {
				assert.True(t, mockMachineA.HasSeenPacket(packetId))
			}

			// To reduce complexity, only check outbox for simple test cases
			if tc.shouldCheckOutbox {
				outboxMessage := <-mockMachineA.Outbox
				assert.Equal(t, packet.ID, outboxMessage.ID)
				assert.Equal(t, 2, outboxMessage.TTL)
			}
		})
	}
}
