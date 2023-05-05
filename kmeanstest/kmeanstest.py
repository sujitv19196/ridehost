import pandas as pd
import random
import subprocess
from multiprocessing import Pool

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

def generate_chicago_points():
    
    all_df = pd.read_csv("./Taxi_Trips.csv")
    
    all_df["Trip Start Timestamp"] = pd.to_datetime(all_df["Trip Start Timestamp"])
    # df = all_df[all_df["Trip Start Timestamp"] > pd.to_datetime("2023-03-25")]
    df = all_df

    # Convert latitude/longitude columns to float values
    df['Pickup Centroid Latitude'] = df['Pickup Centroid Latitude'].astype(float)
    df['Pickup Centroid Longitude'] = df['Pickup Centroid Longitude'].astype(float)
    df['Dropoff Centroid Latitude'] = df['Dropoff Centroid Latitude'].astype(float)
    df['Dropoff Centroid Longitude'] = df['Dropoff Centroid Longitude'].astype(float)

    df['start_date'] = df['Trip Start Timestamp'].apply(lambda x: x.date())
    
    counts = df.groupby('start_date')[['start_date']].count()
    counts_df = pd.DataFrame(counts)
    counts_df.columns = ["count of rides"]
    counts_df = counts_df.reset_index()
    counts_df.columns = ["start_date", "count_of_rides"]
    print("Average count of rides per day for month of March : ")
    print(counts_df["count_of_rides"].sum(), len(counts_df), counts_df["count_of_rides"].sum()/(len(counts_df)))
    
    df['start_date_with_hr'] = pd.to_datetime(df['start_date'])
    df['hr'] = df["Trip Start Timestamp"].apply(lambda x: x.hour)
    df['start_date_with_hr'] = df[['start_date_with_hr', 'hr']].apply(lambda x: x['start_date_with_hr']+pd.DateOffset(hours=x['hr']), axis=1)  
    counts = df.groupby('start_date_with_hr')[['start_date_with_hr']].count()
    counts_df = pd.DataFrame(counts)
    counts_df.columns = ["count of rides"]
    counts_df = counts_df.reset_index()
    counts_df.columns = ["start_date_with_hr", "count_of_rides"]
    print("Average count of rides per hour for month of March : ")
    print(counts_df["count_of_rides"].sum(), len(counts_df), counts_df["count_of_rides"].sum()/(len(counts_df)))
    print("Count of rides in the busiest hour for month of March : ")
    max_count_hr_date = counts_df["count_of_rides"].max()
    print(max_count_hr_date, counts_df[counts_df["count_of_rides"]==max_count_hr_date]["start_date_with_hr"])


    some_points = []
    for _,row in df.loc[(df["Trip Start Timestamp"]>=pd.to_datetime("2023-03-16 17:00:00")) & (df["Trip Start Timestamp"]<pd.to_datetime("2023-03-16 18:00:00"))].iterrows():
        start_lat = row["Pickup Centroid Latitude"]
        start_lon = row["Pickup Centroid Longitude"]
        dest_lat = row["Dropoff Centroid Latitude"]
        dest_lon = row["Dropoff Centroid Longitude"]
        some_points.append((start_lat, start_lon, dest_lat, dest_lon))

    return some_points
    

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
        pool = Pool(processes=len(coords))
        for i in range(len(coords)):
            nodeType = random.randint(0, 1)
            if nodeType == 1: ## rider
                pool.apply_async(subprocess.run, args=[[executable_path] + [str(nodeType)] + ["172.22.150.238"] + [str(coords[i][0])] + [str(coords[i][1])] + [str(coords[i][2])] + [str(coords[i][3])]], kwds={'stdout': subprocess.PIPE, 'universal_newlines': True})
            else: ## driver
                pool.apply_async(subprocess.run, args=[[executable_path] + [str(nodeType)] + ["172.22.150.238"] + [str(coords[i][0])] + [str(coords[i][1])]], kwds={'stdout': subprocess.PIPE, 'universal_newlines': True})

            # pool.apply_async(subprocess.run, args=[[executable_path] + [str(nodeType)] + ["0.0.0.0"] + [str(coords[i][0])] + [str(coords[i][1])] + [str(i)]], kwds={'stdout': subprocess.PIPE, 'universal_newlines': True})
        pool.close()
        pool.join()
    f.close() 


if __name__ == '__main__':  
    # coords = generate_random_points(8, -90, 90, -180, 180)
    coords = generate_chicago_points()
    print("Number of coordinates",len(coords))
    print(coords[0], coords[1])

    # run_executable("client/client", coords)
