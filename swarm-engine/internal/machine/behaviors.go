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
	distToStation := getEuclDistance(mX, mY, mTargetX, mTargetY)
	chargingStationThreshold := 5.0

	if distToStation > chargingStationThreshold && !m.GetRescueInfo().StationTargeted {
		// If the machine is in low power mode, it needs to head to
		// the closest charging station.
		knownStations := m.GetKnownStations()
		if len(knownStations) > 0 {
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
			m.SetRescueInfo(
				RescueData{
					StationTargeted:   true,
					HasRescueTarget:   false,
					MachineIdToRescue: "",
					DrainFromStation:  0.0,
				},
			)
			mTargetX, mTargetY = targetStation.X, targetStation.Y
		} else {
			// No stations known, keep moving toward your existing target
			// or pick a random drift vector if close to existing target so that
			// the machine doesn't freeze.
			mTargetX, mTargetY := m.GetTarget()
			if getEuclDistance(mX, mY, mTargetX, mTargetY) < 5.0 {
				m.SetTarget(rand.Float64()*params.ScreenWidth, rand.Float64()*params.ScreenHeight)
			}
		}
	} else if distToStation <= chargingStationThreshold {
		fmt.Println("HERERERE3333")
		// Machine is at the charging state, so transition to "CHARGING" state
		m.SetState(StateCharging)
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
		m.SetRescueInfo(
			RescueData{
				StationTargeted:   false,
				HasRescueTarget:   false,
				MachineIdToRescue: "",
				DrainFromStation:  0.0,
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
	// TODO: We need to be able to calculate how much battery was drained to get to the rescue target
	batteryLevel := m.GetBatteryLevel()
	rescueInfo := m.GetRescueInfo()
	blueprint := m.GetBlueprint()
	drain := rescueInfo.DrainFromStation
	rechargeLimit := drain + (drain * .1)

	targetMachine := m.Env.ConnectToMachine(rescueInfo.MachineIdToRescue)

	// We set a limit to how much we allow the battery to be drained to recharge another bot.
	// Once we drop below that limit, the sparker needs to head back to the station to recharge.
	//
	// If the target machine gets to be above 50%, a charge is no longer needed. We set the
	// sparkers state to low power, even though it might not be below the threshold, so that
	// it goes back to the charging station to recharge.
	if batteryLevel <= rechargeLimit || targetMachine.GetBatteryLevel() >= 50.0 {
		m.SetState(StateLowPower)
		targetMachine.SetState(blueprint.States.InitialState)
		return
	}

	targetMachine.SetState(StateCharging)
}

func (m *MachineState) HandleRescuingKinematics(params KinematicActionParams) {
	mX, mY := m.GetPosition()
	mTargetX, mTargetY := m.GetTarget()
	distToTarget := getEuclDistance(mX, mY, mTargetX, mTargetY)

	if distToTarget <= 10 {
		batteryDrainFromStation := 100.0 - m.GetBatteryLevel()
		rescueInfo := m.GetRescueInfo()
		m.SetRescueInfo(
			RescueData{
				StationTargeted:   rescueInfo.StationTargeted,
				HasRescueTarget:   rescueInfo.HasRescueTarget,
				MachineIdToRescue: rescueInfo.MachineIdToRescue,
				DrainFromStation:  batteryDrainFromStation,
			},
		)
		m.SetState(StateRecharging)
	}

	m.ExecuteMovement(
		MovementParams{
			kap:    params,
			mSpeed: m.GetBlueprint().Locomotion.Speed,
		})
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
