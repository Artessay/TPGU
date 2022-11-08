import csv

def load_stations(station_filename):
    stations = {}
    with open(station_filename) as f:
        r = csv.DictReader(f)
        for row in r:
            stations[int(row['id'])] = {
                'id': int(row['id']),
                'name': row['name'],
                'wgs': [float(row['wgs_lon']), float(row['wgs_lat'])],
                'gcj': [float(row['gcj_lon']), float(row['gcj_lat'])]
            }
    return stations

def load_routes(route_filename):
    routes = []
    with open(route_filename) as f:
        r = csv.DictReader(f)
        for row in r:
            routes.append({
                'id': int(row['id']),
                'name': row['name'],
                'stations': list(map(lambda n: int(n), row['stations'].split(';')))
            })
    return routes