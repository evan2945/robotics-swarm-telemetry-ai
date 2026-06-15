import sys
import pygame

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

def main():
    running = True
    
    while running:
        # 1. Event Handling (Prevents the window from freezing/crashing)
        for event in pygame.event.get():
            if event.type == pygame.QUIT:
                running = False
            elif event.type == pygame.KEYDOWN:
                if event.key == pygame.K_ESCAPE:
                    running = False

        # 2. Game State / Telemetry Updates (To be added later)

        # 3. Drawing / Rendering
        screen.fill(BG_COLOR)
        
        # Draw a test circle at the center of the screen (radius 20)
        center_pos = (SCREEN_WIDTH // 2, SCREEN_HEIGHT // 2)
        pygame.draw.circle(screen, CIRCLE_COLOR, center_pos, 20)
        
        # Flip the display buffer to the screen
        pygame.display.flip()
        
        # Enforce 60 FPS cap
        clock.tick(FPS)

    pygame.quit()
    sys.exit()

if __name__ == "__main__":
    main()