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
    """
    Runs an executable file n times and logs the output.

    Parameters:
    executable_path (str): The path to the executable file.
    n (int): The number of times to run the executable file.

    Returns:
    None.
    """

    times = [0]
    for i in range(num_ittr):
        nodeType = random.randint(0, 1)
        startLat, startLng = generate_random_point_in_CU()
        destLat, destLng = generate_random_point_in_CU()
    
        result = None
        try: 
            result = subprocess.run([executable_path, str(nodeType), "172.22.150.238", str(startLat), str(startLng), str(destLat), str(destLng)], stdout=subprocess.PIPE, stderr=subprocess.PIPE, universal_newlines=True, timeout=90)
        except subprocess.TimeoutExpired:
            # If the program times out, send SIGKILL to the process
            result = subprocess.CompletedProcess(args=[], returncode=-9, stdout=b"", stderr=b"SIGKILL sent to process\n")
        
        output_lines = result.stdout.split('\n')
        for line in output_lines:
            match = re.search(r'Time: (\d+\.\d+)ms', line)
            if match:
                times.append(float(match.group(1)))
    average = sum(times) / len(times)
    print(average)
run_executable("client/client", 2)
