package machine

import "math/rand/v2"

type KinematicAction func(params KinematicActionParams)

var KinematicRegistry = map[State]func(m *MachineState, params KinematicActionParams){
	"PATROL":    (*MachineState).HandlePatrolKinematics,
	"LOW_POWER": (*MachineState).HandleLowPowerKinematics,
	"CHARGING":  (*MachineState).HandleChargingKinematics,
}

type KinematicActionParams struct {
	Dt           float64
	ScreenWidth  float64
	ScreenHeight float64
}

func (m *MachineState) HandlePatrolKinematics(params KinematicActionParams) {
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

	if getEuclDistance(mX, mY, mTargetX, mTargetY) > 5.0 {
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
