import sys
import threading
import pygame
import redis

# Initialize Pygame
pygame.init()

SCREEN_WIDTH = 800
SCREEN_HEIGHT = 600
FPS = 60

# Colors (RGB)
BG_COLOR = (30, 30, 30)
CIRCLE_COLOR = (0, 255, 128)

# Set up the display canvas
screen = pygame.display.set_mode((SCREEN_WIDTH, SCREEN_HEIGHT))
pygame.display.set_caption("Robotics Swarm Telemetry UI")
clock = pygame.time.Clock()

# Shared state between the network thread and the UI render loop
# Keys will be the machine name, values will be a tuple/list of (x, y)
telemetry_state = {}

def fetch_telemetry():
    """Background worker that continuously pulls data from the Redis Stream."""
    r_client = redis.Redis(host="localhost", port=6379, decode_responses=True)
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
        
        # Check if our background thread has successfully captured data yet
        if "machine_01" in telemetry_state:
            # Pull the coordinates directly out of our shared thread state
            pos = telemetry_state["machine_01"]
            pygame.draw.circle(screen, CIRCLE_COLOR, (int(pos[0]), int(pos[1])), 15)
        else:
            # Fallback text or shape if Redis hasn't gotten data yet
            pygame.draw.circle(screen, (100, 100, 100), (SCREEN_WIDTH // 2, SCREEN_HEIGHT // 2), 15)
        
        pygame.display.flip()
        clock.tick(FPS)

    pygame.quit()
    sys.exit()

if __name__ == "__main__":
    main()