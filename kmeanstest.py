import random
import subprocess

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


def run_executable(executable_path, coords):
    """
    Runs an executable file n times and logs the output.

    Parameters:
    executable_path (str): The path to the executable file.
    n (int): The number of times to run the executable file.

    Returns:
    None.
    """
    with open('output.log', 'w') as f:
        for i in range(len(coords)):
            nodeType = random.randint(0, 1)
            subprocess.run([executable_path] + [str(nodeType)] + ["172.22.150.238"] + [str(coords[i][0])] + [str(coords[i][1])], stdout=f)
    f.close()   

coords = generate_random_points(10, -90, 90, -180, 180)
run_executable("../client/client", coords)