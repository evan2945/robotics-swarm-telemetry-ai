import sys
import threading
import pygame
import redis
import json

# Initialize Pygame
pygame.init()
machine_font = pygame.font.SysFont("Arial", 14, bold=True)

SCREEN_WIDTH = 800
SCREEN_HEIGHT = 600
FPS = 60

# Colors (RGB)
BG_COLOR = (30, 30, 30)
CIRCLE_COLOR = (0, 255, 128)
WATCHER_COLOR = (0, 255, 128)
SCRAPPER_COLOR = (238, 130, 238)
LOW_POWER_COLOR = (255, 140, 0)
CHARGING_COLOR = (0, 255, 255)
DEAD_COLOR = (255, 0, 0)
DEFAULT_COLOR = (200, 200, 200)

# Set up the display canvas
screen = pygame.display.set_mode((SCREEN_WIDTH, SCREEN_HEIGHT))
pygame.display.set_caption("Robotics Swarm Telemetry UI")
clock = pygame.time.Clock()

# Shared state between the network thread and the UI render loop
# Keys will be the machine name, values will be a tuple/list of (x, y)
telemetry_state = {}

# This method was used to validate fetching telemetry from redis and isn't
# used in the actual simulation
def fetch_mock_telemetry():
    """Background worker that continuously pulls data from the Redis Stream."""
    r_client = redis.Redis(host="localhost", port=6379, db=0, decode_responses=True)
    stream_name = "swarm:machine_01:telemetry"
    
    # Start by listening only for messages that arrive AFTER this thread boots
    last_id = "$"
    
    print("[Telemetry Thread] Connected to Redis, listening for stream data...")
    
    while True:
        try:
            response = r_client.xread(streams={stream_name: last_id}, block=100)
            
            if response:
                # Extract the latest message entry from the raw nested list
                latest_message = response[0][1][-1]
                
                msg_id = latest_message[0]
                data = latest_message[1]
                
                # Update our cursor so the next read fetches newer messages
                last_id = msg_id
                
                # Parse the strings back into coordinates and update the shared state
                # Python dict assignments are atomic, making this simple approach thread-safe enough for our needs
                x_val = float(data.get("x", 0))
                y_val = float(data.get("y", 0))
                
                telemetry_state["machine_01"] = (x_val, y_val)
                
        except redis.RedisError as e:
            print(f"[Telemetry Thread] Redis Error: {e}")

def fetch_telemetry():
    """Background worker that continuously pulls unified frames from the Redis Stream."""
    # Ensure decode_responses=True so we get clean strings instead of bytes
    r_client = redis.Redis(host="localhost", port=6379, db=0, decode_responses=True)
    stream_name = "swarm:telemetry"  # Your unified stream key
    
    last_id = "$"
    print("[Telemetry Thread] Connected to Redis, listening for unified swarm frames...")
    
    while True:
        try:
            # Block for up to 100ms waiting for new 60Hz physics frames
            response = r_client.xread(streams={stream_name: last_id}, block=100)
            
            if response:
                latest_message = response[0][1][-1]
                msg_id = latest_message[0]
                data = latest_message[1]
                
                last_id = msg_id
                
                if "frame" in data:
                    frame_snapshot = json.loads(data["frame"])
                    
                    telemetry_state["latest_frame"] = frame_snapshot

        except redis.RedisError as e:
            print(f"[Telemetry Thread] Redis Error: {e}")


def main():
    # Spin up the telemetry consumer thread in the background
    telemetry_thread = threading.Thread(target=fetch_telemetry, daemon=True)
    telemetry_thread.start()

    running = True
    while running:
        for event in pygame.event.get():
            if event.type == pygame.QUIT:
                running = False
            elif event.type == pygame.KEYDOWN:
                if event.key == pygame.K_ESCAPE:
                    running = False

        screen.fill(BG_COLOR)
        
        current_frame = telemetry_state.get("latest_frame")
        
        if current_frame:
            if "items" in current_frame:
                STATION_COLOR = (0, 128, 255)
                for item in current_frame["items"]:
                    if item.get("type") == "CHARGING_STATION":
                        ix = int(item.get("x", 0))
                        iy = int(item.get("y", 0))
                        pygame.draw.circle(screen, STATION_COLOR, (ix, iy), 18)

            if "machines" in current_frame:
                color = (0, 0, 0)
                for mach in current_frame["machines"]:
                    x = int(mach.get("x", 0))
                    y = int(mach.get("y", 0))
                    m_type = mach.get("name")
                    state = mach.get("state", "PATROL")
                    if m_type == "WATCHER":
                        color = WATCHER_COLOR
                    elif m_type == "SCRAPPER":
                        color = SCRAPPER_COLOR
                    else:
                        color = DEFAULT_COLOR
                    
                    # Pick a color based on state (helps with our upcoming low-power milestone!)
                    if state == "LOW_POWER":
                        color = LOW_POWER_COLOR
                    elif state == "CHARGING":
                        color = CHARGING_COLOR
                    elif state == "DEAD":
                        color = DEAD_COLOR
                    
                    # Draw the physical machine
                    pygame.draw.circle(screen, color, (x, y), 12)
                
                    if m_type:
                        # Take the first letter of the machine name (e.g., "W", "S")
                        label_letter = m_type[:2].upper()
                        
                        # Render the letter to a text surface (using crisp white text)
                        text_surface = machine_font.render(label_letter, True, (0, 0, 0))
                        
                        # Get the surface bounding rect and pin its center to the circle's center
                        text_rect = text_surface.get_rect()
                        text_rect.center = (x, y)
                        
                        # Blit the text surface onto the screen
                        screen.blit(text_surface, text_rect)
        else:
            # Fallback graphic if Go engine isn't running or streaming yet
            pygame.draw.circle(screen, (100, 100, 100), (SCREEN_WIDTH // 2, SCREEN_HEIGHT // 2), 15)
        
        pygame.display.flip()
        clock.tick(FPS)

    pygame.quit()
    sys.exit()

if __name__ == "__main__":
    main()