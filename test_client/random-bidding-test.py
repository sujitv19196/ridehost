import random 
import math
import subprocess
import re
from sys import executable

def generate_random_point_in_CU(): 
    # Define the center point (Champaign-Urbana) as a latitude-longitude pair
    center_lat = 40.1106
    center_long = -88.2073

    # Define the radius of the circle in which to generate random points (in miles)
    radius = 10

    # Define the conversion factor from miles to degrees (for latitude and longitude)
    deg_per_mile = 1 / 69.0

    # Generate a random distance (in miles) and bearing (in radians)
    distance = random.uniform(0, radius)
    bearing = random.uniform(0, 2*math.pi)
    
    # Calculate the latitude and longitude of the new point
    new_lat = center_lat + (distance * deg_per_mile * math.cos(bearing))
    new_long = center_long + (distance * deg_per_mile * math.sin(bearing))
    
    # Print the latitude-longitude pair in the desired format
    return (new_lat, new_long) 

def run_executable(executable_path, num_ittr):
    times = []
    for i in range(num_ittr):
        nodeType = random.randint(0, 1)
        startLat, startLng = generate_random_point_in_CU()
        destLat, destLng = generate_random_point_in_CU()

        result = None
        output = None
        try: 
            result = subprocess.run([executable_path, str(nodeType), "172.22.150.238", str(startLat), str(startLng), str(destLat), str(destLng)], stdout=subprocess.PIPE, stderr=subprocess.PIPE, universal_newlines=True, timeout=80)
            output = result.stdout
        except subprocess.TimeoutExpired as e:
            # If the program times out, save the output before sending SIGINT to the process
            output = e.stdout
            # result = subprocess.CompletedProcess(args=[], returncode=-2, stdout=output, stderr=b"SIGINT sent to process\n")
            # subprocess.run(['kill', '-2', str(result.pid)])  # Send SIGINT to the process

        s = str(output)
        output_lines = s.split('\n')
        print(output_lines)
        for line in output_lines:
            match = re.search(r'Time:  (\d+\.\d+)ms', line)
            if match:
                times.append(float(match.group(1)))

    if (len(times) != 0):
        average = sum(times) / len(times)
        print(average)

run_executable("client/client", 10)
