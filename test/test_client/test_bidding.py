import random
import subprocess
import math
import csv
from multiprocessing import Pool

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

def generate_random_points(num_points, min_lat, max_lat, min_lon, max_lon):
    """
    Generates a set of random latitude and longitude points within a specified range.

    Parameters:
    num_points (int): The number of points to generate.
    min_lat (float): The minimum latitude value.
    max_lat (float): The maximum latitude value.
    min_lon (float): The minimum longitude value.
    max_lon (float): The maximum longitude value.

    Returns:
    A list of tuples, where each tuple contains a latitude and longitude value.
    """
    points = []
    for i in range(num_points):
        lat = round(random.uniform(min_lat, max_lat), 6)
        lon = round(random.uniform(min_lon, max_lon), 6)
        points.append((lat, lon))
    return points

def write_member_data_to_csv(data):
    # Open the CSV file in write mode
    filename = "bidding_pool.csv"
    with open(filename, mode="w", newline="") as file:
        # Create a CSV writer object
        writer = csv.writer(file)
        
        # Write the header row to the CSV file
        writer.writerow(["nodeType", "startLat", "startLng", "destLat", "destLng"])
        
        # Write each row of data to the CSV file
        for row in data:
            writer.writerow([row["nodeType"], row["startLat"], row["startLng"], row["destLat"], row["destLng"]])

def run_executable(executable_path, num_clients):
    """
    Runs an executable file n times and logs the output.

    Parameters:
    executable_path (str): The path to the executable file.
    n (int): The number of times to run the executable file.

    Returns:
    None.
    """

    data = []
    for i in range(num_clients):
        nodeType = random.randint(0, 1)
        startLat, startLng = generate_random_point_in_CU()
        destLat, destLng = generate_random_point_in_CU()
        data.append({"nodeType": nodeType, "startLat": startLat, "startLng": startLat, "destLat":  })
    
        write_member_data_to_csv()

    with open('output.log', 'w') as f:
        pool = Pool(processes=num_clients)
        for i in range(num_clients):
            nodeType = random.randint(0, 1)
            startLat, startLng = generate_random_point_in_CU()
            destLat, destLng = generate_random_point_in_CU()
            write_member_data_to_csv()
            pool.apply_async(subprocess.run, args=[[executable_path] + [str(nodeType)] + ["172.22.150.238"] + [str(startLat)] + [str(startLng)]] + [str(destLat)] + [str(destLng)], kwds={'stdout': subprocess.PIPE, 'universal_newlines': True})
            
            # pool.apply_async(subprocess.run, args=[[executable_path] + [str(nodeType)] + ["0.0.0.0"] + [str(coords[i][0])] + [str(coords[i][1])] + [str(i)]], kwds={'stdout': subprocess.PIPE, 'universal_newlines': True})
        pool.close()
        pool.join()
        f.close() 


if __name__ == '__main__':  
    run_executable("client/client", 30)
