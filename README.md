# Autonomous Multi-Agent Mesh Simulation

A decentralized multi-agent simulation built in **Go** and visualized with **Pygame** (Python). Machines wander a shared 2D world, talk to each other only through a **gossip protocol**, run on small kinematic state machines, and recover downed peers on their own. Inspired by the machine mesh in the Horizon games.

The Go engine publishes a telemetry frame to a **Redis stream** 60 times a second. The Pygame visualizer reads that stream and draws the world.

---

## Key features

- **Gossip protocol network.** Packets propagate hop by hop with a TTL, are deduplicated with a per-machine seen-packet set (`SeenPackets`), and use heartbeat request/response pairs (`PendingPing`) to check on visible peers.
- **Autonomous recovery ("Sparker" units).** A Sparker listens for network-wide `SOS` packets, pauses what it is doing, drives to the reported location, jump-starts the dead unit's battery, and then heads back to a charging station.
- **Kinematic state engine.** A finite state machine drives movement, battery drain, charging and low-power behavior for three blueprints: Watcher, Scrapper and Sparker.
- **Concurrency by design.** Each machine runs its own goroutine, owns an `Inbox` and an `Outbox` channel, and guards its state with a mutex. The engine acts as the radio medium and delivers packets between machines that are in range.
- **Config-driven machines.** Speeds, vision, radio range and battery behavior live in a JSON registry, not in code.

---

## How it works

- **Patrol and station discovery.** Units wander toward random waypoints. When a unit sees a charging station it broadcasts a resource packet, so peers learn where it is without line of sight.
- **Low power.** At 20% battery a unit switches to `LOW_POWER`, slows down, and heads for the nearest station it knows about. If it knows none, it wanders and keeps listening.
- **Heartbeats and SOS.** A unit that sees a peer pings it. If no response comes back within 4 seconds (the peer died, or is just out of radio range), the reporting unit broadcasts an `SOS` packet across the mesh. That is why false alarms happen, and Sparkers have to handle them.
- **Rescue.** Sparkers parse `SOS` packets and go to the reported coordinates. If the target is really there, the Sparker recharges it with part of its own battery. If not, it treats the report as a false alarm and returns to a station.

### Data flow

```
 Go engine (swarm-engine)                Redis                  Pygame visualizer
┌────────────────────────────┐   XADD   ┌───────────────┐  XREAD  ┌──────────────────┐
│ machines <-> gossip mesh   │ ───────► │ swarm:telemetry│ ─────► │ simulation-ui/   │
│ engine tick at 60 Hz       │  "frame" │ (about 100     │         │ visualizer.py    │
└────────────────────────────┘  (JSON)  │  newest frames)│         └──────────────────┘
                                        └───────────────┘
```

---

## Repository layout

```
.
├── docker-compose.yml          # Redis container (swarm-redis, port 6379)
├── swarm-engine/               # Go simulation engine (module: swarm-engine)
│   ├── main.go                 # Entry point: spawns machines, runs the 60 Hz tick, publishes telemetry
│   ├── configs/
│   │   └── cauldron_registry.json   # Blueprints for WATCHER, SCRAPPER, SPARKER
│   └── internal/
│       ├── machine/            # Machine state, state machines, gossip packets and handlers, tests
│       └── simulation/         # World engine: tick, radio broadcast, visibility queries, tests
└── simulation-ui/              # Python visualizer
    ├── visualizer.py           # Pygame renderer (reads swarm:telemetry)
    ├── requirements.txt
    └── mock_telemetry.py       # Old standalone Redis test script (not used by the simulation)
```

---

## Prerequisites

| Tool | Version | Notes |
| --- | --- | --- |
| Go | 1.26.1 or newer | `go.mod` pins `go 1.26.1`. With Go 1.21 or newer, an older install downloads the right toolchain automatically (needs internet). |
| Python | 3.x (tested with 3.13) | Used only for the visualizer. |
| Docker with Compose | recent | Runs Redis. Any local Redis on `localhost:6379` also works. |

The visualizer opens a window, so it needs a desktop session (it will not run over plain SSH).

---

## Quick start

Run each numbered step in its own terminal, starting from the repository root.

### 1. Clone

```bash
git clone https://github.com/evan2945/robotics-swarm-telemetry-ai.git
cd robotics-swarm-telemetry-ai
```

### 2. Start Redis (repository root)

```bash
docker compose up -d
```

If your Docker is older and has no `compose` subcommand, use `docker-compose up -d` instead.

### 3. Start the Go engine (from `swarm-engine/`)

```bash
cd swarm-engine
go run .
```

You must run this from inside `swarm-engine/`. The engine loads `configs/cauldron_registry.json` relative to the current directory, so running it from anywhere else fails with `failed to open registry file`.

You will see a lot of console output: packet propagation, heartbeat checks and state changes such as `[SPARKER] SOS message received...`. That is intentional, so you can watch the machines talk to each other.

### 4. Start the visualizer (from `simulation-ui/`)

Open a new terminal at the repository root:

```bash
cd simulation-ui
python3 -m venv venv
source venv/bin/activate        # Windows: .\venv\Scripts\activate
pip install -r requirements.txt
python visualizer.py
```

On later runs you only need `source venv/bin/activate` and `python visualizer.py`.

You can start the visualizer before or after the engine. Until the first frame arrives it shows a gray placeholder dot.

### 5. Stop everything

- Engine: `Ctrl+C` in its terminal.
- Visualizer: close the window or press `Esc`.
- Redis: `docker compose down` from the repository root.

---

## Check that it is working

With the engine running, these should return data:

```bash
docker exec swarm-redis redis-cli XLEN swarm:telemetry
docker exec swarm-redis redis-cli XREVRANGE swarm:telemetry + - COUNT 1
```

`XLEN` should be a number near 100 (the engine trims the stream to roughly the newest 100 frames). The second command prints the latest frame, a JSON object stored under the field `frame` that contains `timestamp`, `machines` and `items`.

### What you see in the visualizer

| Element | Meaning |
| --- | --- |
| Green circle `WA` | Watcher |
| Violet circle `SC` | Scrapper |
| Light gray circle `SP` | Sparker |
| Orange | Unit in `LOW_POWER` (overrides its base color) |
| Cyan | Unit `CHARGING` |
| Red | Unit `DEAD` |
| Large blue circle | Charging station |

A Sparker keeps its gray color while rescuing or recharging another unit. Watch its console messages to follow those states.

---

## Configuration

Machine blueprints are defined in `swarm-engine/configs/cauldron_registry.json`:

- **Locomotion:** `speed`, `low_power_speed`, `evasion_threshold`.
- **Vision:** `range` and `angle_degrees`.
- **Communication:** radio `range` and `low_power_range`.
- **Battery:** `initial_level`, `drain_rate`, `low_power_drain`, `charge_rate`, plus `recharging_drain_rate` and `transfer_efficiency` for Sparkers.

The default drain rates are deliberately high so that units reach low power and die within about a minute. This lets you watch the Sparker rescue cycle in real time without waiting. Lower `drain_rate` for a slower, more realistic pace.

A few values are still constants in code: the 20% low-power threshold (`internal/machine/behaviors.go`) and the station and heartbeat memory TTLs, 10 s and 4 s (`internal/machine/blueprints.go`).

### Changing the fleet size

`main.go` spawns machines with `createMachine(1, name, specs, stationId)` for each class in the registry. The first argument is the number of machines per class, so the default fleet is one Watcher, one Scrapper and one Sparker. Change it and restart the engine to test gossip routing with more machines.

Larger fleets currently hit the limitations listed below, so treat them as experimental.

---

## Tests

From `swarm-engine/`:

```bash
go test ./...
go test -race ./...     # also checks for data races
```

The tests cover packet handling (resource, heartbeat and SOS), the machine lifecycle and battery physics, vector math, engine ticks and gossip routing.

---

## Troubleshooting

| Symptom | Likely cause and fix |
| --- | --- |
| `go: go.mod requires go >= 1.26.1` | Your Go is older and cannot fetch a newer toolchain. Upgrade Go, or use Go 1.21+ with internet access so it downloads the toolchain itself. |
| `failed to open registry file ... configs/cauldron_registry.json` | You are not in `swarm-engine/`. Run `cd swarm-engine && go run .`. |
| Engine prints `Redis stream publish failed ... connection refused` repeatedly | Redis is not running. Run `docker compose up -d` from the repository root and check `docker ps`. |
| Visualizer shows only a gray dot | The engine is not running or is not publishing. Check `XLEN swarm:telemetry` as shown above. |
| `ModuleNotFoundError: No module named 'pygame'` (or `redis`) | The virtual environment is not active, or requirements are not installed. Activate `simulation-ui/venv` and run `pip install -r requirements.txt`. |
| `python: can't open file 'main.py'` | The entry point is `simulation-ui/visualizer.py`. |
| `pygame.error: No available video device` | You are on a headless session (SSH, container). Run the visualizer on a machine with a display. |
| Docker warns that `version` is obsolete | Harmless. You can delete the `version: '3.8'` line from `docker-compose.yml`. |
| Port 6379 already in use | A local Redis is already running. Stop it, or change the port mapping in `docker-compose.yml` and the `localhost:6379` address in `swarm-engine/main.go` and `simulation-ui/visualizer.py`. |
| Console is very noisy | Expected. The engine logs gossip traffic on purpose. |

---

## Known limitations

- **Fleet scaling is unfinished.** When you spawn more than one machine per class, all copies start at the same position, and all machines of a class share one `KnownStations` map. For Sparkers, that shared map is guarded by per-machine mutexes, which is not safe.
- **Runs are not reproducible.** The random generator is unseeded, movement uses wall-clock time deltas, and each machine ticks on its own goroutine timer, so the same setup plays out differently every run.
- **No run end.** The engine runs until you press `Ctrl+C`.
- **Redis address is hardcoded** to `localhost:6379` in both the engine and the visualizer.

## Roadmap

Not implemented yet:

- Seeded, repeatable scenarios with a fixed run length.
- An LLM supervisory controller that reads telemetry and sends validated commands back to the machines, evaluated against a rule-based baseline.