package machine

import (
	"sync"
	"time"
)

type StationData struct {
	DiscoveredAt time.Time `json:"discovered_at"`
	X            float64   `json:"x"`
	Y            float64   `json:"y"`
	ExpiresAfter float64   `json:"expires_after"`
}

type PendingPing struct {
	LastPingedAt  time.Time `json:"last_pinged_at"`
	LastKnownX    float64   `json:"last_known_x"`
	LastKnownY    float64   `json:"last_known_y"`
	HeardResponse bool      `json:"heard_response"`
	TargetX       float64   `json:"target_x"`
	TargetY       float64   `json:"target_y"`
}

type RescueData struct {
	HasRescueTarget   bool    `json:"has_rescue_target"`
	MachineIdToRescue string  `json:"machine_id_to_rescue"`
	DrainFromStation  float64 `json:"drain_from_station"`
	AtRescueTarget    bool    `json:"at_rescue_target"`
}

type WorldItem struct {
	ID   string       `json:"id"`
	Type ResourceType `json:"type"`
	X    float64      `json:"x"`
	Y    float64      `json:"y"`
}

type Environment interface {
	GetVisibleItems(x, y, visionRange float64) []WorldItem
	GetVisibleMachines(machineId string, x, y, visionRange float64) []*MachineState
	ConnectToMachine(machineId string) *MachineState
	IsMachineInRange(machineId string) bool
}

type MachineState struct {
	mu                sync.Mutex
	Inbox             chan GossipPacket
	Outbox            chan GossipPacket
	ID                string                  `json:"id"`
	Name              string                  `json:"name"`
	State             State                   `json:"state"`
	X                 float64                 `json:"x"`
	Y                 float64                 `json:"y"`
	TargetX           float64                 `json:"target_x"`
	TargetY           float64                 `json:"target_y"`
	Vx                float64                 `json:"v_x"`
	Vy                float64                 `json:"v_y"`
	BatteryLevel      float64                 `json:"battery_level"`
	Blueprint         MachineBlueprint        `json:"blueprint"`
	TickerInterval    time.Duration           `json:"ticker_interval"`
	SeenPackets       map[string]bool         `json:"seen_packets"`
	Env               Environment             `json:"environment"`
	KnownStations     map[string]*StationData `json:"known_stations"`
	PendingHeartbeats map[string]PendingPing  `json:"pending_heartbeats"`
	RescueInfo        *RescueData             `json:"rescue_info"`
	SosMessages       []*SosData              `json:"sos_messages"`
	StationTargeted   bool                    `json:"station_targeted"`
}
