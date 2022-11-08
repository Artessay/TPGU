import csv
import sys

from utils import load_stations, load_routes

def cook_trips(filename, outfilename, stations, routes):
    fin = open(filename)
    fout = open(outfilename, 'w')

    r = csv.DictReader(fin)

    w = csv.DictWriter(fout, fieldnames=['card_id', 'time', 'route_id', 'from_station_id', 'from_stop_num', 'to_station_id', 'to_stop_num'])
    w.writeheader()

    routes_by_name = {r['name']: r for r in routes}
    for i, row in enumerate(r):
        print('\rprocessing row %d... ' % i, end='')
        if row['line'] in routes_by_name:
            route = routes_by_name[row['line']]
            from_station_id, to_station_id = -1, -1
            from_stop_num, to_stop_num = -1, -1
            for k, s in enumerate(route['stations']):
                if stations[s]['name'] == row['from_station']:
                    from_station_id = s
                    from_stop_num = k + 1
                if stations[s]['name'] == row['to_station']:
                    to_station_id = s
                    to_stop_num = k + 1
                if from_stop_num != -1 and to_stop_num != -1:
                    w.writerow({
                        'card_id': row['card_id'],
                        'time': row['time'],
                        'route_id': route['id'],
                        'from_station_id': from_station_id,
                        'from_stop_num': from_stop_num,
                        'to_station_id': to_station_id,
                        'to_stop_num': to_stop_num
                    })
                    break
            else:
                if from_stop_num == -1:
                    print('warning: station %s not found in route %s' % (row['from_station'], row['line']))
                if to_stop_num == -1:
                    print('warning: station %s not found in route %s' % (row['to_station'], row['line']))
        else:
            print('warning: route %s not found' % row['line'])

def main():
    if len(sys.argv) != 5:
        print('Usage: %s station_csv_file route_csv_file raw_trips_csv_file output_trips_csv_file')
        exit(1)

    stations = load_stations(sys.argv[1])
    routes = load_routes(sys.argv[2])
    cook_trips(sys.argv[3], sys.argv[4], stations, routes)

if __name__ == "__main__":
    main()

