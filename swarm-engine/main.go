package main

import (
	"fmt"
	"log"
	"path/filepath"

	"swarm-engine/internal/machine"
)

func main() {
	fmt.Println("🤖 [Hephaestus Core] Initializing Swarm Simulation Engine Skeleton...")

	// Target the config path relative to project workspace execution
	configPath := filepath.Join("configs", "cauldron_registry.json")

	registry, err := machine.LoadCauldronRegistry(configPath)
	if err != nil {
		log.Fatalf("❌ Critical Initialization Failure: %v", err)
	}

	fmt.Println("✅ Cauldron Blueprints loaded successfully from disk!")
	for name, specs := range registry {
		fmt.Printf("   -> Loaded Class Profile: %-10s [Vision Range: %.1f | Base Speed: %.1f]\n",
			name, specs.Vision.Range, specs.Locomotion.Speed)
	}
}
