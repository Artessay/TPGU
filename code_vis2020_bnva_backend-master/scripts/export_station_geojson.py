# bus station location csv file syntax:
#   id, lat, lon

import json
import requests
import sys

from utils import load_stations

def main():
    if len(sys.argv) != 3:
        print('Usage: %s station_location_csv_file output_geojson_file' % sys.argv[0])
        exit(1)

    stations = load_stations(sys.argv[1])

    features = []
    for s in stations.values():
        features.append({
            'type': 'Feature',
            'properties': {
                'id': s['id'],
                'name': s['name'],
                'coordinates': s['wgs']
            },
            'geometry': {
                'type': 'Point',
                'coordinates': s['wgs']
            }
        })

    with open(sys.argv[2], 'w') as fout:
        fout.write(json.dumps({
            'type': 'FeatureCollection',
            'features': features
        }))

if __name__ == '__main__':
    main()