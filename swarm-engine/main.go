package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"path/filepath"
	"time"

	"swarm-engine/internal/machine"
	"swarm-engine/internal/simulation"

	"github.com/redis/go-redis/v9"
)

type MachineTelemetry struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	X            float64       `json:"x"`
	Y            float64       `json:"y"`
	HeadingX     float64       `json:"vx"`
	HeadingY     float64       `json:"vy"`
	State        machine.State `json:"state"`
	BatteryLevel float64       `json:"battery"`
}

type ItemTelemetry struct {
	ID   string  `json:"id"`
	Type string  `json:"type"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
}

type FrameTelemetry struct {
	Timestamp int64              `json:"timestamp"`
	Machines  []MachineTelemetry `json:"machines"`
	Items     []ItemTelemetry    `json:"items"`
}

const stationId = "station-01"

func main() {
	fmt.Println("[Hephaestus Core] Initializing Swarm Simulation Engine Skeleton...")

	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	// Target the config path relative to project workspace execution
	configPath := filepath.Join("configs", "cauldron_registry.json")

	registry, err := machine.LoadCauldronRegistry(configPath)
	if err != nil {
		log.Fatalf("Critical Initialization Failure: %v", err)
	}

	machines := []*machine.MachineState{}
	for name, specs := range registry {
		machines = append(machines, createMachine(1, name, specs, stationId)...)
	}

	engine := simulation.NewEngine(machines, 0.0)
	engine.Items = []machine.WorldItem{
		{
			ID:   stationId,
			Type: machine.ResourceStation,
			X:    engine.Width / 2,
			Y:    engine.Height / 2,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, m := range engine.Machines {
		go m.RunLifecycle(ctx)
	}

	fmt.Println("[Hephaestus Core] Swarm routines online, running engine tick...")

	frameInterval := time.Second / 60
	ticker := time.NewTicker(frameInterval)
	defer ticker.Stop()

	lastTime := time.Now()

	for range ticker.C {
		now := time.Now()
		dt := now.Sub(lastTime).Seconds()
		lastTime = now

		engine.Tick(dt)

		telemetryMachines := make([]MachineTelemetry, len(engine.Machines))
		for i, m := range engine.Machines {
			mx, my := m.GetPosition()
			vx, vy := m.GetHeading()

			telemetryMachines[i] = MachineTelemetry{
				ID:           m.GetId(),
				Name:         m.GetName(),
				X:            mx,
				Y:            my,
				HeadingX:     vx,
				HeadingY:     vy,
				State:        m.GetState(),
				BatteryLevel: m.GetBatteryLevel(),
			}
		}

		itemTelemetrys := make([]ItemTelemetry, len(engine.Items))
		for i, item := range engine.Items {
			itemTelemetrys[i] = ItemTelemetry{
				ID:   item.ID,
				Type: "CHARGING_STATION",
				X:    item.X,
				Y:    item.Y,
			}
		}

		frameData := FrameTelemetry{
			Timestamp: time.Now().UnixMilli(),
			Machines:  telemetryMachines,
			Items:     itemTelemetrys,
		}

		jsonBytes, err := json.Marshal(frameData)
		if err != nil {
			log.Printf("Telemetry serialization error: %v", err)
			continue
		}

		err = rdb.XAdd(ctx, &redis.XAddArgs{
			Stream: "swarm:telemetry",
			MaxLen: 100,
			Approx: true,
			ID:     "*",
			Values: map[string]interface{}{
				"frame": string(jsonBytes),
			},
		}).Err()

		if err != nil {
			log.Printf("Redis stream publish failed: %v", err)
		}
	}
}

func createMachine(numOfMachines int, name string, specs machine.MachineBlueprint, stationId string) []*machine.MachineState {
	createdMachines := []*machine.MachineState{}
	machineNum := 1

	const canvasWidth = 800.0
	const canvasHeight = 600.0

	startX := rand.Float64() * canvasWidth
	startY := rand.Float64() * canvasHeight

	targetX := rand.Float64() * canvasWidth
	targetY := rand.Float64() * canvasHeight

	knownStations := make(map[string]*machine.StationData)

	// Specific setup for the sparker
	if name == "SPARKER" {
		startX = canvasWidth / 2
		startY = canvasHeight / 2

		targetX = startX
		targetY = startY

		knownStations[stationId] = &machine.StationData{
			DiscoveredAt: time.Now(),
			X:            startX,
			Y:            startY,
			ExpiresAfter: -1,
		}
	}

	for {
		if numOfMachines == 0 {
			return createdMachines
		} else {
			machine := &machine.MachineState{
				ID:                fmt.Sprintf("%s-%d", name, machineNum),
				Name:              name,
				X:                 startX,
				Y:                 startY,
				TargetX:           targetX,
				TargetY:           targetY,
				BatteryLevel:      specs.Battery.InitialLevel,
				TickerInterval:    time.Second,
				Blueprint:         specs,
				State:             specs.States.InitialState,
				Inbox:             make(chan machine.GossipPacket, 20),
				Outbox:            make(chan machine.GossipPacket, 20),
				SeenPackets:       make(map[string]bool),
				KnownStations:     knownStations,
				PendingHeartbeats: make(map[string]machine.PendingPing),
			}

			createdMachines = append(createdMachines, machine)
			numOfMachines -= 1
			machineNum += 1
		}
	}
}
