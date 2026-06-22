package machine

import (
	"fmt"
	"math/rand/v2"
)

type KinematicAction func(params KinematicActionParams)

var KinematicRegistry = map[State]func(m *MachineState, params KinematicActionParams){
	"PATROL":     (*MachineState).HandlePatrolKinematics,
	"LOW_POWER":  (*MachineState).HandleLowPowerKinematics,
	"CHARGING":   (*MachineState).HandleChargingKinematics,
	"IDLE":       (*MachineState).HandleIdleKinematics,
	"RECHARGING": (*MachineState).HandleRechargingKinematics,
	"RESCUING":   (*MachineState).HandleRescuingKinematics,
}

type KinematicActionParams struct {
	Dt           float64
	ScreenWidth  float64
	ScreenHeight float64
}

func (m *MachineState) HandlePatrolKinematics(params KinematicActionParams) {
	batteryLevel := m.GetBatteryLevel()
	if batteryLevel <= 20 && m.GetState() == StatePatrol {
		fmt.Printf("[%s] WARNING: Battery critically low (%f%%). Entering LOW_POWER mode.\n", m.GetName(), batteryLevel)
		m.SetState(StateLowPower)
		return
	}

	blueprint := m.GetBlueprint().Locomotion

	mX, mY := m.GetPosition()
	mTargetX, mTargetY := m.GetTarget()

	if getEuclDistance(mX, mY, mTargetX, mTargetY) < 5.0 {
		mTargetX = rand.Float64() * params.ScreenWidth
		mTargetY = rand.Float64() * params.ScreenHeight
		m.SetTarget(mTargetX, mTargetY)
	}

	resp := m.ExecuteMovement(MovementParams{
		kap:    params,
		mSpeed: blueprint.Speed,
	})

	m.SetTarget(resp.mTargetX, resp.mTargetY)
	m.SetPosition(resp.mX, resp.mY)
	m.SetHeading(resp.mVx, resp.mVy)
}

func (m *MachineState) HandleLowPowerKinematics(params KinematicActionParams) {
	mX, mY := m.GetPosition()
	mTargetX, mTargetY := m.GetTarget()
	locBlueprint := m.GetBlueprint().Locomotion
	speed := locBlueprint.LowPowerSpeed
	chargingStationThreshold := 5.0

	if !m.GetStationTargeted() {
		knownStations := m.GetKnownStations()

		if len(knownStations) > 0 {
			// Found a station, pick the closest one and lock onto it.
			dist := -1.0
			var targetStation *StationData
			for _, station := range knownStations {
				newDist := getEuclDistance(mX, mY, station.X, station.Y)
				if dist == -1 || newDist < dist {
					dist = newDist
					targetStation = station
				}
			}

			m.SetTarget(targetStation.X, targetStation.Y)
			m.SetStationTargeted(true)

		} else {
			// No stations known. Manage clean wandering waypoints.
			mTargetX, mTargetY := m.GetTarget()
			if getEuclDistance(mX, mY, mTargetX, mTargetY) < 5.0 {
				// Pick a brand new random wandering destination
				m.SetTarget(rand.Float64()*params.ScreenWidth, rand.Float64()*params.ScreenHeight)
			}
		}
	}

	mTargetX, mTargetY = m.GetTarget()
	distToTarget := getEuclDistance(mX, mY, mTargetX, mTargetY)

	if m.GetStationTargeted() && distToTarget <= chargingStationThreshold {
		fmt.Printf("[%s] Successfully docked at station. Initiating charge.\n", m.GetId())
		m.SetState(StateCharging)
		return
	}

	resp := m.ExecuteMovement(MovementParams{
		kap:    params,
		mSpeed: speed,
	})

	m.SetPosition(resp.mX, resp.mY)
	m.SetHeading(resp.mVx, resp.mVy)
}

func (m *MachineState) HandleChargingKinematics(params KinematicActionParams) {
	m.SetHeading(0.0, 0.0)

	// Explicitly setting position to previous location to prevent minor drift
	mX, mY := m.GetPosition()
	m.SetPosition(mX, mY)

	if m.GetBatteryLevel() >= 100 {
		blueprint := m.GetBlueprint()
		m.SetBatteryLevel(100.0)
		m.SetStationTargeted(false)
		m.SetRescueInfo(
			&RescueData{
				HasRescueTarget:   false,
				MachineIdToRescue: "",
				DrainFromStation:  0.0,
				AtRescueTarget:    false,
			},
		)
		m.SetState(blueprint.States.InitialState)
	}
}

func (m *MachineState) HandleIdleKinematics(params KinematicActionParams) {
	if m.GetRescueInfo().HasRescueTarget {
		m.SetState(StateRescuing)
		m.ExecuteMovement(
			MovementParams{
				kap:    params,
				mSpeed: m.GetBlueprint().Locomotion.Speed,
			})
		return
	}

	m.SetHeading(0.0, 0.0)

	// Explicitly setting position to previous location to prevent minor drift
	mX, mY := m.GetPosition()
	m.SetPosition(mX, mY)
}

func (m *MachineState) HandleRechargingKinematics(params KinematicActionParams) {
	batteryLevel := m.GetBatteryLevel()
	rescueInfo := m.GetRescueInfo()
	drain := rescueInfo.DrainFromStation
	rechargeLimit := drain + (drain * .1)

	targetMachine := m.Env.ConnectToMachine(rescueInfo.MachineIdToRescue)
	tBlueprint := targetMachine.GetBlueprint()

	// We set a limit to how much we allow the battery to be drained to recharge another bot.
	// Once we drop below that limit, the sparker needs to head back to the station to recharge.
	//
	// If the target machine gets to be above 50%, a charge is no longer needed. We set the
	// sparkers state to low power, even though it might not be below the threshold, so that
	// it goes back to the charging station to recharge.
	if batteryLevel <= rechargeLimit || targetMachine.GetBatteryLevel() >= 50.0 {
		fmt.Printf("[%s] Target %s stabilized at %.2f%%. Releasing unit.\n",
			m.GetName(), targetMachine.GetName(), targetMachine.GetBatteryLevel())

		m.SetState(StateLowPower)

		// Pick a new random target waypoint
		var targetStation *StationData
		dist := -1.
		mX, mY := targetMachine.GetPosition()
		for _, station := range m.GetKnownStations() {
			newDist := getEuclDistance(mX, mY, station.X, station.Y)
			if dist == -1 || newDist < dist {
				dist = newDist
				targetStation = station
			}
		}
		targetMachine.SetTarget(targetStation.X, targetStation.Y)

		speed := tBlueprint.Locomotion.Speed
		vx, vy := CalculateHeading(mX, mY, targetStation.X, m.TargetY, speed)
		targetMachine.SetHeading(vx, vy)

		targetMachine.SetStationTargeted(true)
		targetMachine.SetState(StateLowPower)

		return
	}

	if targetMachine.GetBatteryLevel() <= 0.0 || targetMachine.GetState() == StateDead {
		fmt.Printf("[%s] Target %s is dead. Injecting initial jump-start voltage.\n", m.GetName(), targetMachine.GetName())

		targetMachine.SetBatteryLevel(1.0)    // Bring it just above 0 to clear the lifecycle guard clause
		targetMachine.SetState(StateCharging) // Set its state so it begins its own charging routine
	} else {
		// The bot is awake and charging itself now!
		// We just print logs and maintain the link while our own StateRecharging drain rate ticks down.
		fmt.Printf("[%s] Monitoring recovery -> Target Battery: %.2f%% | Sparker Battery: %.2f%%\n",
			m.GetName(), targetMachine.GetBatteryLevel(), batteryLevel)
	}
}

func (m *MachineState) HandleRescuingKinematics(params KinematicActionParams) {
	mX, mY := m.GetPosition()
	mTargetX, mTargetY := m.GetTarget()
	distToTarget := getEuclDistance(mX, mY, mTargetX, mTargetY)

	// If we are within range of the reported target location, check to see if we can
	// visually "see" the downed machine. If so, begin recharging it.
	// If not, the machine wasn't actually downed and a reporting machine never
	// got a heartbeat response from the machine, even though it wasn't down. For
	// this situation, switch back to "low power" state to return to the station.
	if distToTarget <= 10 {
		rescueInfo := m.GetRescueInfo()
		if rescueInfo != nil && m.Env.IsMachineInRange(rescueInfo.MachineIdToRescue) {
			batteryDrainFromStation := 100.0 - m.GetBatteryLevel()

			m.SetRescueInfo(
				&RescueData{
					HasRescueTarget:   rescueInfo.HasRescueTarget,
					MachineIdToRescue: rescueInfo.MachineIdToRescue,
					DrainFromStation:  batteryDrainFromStation,
					AtRescueTarget:    true,
				},
			)
			m.SetState(StateRecharging)
			return
		} else {
			fmt.Printf("[%s] Rescue target missing or false alarm. Aborting.\n", m.Name)
			m.SetState(StateLowPower)
			return
		}
	}

	resp := m.ExecuteMovement(
		MovementParams{
			kap:    params,
			mSpeed: m.GetBlueprint().Locomotion.Speed,
		})

	m.SetPosition(resp.mX, resp.mY)
	m.SetHeading(resp.mVx, resp.mVy)
}

type MovementParams struct {
	kap    KinematicActionParams
	mSpeed float64
}

type MovementResponse struct {
	mX       float64
	mY       float64
	mVx      float64
	mVy      float64
	mTargetX float64
	mTargetY float64
}

func (m *MachineState) ExecuteMovement(params MovementParams) (resp MovementResponse) {
	mX, mY := m.GetPosition()
	mVx, mVy := m.GetHeading()
	mTargetX, mTargetY := m.GetTarget()
	mVx, mVy = CalculateHeading(mX, mY, mTargetX, mTargetY, params.mSpeed)
	mX = mX + (mVx * params.kap.Dt)
	mY = mY + (mVy * params.kap.Dt)

	if mX <= 0 {
		mX = 0
		mVx = -mVx
		mTargetX = -mTargetX
	} else if mX >= params.kap.ScreenWidth {
		mX = params.kap.ScreenWidth
		mVx = -mVx
		mTargetX = params.kap.ScreenWidth - (mTargetX - params.kap.ScreenWidth)
	}

	if mY <= 0 {
		mY = 0
		mVy = -mVy
		mTargetY = -mTargetY
	} else if mY >= params.kap.ScreenHeight {
		mY = params.kap.ScreenHeight
		mVy = -mVy
		mTargetY = params.kap.ScreenHeight - (mTargetY - params.kap.ScreenHeight)
	}

	return MovementResponse{
		mX:       mX,
		mY:       mY,
		mVx:      mVx,
		mVy:      mVy,
		mTargetX: mTargetX,
		mTargetY: mTargetY,
	}
}
