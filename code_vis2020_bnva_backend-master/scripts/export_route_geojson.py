# bus station location csv file syntax:
#   id, lat, lon
# bus route csv file syntax:
#   id, "{station_ids}"

import json
import requests
import sys

from utils import load_stations, load_routes

def main():
    if len(sys.argv) != 4:
        print('Usage: %s station_location_csv_file bus_route_csv_file output_geojson_file' % sys.argv[0])
        exit(1)

    stations = load_stations(sys.argv[1])
    routes = load_routes(sys.argv[2])

    features = []
    for i, r in enumerate(routes):
        print('\r[%d/%d] fetching geojson... ' % (i + 1, len(routes)), end='')
        resp = requests.get('https://osrm.zjvis.org/match/v1/car/%s?geometries=geojson&overview=full' % ';'.join(map(lambda n: ','.join(map(str, stations[n]['wgs'])), r['stations'])))
        data = resp.json()
        if data['code'] != 'Ok':
            print('error: failed to retrieve route geojson for route %d: %s' % (r['id'], data['code']))
        else:
            features.append({
                'type': 'Feature',
                'properties': r,
                'geometry': data['matchings'][0]['geometry']
            })

    with open(sys.argv[3], 'w') as fout:
        fout.write(json.dumps({
            'type': 'FeatureCollection',
            'features': features
        }))

if __name__ == '__main__':
    main()