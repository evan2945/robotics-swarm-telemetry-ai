package machine

import (
	"fmt"
)

// Main router for incoming gossip network packets
func (m *MachineState) handleIncomingPacket(packet *GossipPacket) {
	if m.HasSeenPacket(packet.ID) {
		return
	}
	m.AddSeenPacket(packet.ID)

	switch packet.Payload.Type {
	case PayloadResource:
		m.handleResourcePacket(packet)
	case PayloadHeartbeat:
		m.handleHeartbeatPacket(packet)
	case PayloadSos:
		m.handleSosPacket(packet)
	}

	packet.TTL -= 1
	if packet.TTL > 0 {
		m.QueuePacketToSend(*packet)
	}
}

func (m *MachineState) handleResourcePacket(packet *GossipPacket) {
	res := packet.Payload.Resource
	if res == nil || res.Type != ResourceStation {
		return
	}

	knownStations := m.GetKnownStations()
	if _, ok := knownStations[res.ID]; ok {
		return
	}

	fmt.Printf("[%s] Received Gossip Packet: Saw %s from %s\n", m.GetName(), res.Type, packet.OriginId)
	m.AddToKnownStations(WorldItem{
		ID:   res.ID,
		Type: res.Type,
		X:    res.X,
		Y:    res.Y,
	}, StationMemoryTTL.Seconds())
}

func (m *MachineState) handleHeartbeatPacket(packet *GossipPacket) {
	hb := packet.Payload.Heartbeat
	if hb == nil {
		return
	}

	switch hb.Type {
	case HeartbeatRequest:
		fmt.Printf("[%s] Heartbeat request received from peer: %s\n", m.Name, packet.OriginId)
		m.Outbox <- m.NewHeartbeatResponsePacket(packet.OriginId, m.GetBatteryLevel())

	case HeartbeatResponse:
		m.UpdatePendingHeartbeat(packet.OriginId, true)
		fmt.Printf("[%s] Heartbeat confirmed for peer: %s\n", m.Name, packet.OriginId)
	}
}

func (m *MachineState) handleSosPacket(packet *GossipPacket) {
	if m.GetName() != "SPARKER" || packet.Payload.Sos == nil {
		return
	}

	sos := packet.Payload.Sos

	// Self-rescue guard
	if m.ID == sos.TargetMachineId {
		packet.TTL = 0
		return
	}

	if m.GetState() == StateLowPower {
		return
	}

	rescueInfo := m.GetRescueInfo()
	isAlreadyRescuingThisBot := rescueInfo != nil && rescueInfo.MachineIdToRescue == sos.TargetMachineId

	fmt.Println(m.GetId() != sos.TargetMachineId)
	fmt.Println(isAlreadyRescuingThisBot)

	if m.GetId() != sos.TargetMachineId && !isAlreadyRescuingThisBot {
		fmt.Printf("[%s] SOS message received from %s! Dispatching for rescue.\n", m.GetName(), sos.OriginName)
		m.SetTarget(sos.OriginX, sos.OriginY)
		m.SetRescueInfo(&RescueData{
			HasRescueTarget:   true,
			MachineIdToRescue: sos.TargetMachineId,
			DrainFromStation:  0.0,
			AtRescueTarget:    false,
		})
		m.SetState(StateRescuing)
	}
}
