import redis
import time

r_client = redis.Redis(host="localhost", port=6379, decode_responses=True)

x = 0
y = 0

stream_name = "swarm:machine_01:telemetry"

r_client.delete(stream_name)

# Write inital message to establish a starting point. "*" tells redis to auto-gen the message id
last_id = r_client.xadd(stream_name, {"x": x, "y": y}, id="*")

while True:
    x += 1
    y += 2

    new_id = r_client.xadd("swarm:machine_01:telemetry", {"x": x, "y": y}, id="*", maxlen=100)

    time.sleep(0.5)

    # The following was used to validate this file in isolations, reading back the written data from above.
    # r_response = r_client.xread(streams={"swarm:machine_01:telemetry": last_id})
    # print(f"done sleeping, previous values: {r_response}")

    # if r_response:
    #     for stream, messages in r_response:
    #         for msg_id, data in messages:
    #             if msg_id == new_id:
    #                 print("Read matches the latest write")

    #             last_id = msg_id
    # else:
    #     print("No new data found.")
    