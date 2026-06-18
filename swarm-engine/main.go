package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"path/filepath"
	"time"

	"swarm-engine/internal/machine"
	"swarm-engine/internal/simulation"
)

func main() {
	fmt.Println("[Hephaestus Core] Initializing Swarm Simulation Engine Skeleton...")

	// Target the config path relative to project workspace execution
	configPath := filepath.Join("configs", "cauldron_registry.json")

	registry, err := machine.LoadCauldronRegistry(configPath)
	if err != nil {
		log.Fatalf("Critical Initialization Failure: %v", err)
	}

	// fmt.Println("Cauldron Blueprints loaded successfully from disk!")
	// for name, specs := range registry {
	// 	fmt.Printf("   -> Loaded Class Profile: %-10s [Vision Range: %.1f | Base Speed: %.1f]\n",
	// 		name, specs.Vision.Range, specs.Locomotion.Speed)
	// }

	machines := []*machine.MachineState{}
	for name, specs := range registry {
		machines = append(machines, createMachine(1, name, specs)...)
	}

	engine := simulation.NewEngine(machines, 0.0)
	engine.Items = []machine.WorldItem{
		{
			ID:   "station-01",
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
	}
}

func createMachine(numOfMachines int, name string, specs machine.MachineBlueprint) []*machine.MachineState {
	createdMachines := []*machine.MachineState{}
	machineNum := 1

	const canvasWidth = 800.0
	const canvasHeight = 600.0
	for {
		if numOfMachines == 0 {
			return createdMachines
		} else {
			machine := &machine.MachineState{
				ID:             fmt.Sprintf("%s-%d", name, machineNum),
				Name:           name,
				X:              rand.Float64() * canvasWidth,
				Y:              rand.Float64() * canvasHeight,
				TargetX:        rand.Float64() * canvasWidth,
				TargetY:        rand.Float64() * canvasHeight,
				BatteryLevel:   specs.Battery.InitialLevel,
				TickerInterval: time.Second,
				Blueprint:      specs,
				Inbox:          make(chan machine.GossipPacket, 20),
				Outbox:         make(chan machine.GossipPacket, 20),
				SeenPackets:    make(map[string]bool),
				KnownStations:  make(map[string]machine.StationData),
			}

			createdMachines = append(createdMachines, machine)
			numOfMachines -= 1
			machineNum += 1
		}
	}
}
