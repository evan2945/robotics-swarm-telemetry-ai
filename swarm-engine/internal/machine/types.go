package machine

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

type MachineState struct {
	ID           string           `json:"id"`
	Name         string           `json:"name"`
	State        string           `json:"state"`
	X            float64          `json:"x"`
	Y            float64          `json:"y"`
	TargetX      float64          `json:"target_x"`
	TargetY      float64          `json:"target_y"`
	Vx           float64          `json:"v_x"`
	Vy           float64          `json:"v_y"`
	BatteryLevel float64          `json:"battery_level"`
	Blueprint    MachineBlueprint `json:"blueprint"`
}
