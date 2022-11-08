# raw bus station csv file syntax:
#   route_name, station_num, station_name, gcj02_coordinates ('POINT (lon lat)')

import csv
import sys
import re

import eviltransform

def parse_gcj_coord_in_wkt(wkt_string):
    m = re.match('POINT \(([\d\.]+)\s+([\d\.]+)\)', wkt_string)
    if not m:
        raise RuntimeError('invalid wkt coordinates %s' % wkt_string)
    gcj_lon = float(m.group(1))
    gcj_lat = float(m.group(2))
    wgs_lat, wgs_lon = eviltransform.gcj2wgs_exact(gcj_lat, gcj_lon)
    return {
        'gcj': [gcj_lon, gcj_lat],
        'wgs': [wgs_lon, wgs_lat],
        'wkt': wkt_string
    }

def load_raw(raw_filename):
    f = open(raw_filename)
    r = csv.reader(f)
    return [{
        'route_name': row[0],
        'station_num': int(row[1]),
        'station_name': row[2],
        'coords': parse_gcj_coord_in_wkt(row[3])
    } for row in r]

def station_from_raw_record(id, rec):
    return {
        'id': id,
        'name': rec['station_name'],
        'wgs_lon': rec['coords']['wgs'][0],
        'wgs_lat': rec['coords']['wgs'][1],
        'gcj_lon': rec['coords']['gcj'][0],
        'gcj_lat': rec['coords']['gcj'][1],
    }

def build_stations_from_raw(raw):
    stations = {}
    stations_by_name = {}
    assigned_id = 1
    for rec in raw:
        if rec['station_name'] in stations_by_name:
            for matched_rec in stations_by_name[rec['station_name']]:
                if matched_rec['coords']['wkt'] == rec['coords']['wkt'] or eviltransform.distance(matched_rec['coords']['wgs'][1], matched_rec['coords']['wgs'][0], rec['coords']['wgs'][1], rec['coords']['wgs'][0]) < 1:
                    rec['station_id'] = matched_rec['station_id']
                    break
            else:
                stations_by_name[rec['station_name']].append(rec)
                stations[assigned_id] = station_from_raw_record(assigned_id, rec)
                rec['station_id'] = assigned_id
                assigned_id += 1
        else:
            stations_by_name[rec['station_name']] = [rec]
            stations[assigned_id] = station_from_raw_record(assigned_id, rec)
            rec['station_id'] = assigned_id
            assigned_id += 1
    return stations

def build_routes_from_raw(raw, stations):
    routes = {}
    assigned_id = 1
    for rec in raw:
        if rec['route_name'] in routes:
            routes[rec['route_name']]['stations'].append((rec['station_num'], rec['station_id']))
        else:
            routes[rec['route_name']] = {
                'id': assigned_id,
                'stations': [(rec['station_num'], rec['station_id'])]
            }
            assigned_id += 1

    for name, route in routes.items():
        route['stations'] = sorted(route['stations'], key=lambda s: s[0])
        if route['stations'][0][0] == route['stations'][1][0]:
            print('warning: duplicated stations found in route %s' % name)
        if route['stations'][-1][0] != len(route['stations']):
            print('warning: route %s incomplete; missing %d stations' % (name, route['stations'][-1][0] - len(route['stations'])))
    return routes

def write_stations(stations, filename):
    f = open(filename, 'w')
    w = csv.writer(f)
    headers = ['id', 'name', 'wgs_lon', 'wgs_lat', 'gcj_lon', 'gcj_lat']
    w.writerow(headers)
    for s in stations.values():
        w.writerow([s[h] for h in headers])

def write_routes(routes, filename):
    f = open(filename, 'w')
    w = csv.writer(f)
    w.writerow(['id', 'name', 'stations'])
    for n, r in routes.items():
        w.writerow([r['id'], n, ';'.join([str(p[1]) for p in r['stations']])])

def main():
    if len(sys.argv) != 4:
        print('Usage: %s raw_station_csv_file station_csv_file route_csv_file')
        exit(1)

    raw = load_raw(sys.argv[1])
    stations = build_stations_from_raw(raw)
    routes = build_routes_from_raw(raw, stations)
    write_stations(stations, sys.argv[2])
    write_routes(routes, sys.argv[3])

if __name__ == '__main__':
    main()