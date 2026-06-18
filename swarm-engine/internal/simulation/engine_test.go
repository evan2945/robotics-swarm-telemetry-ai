package simulation

import (
	"context"
	"swarm-engine/internal/machine"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEngineTick(t *testing.T) {
	testCases := []struct {
		description string
		startX      float64
		startY      float64
		targetX     float64
		targetY     float64
		speed       float64
		dt          float64

		expectedX       float64
		expectedY       float64
		expectedVx      float64
		expectedVy      float64
		expectedTargetX float64
		expectedTargetY float64
	}{
		{
			description: "Calculates proper values for standard movement",
			startX:      100.0,
			startY:      100.0,
			targetX:     200.0,
			targetY:     100.0,
			speed:       10.0,
			dt:          1.0,

			expectedX:       110.0,
			expectedY:       100.0,
			expectedVx:      10.0,
			expectedVy:      0.0,
			expectedTargetX: 200.0,
			expectedTargetY: 100.0,
		},
		{
			description: "Ensures proper state updates in the case of an encounter with a left wall boundary",
			startX:      5.0,
			startY:      100.0,
			targetX:     -50.0,
			targetY:     100.0,
			speed:       -10.0,
			dt:          1.0,

			expectedX:       0.0,
			expectedY:       100.0,
			expectedVx:      10.0,
			expectedVy:      0.0,
			expectedTargetX: 50.0,
			expectedTargetY: 100.0,
		},
		{
			description: "Ensures proper state updates in the case of an encounter with a right wall boundary",
			startX:      795.0,
			startY:      100.0,
			targetX:     850.0,
			targetY:     100.0,
			speed:       10.0,
			dt:          1.0,

			expectedX:       800.0,
			expectedY:       100.0,
			expectedVx:      -10.0,
			expectedVy:      0.0,
			expectedTargetX: 750.0,
			expectedTargetY: 100.0,
		},
		{
			description: "Ensures proper state updates in the case of an encounter with an lower wall boundary",
			startX:      100.0,
			startY:      5.0,
			targetX:     100.0,
			targetY:     -50.0,
			speed:       -10.0,
			dt:          1.0,

			expectedX:       100.0,
			expectedY:       0.0,
			expectedVx:      0.0,
			expectedVy:      10.0,
			expectedTargetX: 100.0,
			expectedTargetY: 50.0,
		},
		{
			description: "Ensures proper state updates in the case of an encounter with an upper wall boundary",
			startX:      100.0,
			startY:      595.0,
			targetX:     100.0,
			targetY:     650.0,
			speed:       10.0,
			dt:          1.0,

			expectedX:       100.0,
			expectedY:       600.0,
			expectedVx:      0.0,
			expectedVy:      -10.0,
			expectedTargetX: 100.0,
			expectedTargetY: 550.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			mockMachine := &machine.MachineState{
				ID:             "machine-01",
				Name:           "mock machine",
				State:          "PATROL",
				X:              tc.startX,
				Y:              tc.startY,
				TargetX:        tc.targetX,
				TargetY:        tc.targetY,
				BatteryLevel:   100,
				TickerInterval: time.Second,
				Blueprint: machine.MachineBlueprint{
					Locomotion: machine.LocomotionBlueprint{
						Speed: 10,
					},
				},
				Inbox:       make(chan machine.GossipPacket, 10),
				Outbox:      make(chan machine.GossipPacket, 10),
				SeenPackets: make(map[string]bool),
			}

			engine := NewEngine([]*machine.MachineState{mockMachine}, 0.0)
			engine.Tick(tc.dt)

			assert.Equal(t, tc.expectedX, mockMachine.X)
			assert.Equal(t, tc.expectedY, mockMachine.Y)
			assert.Equal(t, tc.expectedVx, mockMachine.Vx)
			assert.Equal(t, tc.expectedVy, mockMachine.Vy)
			assert.Equal(t, tc.expectedTargetX, mockMachine.TargetX)
			assert.Equal(t, tc.expectedTargetY, mockMachine.TargetY)
		})
	}
}

func TestGossipRouting(t *testing.T) {
	testCases := []struct {
		description string
		packet      machine.GossipPacket
	}{
		{
			description: "Communication between machines, using multiple hops, executes as expected",
			packet: machine.GossipPacket{
				ID:       "msg-001",
				OriginId: "machine-01",
				Payload: machine.PacketPayload{
					Type: machine.PayloadResource,
					Resource: &machine.ResourceData{
						ID:   "station-01",
						Type: machine.ResourceStation,
						X:    142,
						Y:    438,
					},
				},
				TTL: 3,
			},
		},
		{
			description: "Message is not forwarded if the TTL has expired",
			packet: machine.GossipPacket{
				ID:       "msg-001",
				OriginId: "machine-01",
				Payload: machine.PacketPayload{
					Type: machine.PayloadResource,
					Resource: &machine.ResourceData{
						ID:   "station-01",
						Type: machine.ResourceStation,
						X:    142,
						Y:    438,
					},
				},
				TTL: 1,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			mockMachineA := &machine.MachineState{
				ID:             "machine-01",
				Name:           "mock machine",
				State:          "PATROL",
				X:              10,
				Y:              10,
				TargetX:        10,
				TargetY:        10,
				BatteryLevel:   100,
				TickerInterval: time.Millisecond,
				Blueprint: machine.MachineBlueprint{
					Locomotion: machine.LocomotionBlueprint{
						Speed: 10,
					},
					Communication: machine.CommunicationBlueprint{
						Range: 15,
					},
				},
				Inbox:         make(chan machine.GossipPacket, 10),
				Outbox:        make(chan machine.GossipPacket, 10),
				SeenPackets:   make(map[string]bool),
				KnownStations: make(map[string]*machine.StationData),
			}
			mockMachineB := &machine.MachineState{
				ID:             "machine-02",
				Name:           "mock machine",
				State:          "PATROL",
				X:              20,
				Y:              10,
				TargetX:        10,
				TargetY:        10,
				BatteryLevel:   100,
				TickerInterval: time.Millisecond,
				Blueprint: machine.MachineBlueprint{
					Locomotion: machine.LocomotionBlueprint{
						Speed: 10,
					},
					Communication: machine.CommunicationBlueprint{
						Range: 15,
					},
				},
				Inbox:         make(chan machine.GossipPacket, 10),
				Outbox:        make(chan machine.GossipPacket, 10),
				SeenPackets:   make(map[string]bool),
				KnownStations: make(map[string]*machine.StationData),
			}
			mockMachineC := &machine.MachineState{
				ID:             "machine-03",
				Name:           "mock machine",
				State:          "PATROL",
				X:              32,
				Y:              10,
				TargetX:        10,
				TargetY:        10,
				BatteryLevel:   100,
				TickerInterval: time.Millisecond,
				Blueprint: machine.MachineBlueprint{
					Locomotion: machine.LocomotionBlueprint{
						Speed: 10,
					},
					Communication: machine.CommunicationBlueprint{
						Range: 15,
					},
				},
				Inbox:         make(chan machine.GossipPacket, 10),
				Outbox:        make(chan machine.GossipPacket, 10),
				SeenPackets:   make(map[string]bool),
				KnownStations: make(map[string]*machine.StationData),
			}
			mockMachineA.Outbox <- tc.packet

			engine := NewEngine([]*machine.MachineState{mockMachineA, mockMachineB, mockMachineC}, 0.0)
			// Tick(0) simulates the engine getting the packet from A's outbox and dropping it into B's inbox
			engine.Tick(0)

			// B should now have the packet, but C should not, since it is out of range of machine A
			assert.True(t, len(mockMachineB.Inbox) > 0)
			assert.True(t, len(mockMachineC.Inbox) == 0)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
			defer cancel()

			go mockMachineB.RunLifecycle(ctx)

			<-ctx.Done()

			if tc.packet.TTL > 1 {
				// Simulate the message being put in C's inbox
				engine.Tick(0)

				assert.True(t, len(mockMachineC.Inbox) > 0)
				assert.True(t, len(mockMachineB.GetSeenPackets()) == 1)

				packet := <-mockMachineC.Inbox
				assert.Equal(t, 2, packet.TTL)
			} else {
				// Packet TTL expired before reaching machine C
				assert.True(t, len(mockMachineB.GetSeenPackets()) == 1)
				assert.True(t, len(mockMachineC.Inbox) == 0)
			}
		})
	}
}

func TestGossipRouting_BeaconBroadcast(t *testing.T) {
	testCases := []struct {
		description               string
		packet                    machine.GossipPacket
		secondsSinceLastBroadcast float64
		broadcastShouldHappen     bool
	}{
		{
			description: "If there is an entry in KnownStations, it continues to broadcast.",
			packet: machine.GossipPacket{
				ID:       "msg-001",
				OriginId: "machine-01",
				Payload: machine.PacketPayload{
					Type: machine.PayloadResource,
					Resource: &machine.ResourceData{
						ID:   "station-01",
						Type: machine.ResourceStation,
						X:    142,
						Y:    438,
					},
				},
				TTL: 3,
			},
			secondsSinceLastBroadcast: 1.0,
			broadcastShouldHappen:     true,
		},
		{
			description: "Entry in KnownStations does not broadcast if enough time has not passed",
			packet: machine.GossipPacket{
				ID:       "msg-001",
				OriginId: "machine-01",
				Payload: machine.PacketPayload{
					Type: machine.PayloadResource,
					Resource: &machine.ResourceData{
						ID:   "station-01",
						Type: machine.ResourceStation,
						X:    142,
						Y:    438,
					},
				},
				TTL: 3,
			},
			secondsSinceLastBroadcast: 0.5,
			broadcastShouldHappen:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			mockMachineA := &machine.MachineState{
				ID:             "machine-01",
				Name:           "mock machine A",
				State:          "PATROL",
				X:              10,
				Y:              10,
				TargetX:        10,
				TargetY:        10,
				BatteryLevel:   100,
				TickerInterval: time.Millisecond,
				Blueprint: machine.MachineBlueprint{
					Locomotion: machine.LocomotionBlueprint{
						Speed: 10,
					},
					Communication: machine.CommunicationBlueprint{
						Range: 15,
					},
				},
				Inbox:       make(chan machine.GossipPacket, 10),
				Outbox:      make(chan machine.GossipPacket, 10),
				SeenPackets: make(map[string]bool),
				KnownStations: map[string]*machine.StationData{
					tc.packet.ID: {
						DiscoveredAt: time.Now(),
						X:            142,
						Y:            438,
					},
				},
			}
			mockMachineB := &machine.MachineState{
				ID:             "machine-02",
				Name:           "mock machine B",
				State:          "PATROL",
				X:              20,
				Y:              10,
				TargetX:        10,
				TargetY:        10,
				BatteryLevel:   100,
				TickerInterval: time.Millisecond,
				Blueprint: machine.MachineBlueprint{
					Locomotion: machine.LocomotionBlueprint{
						Speed: 10,
					},
					Communication: machine.CommunicationBlueprint{
						Range: 15,
					},
				},
				Inbox:         make(chan machine.GossipPacket, 10),
				Outbox:        make(chan machine.GossipPacket, 10),
				SeenPackets:   make(map[string]bool),
				KnownStations: make(map[string]*machine.StationData),
			}

			engine := NewEngine([]*machine.MachineState{mockMachineA, mockMachineB}, tc.secondsSinceLastBroadcast)
			// mockMachineA has a known location, but it isn't in the outbox, so this is a case where the "gossip" was sent once before, and needs
			// to be re-transmitted, which is handled by the engine. So run a "Tick".
			engine.Tick(0)

			if tc.broadcastShouldHappen {
				assert.True(t, len(mockMachineB.Inbox) == 1)
			} else {
				assert.True(t, len(mockMachineB.Inbox) == 0)
			}

			// Run the lifecycle once, just to observe the Inbox being processed. There should be an entry in SeenPackets if
			// a broadcast happened.
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
			defer cancel()

			go mockMachineB.RunLifecycle(ctx)

			<-ctx.Done()

			if tc.broadcastShouldHappen {
				assert.True(t, len(mockMachineB.SeenPackets) == 1)
			} else {
				assert.True(t, len(mockMachineB.SeenPackets) == 0)
			}
		})
	}
}
