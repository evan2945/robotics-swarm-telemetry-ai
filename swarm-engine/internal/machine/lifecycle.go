package machine

import (
	"context"
	"fmt"
	"time"
)

func (m *MachineState) RunLifecycle(ctx context.Context) {
	ticker := time.NewTicker(m.GetTickerInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			batteryLevel := m.GetBatteryLevel()
			blueprint := m.GetBlueprint()

			if batteryLevel <= 0 {
				if m.GetState() != StateDead {
					fmt.Printf("[%s] CRITICAL: Battery fully depleted. Unit deactivated.\n", m.GetName())
					m.SetBatteryLevel(0.0)
					m.SetState(StateDead)
				}
				continue
			}

			// Handle state-based battery depletion
			switch m.GetState() {
			case StateCharging:
				m.SetBatteryLevel(batteryLevel + blueprint.Battery.ChargeRate)
			case StateLowPower:
				m.SetBatteryLevel(batteryLevel - blueprint.Battery.LowPowerDrain)
			case StateRecharging:
				m.SetBatteryLevel(batteryLevel - blueprint.Battery.RechargingDrainRate)
			case StateIdle:
			default:
				m.SetBatteryLevel(batteryLevel - blueprint.Battery.DrainRate)
			}

			batteryLevel = m.GetBatteryLevel()
			if batteryLevel <= 20.0 || m.GetName() == "SCRAPPER" {
				fmt.Printf("Machine %s battery: %f\n", m.GetName(), batteryLevel)
			}

			// Scan World Items
			seenItems := []WorldItem{}
			if m.Env != nil {
				seenItems = m.Env.GetVisibleItems(m.X, m.Y, blueprint.Vision.Range)
			}

			knownStations := m.GetKnownStations()
			for _, item := range seenItems {
				now := time.Now()
				if _, ok := knownStations[item.ID]; !ok {
					fmt.Printf("[%s] has seen world item %s!\n", m.GetName(), item.Type)
					m.AddToKnownStations(item, StationMemoryTTL.Seconds())

					m.Outbox <- m.NewResourcePacket(item)
				} else {
					m.UpdateKnownStationsTime(item.ID, now)
				}
			}

			// Scan Visible Peers (Heartbeats)
			seenMachines := []*MachineState{}
			if m.Env != nil && m.GetName() != "SPARKER" {
				seenMachines = m.Env.GetVisibleMachines(m.ID, m.X, m.Y, blueprint.Vision.Range)
			}

			mX, mY := m.GetPosition()
			for _, targetMachine := range seenMachines {
				if _, ok := m.GetPendingHeartbeats()[targetMachine.ID]; !ok {
					heartbeatRequest := PendingPing{
						LastPingedAt:  time.Now(),
						LastKnownX:    mX,
						LastKnownY:    mY,
						HeardResponse: false,
						TargetX:       targetMachine.X,
						TargetY:       targetMachine.Y,
					}
					m.SetPendingHeartbeat(targetMachine.ID, heartbeatRequest)

					fmt.Printf("[%s] Sending heartbeat request for peer: %s\n", m.GetName(), targetMachine.GetId())

					m.Outbox <- m.NewHeartbeatRequestPacket(targetMachine.GetId(), batteryLevel)
				}
			}

			m.GcPendingHeartbeats()

		case packet := <-m.Inbox:
			batteryLevel := m.GetBatteryLevel()
			if batteryLevel <= 0 {
				continue
			}

			m.handleIncomingPacket(&packet)
		}
	}
}
