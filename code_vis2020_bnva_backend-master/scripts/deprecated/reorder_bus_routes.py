# bus station location csv file syntax:
#   id, lat, lon
# bus route csv file syntax:
#   id, "{station_ids}"
# station distance csv file syntax:
#   src_station_id, dst_station_id, distance_in_meters

import csv
import math
import sys

from utils import load_stations, load_routes

def build_dist_matrix(dist_filename):
    dist_matrix = {}
    with open(dist_filename) as f:
        r = csv.reader(f)
        for i, j, dist in r:
            if not int(i) in dist_matrix:
                dist_matrix[int(i)] = {}
            dist_matrix[int(i)][int(j)] = float(dist)
    return dist_matrix

def find_origin(route, stations):
    origin = route['stations'][0]
    for sid in route['stations'][1:]:
        if stations[sid][1] < stations[origin][1] or (abs(stations[sid][1] - stations[origin][1]) < 1e-4 and stations[origin][0] > stations[sid][0]):
            origin = sid
    return origin

def reorder_route(route, stations, dist_matrix):
    curr = find_origin(route, stations)
    route_stations = [curr]

    visited = {}
    visited[curr] = True

    while True:
        min_dist = 2147483647
        next_station = -1
        for sid in route['stations']:
            if not sid in visited and min_dist > dist_matrix[curr][sid]:
                min_dist = dist_matrix[curr][sid]
                next_station = sid
        if next_station == -1:
            break
        curr = next_station
        visited[curr] = True
        route_stations.append(curr)

    unreachable = [sid for sid in route['stations'] if not sid in visited]
    if unreachable:
        print("Route reorder failed for id %d: unable to reach station %s" % (route['id'], ','.join(map(str, unreachable))))

    route['stations'] = route_stations
    return route

def main():
    if len(sys.argv) != 5:
        print('Usage: %s bus_station_location_csv_file bus_route_csv_file station_distance_csv_file output_file' % sys.argv[0])
        exit(1)

    stations = load_stations(sys.argv[1])
    routes = load_routes(sys.argv[2])
    dm = build_dist_matrix(sys.argv[3])

    fout = open(sys.argv[4], 'w')
    for r in routes:
        result = reorder_route(r, stations, dm)
        fout.write('%d,"{%s}"\n' % (result['id'], ','.join(map(str, result['stations']))))

if __name__ == '__main__':
    main()
