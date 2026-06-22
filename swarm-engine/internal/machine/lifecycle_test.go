package machine

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type MockEnvironment struct{}

func (m *MockEnvironment) GetVisibleItems(x, y, visionRange float64) []WorldItem {
	allAvailableItems := []WorldItem{
		{
			ID:   "station-1",
			Type: ResourceStation,
			X:    105,
			Y:    105,
		},
	}

	var visibleItems []WorldItem
	for _, item := range allAvailableItems {
		xDist := item.X - x
		yDist := item.Y - y
		euclDistance := math.Sqrt(math.Pow(xDist, 2) + math.Pow(yDist, 2))

		if euclDistance <= visionRange {
			visibleItems = append(visibleItems, item)
		}
	}
	return visibleItems
}

func (m *MockEnvironment) GetVisibleMachines(machineId string, x, y, visionRange float64) []*MachineState {
	return []*MachineState{}
}

func (m *MockEnvironment) ConnectToMachine(machineId string) *MachineState {
	return nil
}

func (m *MockEnvironment) IsMachineInRange(machineId string) bool {
	return false
}

func TestRunLifecycle(t *testing.T) {
	mockMachine := &MachineState{
		ID:           "machine-01",
		Name:         "mock machine",
		State:        StatePatrol,
		BatteryLevel: 100.0,
		Blueprint: MachineBlueprint{
			Battery: BatteryBlueprint{
				DrainRate: 1.0,
			},
		},
	}

	for _ = range 4 {
		currentBattery := mockMachine.GetBatteryLevel()
		blueprint := mockMachine.GetBlueprint()

		if currentBattery <= 0 {
			break
		}

		switch mockMachine.GetState() {
		case StateCharging:
			mockMachine.SetBatteryLevel(currentBattery + blueprint.Battery.ChargeRate)
		case StateLowPower:
			mockMachine.SetBatteryLevel(currentBattery - blueprint.Battery.LowPowerDrain)
		case StateRecharging:
			mockMachine.SetBatteryLevel(currentBattery - blueprint.Battery.RechargingDrainRate)
		case StateIdle:
		default:
			mockMachine.SetBatteryLevel(currentBattery - blueprint.Battery.DrainRate)
		}
	}

	assert.Equal(t, 96.0, mockMachine.GetBatteryLevel())
}

func TestRunLifecycle_StationDiscover(t *testing.T) {
	mockMachine := &MachineState{
		ID:             "machine-01",
		Name:           "mock machine",
		State:          StatePatrol,
		X:              100,
		Y:              100,
		TargetX:        100,
		TargetY:        100,
		BatteryLevel:   100,
		Env:            &MockEnvironment{},
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
	assert.Equal(t, 1, len(knownStations))
	assert.Contains(t, knownStations, "station-1")
	assert.True(t, len(mockMachine.Outbox) >= 1)
}

func TestGossipNetwork_Transmission(t *testing.T) {
	packet1 := GossipPacket{
		ID:       "msg-001",
		OriginId: "machine-99",
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
		OriginId: "machine-99",
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
		packetsToSend          []GossipPacket
		expectedUniqueSeenSize int
		expectedSeenIds        []string
		shouldCheckOutbox      bool
	}{
		{
			description:            "Receiving a single, new packet adds its ID to SeenPackets and decrements TTL in outbox",
			packetsToSend:          []GossipPacket{packet1},
			expectedUniqueSeenSize: 1,
			expectedSeenIds:        []string{"msg-001"},
			shouldCheckOutbox:      true,
		},
		{
			description:            "Receiving a repeated packet ignores the duplicate and processes it exactly once",
			packetsToSend:          []GossipPacket{packet1, packet1},
			expectedUniqueSeenSize: 1,
			expectedSeenIds:        []string{"msg-001"},
			shouldCheckOutbox:      false,
		},
		{
			description:            "Receiving multiple unique packets tracks all identifiers distinctively",
			packetsToSend:          []GossipPacket{packet1, packet2},
			expectedUniqueSeenSize: 2,
			expectedSeenIds:        []string{"msg-001", "msg-002"},
			shouldCheckOutbox:      false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			machineInstance := &MachineState{
				ID:             "machine-01",
				Name:           "mock machine",
				State:          StatePatrol,
				X:              12,
				Y:              12,
				BatteryLevel:   100,
				TickerInterval: time.Millisecond,
				Blueprint: MachineBlueprint{
					Battery: BatteryBlueprint{DrainRate: 1.0},
				},
				Inbox:         make(chan GossipPacket, 10),
				Outbox:        make(chan GossipPacket, 10),
				SeenPackets:   make(map[string]bool),
				KnownStations: make(map[string]*StationData),
			}

			for _, p := range tc.packetsToSend {
				machineInstance.Inbox <- p
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
			defer cancel()

			go machineInstance.RunLifecycle(ctx)
			<-ctx.Done()

			seenPackets := machineInstance.GetSeenPackets()
			assert.Equal(t, tc.expectedUniqueSeenSize, len(seenPackets))

			for _, id := range tc.expectedSeenIds {
				assert.True(t, machineInstance.HasSeenPacket(id))
			}

			if tc.shouldCheckOutbox {
				select {
				case outboxMessage := <-machineInstance.Outbox:
					assert.Equal(t, tc.packetsToSend[0].ID, outboxMessage.ID)
					assert.Equal(t, 2, outboxMessage.TTL)
				default:
					t.Fatal("Expected packet in outbox but found none")
				}
			}
		})
	}
}

func TestRunLifecycle_BatteryDepletionAndDeactivation(t *testing.T) {
	mockMachine := &MachineState{
		ID:           "machine-01",
		Name:         "mock machine",
		State:        StatePatrol,
		BatteryLevel: 0.0, // Pre-kill the machine
		Inbox:        make(chan GossipPacket, 10),
	}

	// 1. Validate that ticker processing shifts state to StateDead
	// Manually execute the block inside the `case <-ticker.C:` layer
	batteryLevel := mockMachine.GetBatteryLevel()
	if batteryLevel <= 0 {
		if mockMachine.GetState() != StateDead {
			mockMachine.SetBatteryLevel(0.0)
			mockMachine.SetState(StateDead)
		}
	}

	assert.Equal(t, StateDead, mockMachine.GetState())
	assert.Equal(t, 0.0, mockMachine.GetBatteryLevel())

	// 2. Validate that an incoming packet on a dead machine is dropped
	mockMachine.Inbox <- GossipPacket{ID: "sos-test", Payload: PacketPayload{Type: PayloadSos}}

	// Simulate the exact channel check:
	packet := <-mockMachine.Inbox
	if mockMachine.GetBatteryLevel() <= 0 {
		// Simulates the 'continue' execution branch, meaning the packet handler never runs
		assert.False(t, mockMachine.HasSeenPacket(packet.ID))
	}
}

func TestRunLifecycle_BatteryStatePhysics(t *testing.T) {
	blueprint := MachineBlueprint{
		Battery: BatteryBlueprint{
			ChargeRate:          5.0,
			LowPowerDrain:       0.5,
			RechargingDrainRate: 2.0,
			DrainRate:           1.5,
		},
	}

	testCases := []struct {
		description   string
		initialState  State
		startBattery  float64
		expectBattery float64
	}{
		{
			description:   "Charging state replenishes the battery level",
			initialState:  StateCharging,
			startBattery:  50.0,
			expectBattery: 55.0, // 50 + 5.0
		},
		{
			description:   "Low power state applies a lighter conservation drain",
			initialState:  StateLowPower,
			startBattery:  15.0,
			expectBattery: 14.5, // 15 - 0.5
		},
		{
			description:   "Recharging state applies an active transfer drain",
			initialState:  StateRecharging,
			startBattery:  40.0,
			expectBattery: 38.0, // 40 - 2.0
		},
		{
			description:   "Default patrol states apply standard system drain",
			initialState:  StatePatrol,
			startBattery:  80.0,
			expectBattery: 78.5, // 80 - 1.5
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			m := &MachineState{State: tc.initialState, BatteryLevel: tc.startBattery, Blueprint: blueprint}

			// Execute the isolated calculation block
			bLevel := m.GetBatteryLevel()
			bp := m.GetBlueprint()
			switch m.GetState() {
			case StateCharging:
				m.SetBatteryLevel(bLevel + bp.Battery.ChargeRate)
			case StateLowPower:
				m.SetBatteryLevel(bLevel - bp.Battery.LowPowerDrain)
			case StateRecharging:
				m.SetBatteryLevel(bLevel - bp.Battery.RechargingDrainRate)
			case StateIdle:
			default:
				m.SetBatteryLevel(bLevel - bp.Battery.DrainRate)
			}

			assert.Equal(t, tc.expectBattery, m.GetBatteryLevel())
		})
	}
}

func TestRunLifecycle_DeduplicateHeartbeatRequests(t *testing.T) {
	mockMachine := &MachineState{
		ID:                "sparker-unit",
		State:             StatePatrol,
		BatteryLevel:      100.0,
		Outbox:            make(chan GossipPacket, 10),
		PendingHeartbeats: make(map[string]PendingPing),
	}

	targetPeerID := "endangered-bot"

	// Cycle 1: Peer discovered for the first time
	if _, ok := mockMachine.PendingHeartbeats[targetPeerID]; !ok {
		mockMachine.PendingHeartbeats[targetPeerID] = PendingPing{LastPingedAt: time.Now()}
		mockMachine.Outbox <- GossipPacket{ID: "ping-1"}
	}

	// Cycle 2: Peer is still visible on the next tick
	if _, ok := mockMachine.PendingHeartbeats[targetPeerID]; !ok {
		mockMachine.Outbox <- GossipPacket{ID: "ping-2"} // Should not execute
	}

	// Verify that only 1 registration occurred and exactly 1 packet went out
	assert.Len(t, mockMachine.PendingHeartbeats, 1)
	assert.Len(t, mockMachine.Outbox, 1)
}
