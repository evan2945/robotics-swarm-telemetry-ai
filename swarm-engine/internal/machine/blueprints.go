package machine

import "time"

const StationMemoryTTL = 10 * time.Second
const HeartbeatMemoryTTL = 4 * time.Second

type State string

const (
	StatePatrol     State = "PATROL"
	StateCharging   State = "CHARGING"
	StateLowPower   State = "LOW_POWER"
	StateDead       State = "DEAD"
	StateIdle       State = "IDLE"
	StateRecharging State = "RECHARGING"
	StateRescuing   State = "RESCUING"
)

type StateBlueprint struct {
	InitialState     State   `json:"initial_state"`
	AdditionalStates []State `json:"additional_states"`
}

type BatteryBlueprint struct {
	InitialLevel        float64 `json:"initial_level"`
	DrainRate           float64 `json:"drain_rate"`
	ChargeRate          float64 `json:"charge_rate"`
	LowPowerDrain       float64 `json:"low_power_drain"`
	RechargingDrainRate float64 `json:"recharging_drain_rate"`
	TransferEfficiency  float64 `json:"transfer_efficiency"`
}

type LocomotionBlueprint struct {
	Speed            float64 `json:"speed"`
	LowPowerSpeed    float64 `json:"low_power_speed"`
	EvasionThreshold float64 `json:"evasion_threshold"`
}

type VisionBlueprint struct {
	Range        float64 `json:"range"`
	AngleDegrees float64 `json:"angle_degrees"`
}

type CommunicationBlueprint struct {
	Range         float64 `json:"range"`
	LowPowerRange float64 `json:"low_power_range"`
}

type MachineBlueprint struct {
	States        StateBlueprint         `json:"states"`
	Battery       BatteryBlueprint       `json:"battery"`
	Locomotion    LocomotionBlueprint    `json:"locomotion"`
	Vision        VisionBlueprint        `json:"vision"`
	Communication CommunicationBlueprint `json:"communication"`
}

type CauldronRegistry map[string]MachineBlueprint
