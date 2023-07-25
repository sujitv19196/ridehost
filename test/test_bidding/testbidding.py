import random
import statistics
import threading
import math
import time
import matplotlib as mpl
import matplotlib.pyplot as plt
import numpy as np
import pandas as pd

dollar_per_km = 1

class Rider:
    def __init__(self, id, start, dest):
        self.id = id
        self.start = start
        self.dest = dest
        self.maxCost = 0
        self.acceptedBid = False
        self.trip_price = -1
        self.lock = threading.Lock()

    def recv_bid(self, bid):
        with self.lock:
            if self.acceptedBid: 
                return False
            if bid <= self.maxCost:
                self.acceptedBid = True 
                return True 
            return False 

class Driver:
    def __init__(self, id, start, dest, riders):
        self.id = id
        self.start = start
        self.dest = dest
        self.rider_list = riders
        self.num_trys = 0
        self.match = None

    def make_bid(self, rider):
        distance = calculate_distance(self.start, rider.start) + calculate_distance(rider.start, rider.dest)
        bid = distance * dollar_per_km
        return rider.recv_bid(bid), bid

    def auction(self):
        # sort riders by distnace away 
        distances = [(rider, calculate_distance(self.start, rider.start)) for rider in self.rider_list]
        sorted_riders = sorted(distances, key=lambda x: x[1])
        self.rider_list = [rider[0] for rider in sorted_riders]
        
        for i in range(len(self.rider_list)):
            rider = self.rider_list[i]
            self.num_trys+=1 
            accept, price = self.make_bid(rider)
            if (accept):
                self.match = rider
                self.trip_price = price
                return  
            # Random delay with normal distribution to simulate mobile network 
            delay = random.gauss(50, 25)
            # Ensure delay is positive
            delay = max(delay, 0)
            # Convert delay from milliseconds to seconds
            delay = delay / 1000
            # Sleep for the delay time
            time.sleep(delay)


def simulateRandomAuction():
    riders = []
    drivers = []
    for i in range(num_riders):
        r = Rider(i, generate_random_point_in_CU(), generate_random_point_in_CU())
        r.maxCost = calculate_distance(r.start, r.dest) + random.uniform(5.0, 30.0)
        riders.append(r)
    for i in range(num_drivers):
        d = Driver(i, generate_random_point_in_CU(), generate_random_point_in_CU(), riders)
        drivers.append(d)

    threads = []
    for d in drivers:
        t = threading.Thread(target=d.auction)
        threads.append(t)
        t.start()

    for t in threads:
        t.join()
    
    miss = 0 
    rtt = []
    for d in drivers: 
        if d.match != None: 
            # print("Driver %d matched with Rider %d in %d RTTs" % (d.id, d.match.id, d.num_trys))
            rtt.append(d.num_trys)
        else:
            # print("Driver %d NO MATCH in %d RTTs" % (d.id, d.num_trys))
            miss+=1 
    if (len(rtt) > 0):
        rtts.append(statistics.mean(rtt))
    not_matched.append(miss)

def simulateTaxiAuction(num_drivers, num_riders, some_points):
    riders = []
    drivers = []
    for i in range(num_riders):
        rand_choice = random.choice(some_points)
        some_points.remove(rand_choice)
        start, dest = rand_choice
        r = Rider(i, start, dest)
        r.maxCost = calculate_distance(r.start, r.dest) + random.uniform(5.0, 30.0)
        riders.append(r)
    for i in range(num_drivers):
        rand_choice = random.choice(some_points)
        some_points.remove(rand_choice)
        start, dest = rand_choice
        d = Driver(i, dest, generate_random_point_in_CU(), riders)
        drivers.append(d)

    threads = []
    for d in drivers:
        t = threading.Thread(target=d.auction)
        threads.append(t)
        t.start()

    for t in threads:
        t.join()
    
    miss = 0 
    rtt = []
    prices = []
    for d in drivers: 
        if d.match != None: 
            # print("Driver %d matched with Rider %d in %d RTTs" % (d.id, d.match.id, d.num_trys))
            rtt.append(d.num_trys)
            prices.append(d.trip_price)
        else:
            # print("Driver %d NO MATCH in %d RTTs" % (d.id, d.num_trys))
            miss+=1 
    if (len(rtt) > 0):
        return statistics.mean(rtt), miss, statistics.mean(prices)
    else:
        return -1, miss, -1 

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


def calculate_distance(d1, d2):
    EARTH_RADIUS = 6371  # Earth's radius in kilometers
    lat1, lon1 = d1
    lat2, lon2 = d2 
    # Convert latitude and longitude from degrees to radians
    lat1, lon1, lat2, lon2 = map(math.radians, [lat1, lon1, lat2, lon2])

    # Calculate the differences between the latitudes and longitudes
    dlat = lat2 - lat1
    dlon = lon2 - lon1

    # Apply the Haversine formula
    a = math.sin(dlat/2)**2 + math.cos(lat1) * math.cos(lat2) * math.sin(dlon/2)**2
    c = 2 * math.asin(math.sqrt(a))
    distance = EARTH_RADIUS * c

    return distance

def run_taxi_sim():
    all_df = pd.read_csv("test_bidding/Taxi_Trips.csv")
        
    all_df["Trip Start Timestamp"] = pd.to_datetime(all_df["Trip Start Timestamp"])
    # df = all_df[all_df["Trip Start Timestamp"] > pd.to_datetime("2023-03-25")]
    df = all_df

    # Convert latitude/longitude columns to float values
    df['Pickup Centroid Latitude'] = df['Pickup Centroid Latitude'].astype(float)
    df['Pickup Centroid Longitude'] = df['Pickup Centroid Longitude'].astype(float)
    df['Dropoff Centroid Latitude'] = df['Dropoff Centroid Latitude'].astype(float)
    df['Dropoff Centroid Longitude'] = df['Dropoff Centroid Longitude'].astype(float)

    some_points = []
    for _,row in df.loc[(df["Trip Start Timestamp"]>=pd.to_datetime("2023-03-16 17:00:00")) & (df["Trip Start Timestamp"]<pd.to_datetime("2023-03-16 18:00:00"))].iterrows():
        start_lat = row["Pickup Centroid Latitude"]
        start_lon = row["Pickup Centroid Longitude"]
        dest_lat = row["Dropoff Centroid Latitude"]
        dest_lon = row["Dropoff Centroid Longitude"]
        if((not math.isnan(start_lat)) & (not math.isnan(start_lon)) & (not math.isnan(dest_lat)) & (not math.isnan(dest_lon))):    
            some_points.append([(start_lat, start_lon), (dest_lat, dest_lon)])

    n_values = []
    rtt_means = []
    miss_means = []
    price_means = []
    
    # end data 
    rtts = []
    not_matched = []
    prices = []

    for n in range(2, 70, 2): 
        num_drivers = (int)(n/2)
        num_riders = (int)(n/2)
        for i in range(50):
            rtt, miss, price = simulateTaxiAuction(num_drivers, num_riders, some_points.copy())
            if (rtt != -1):
                rtts.append(rtt)
                prices.append(price)
            not_matched.append(miss)
        print("Pool size: %d" % n)
        print("Average RTT for Matched Pairs each round: %f" % (statistics.mean(rtts)))
        print("Average number of misses each round: %f" % (statistics.mean(not_matched)))
        print("Average trip price each round: %f" % (statistics.mean(prices)))
        print()
        n_values.append(n)
        rtt_means.append(statistics.mean(rtts))
        miss_means.append(statistics.mean(not_matched))
        price_means.append(statistics.mean(prices))

        # reset 
        rtts = []
        not_matched = []

    # Bar chart of average RTT for each pool size
    mpl.use('tkagg')
    plt.plot(n_values, rtt_means, 'o-')
    plt.xlabel('Pool size')
    plt.ylabel('Average RTT')
    plt.title('Average RTT for Matched Pairs each round for taxi dataset')
    plt.show()

    # Line chart of average number of misses for each pool size
    plt.plot(n_values, miss_means, 'o-')
    plt.xlabel('Pool size')
    plt.ylabel('Average number of unmatched pairs')
    plt.title('Average number of unmatched pairs each round for taxi dataset')
    plt.show()

     # Line chart of average number of misses for each pool size
    plt.plot(n_values, price_means, 'o-')
    plt.xlabel('Pool size')
    plt.ylabel('Average trip price')
    plt.title('Average trip pirce for taxi dataset')
    plt.show()

run_taxi_sim()