import {Route, Station} from "@/store/modules/Exploration";
import {GeoJSON} from "geojson";
import set = Reflect.set;

export class TargetRoute {
  public selectedStations : Set<number> | null = null

  constructor(public route : Route, public indexedStations: {[id: number]: Station}) {
    console.log('new target route')
  }

  public setSelectedStation(s: number) {
    if (this.selectedStations && this.selectedStations.has(s)) {
      return false
    }
    if (!this.selectedStations || this.selectedStations.size >= 2) {
      this.selectedStations = new Set<number>()
    }
    this.selectedStations.add(s)
    return true
  }

  public optimizingParameters () {
    if (this.selectedStations && this.selectedStations.size > 1) {
      const stops = Array.from(this.selectedStations)
      return { origin: stops[0], dest: stops[1], stops: [] }
    } else if (this.route.stations.length >= 2) {
      return { origin: this.route.stations[0], dest: this.route.stations[this.route.stations.length - 1], stops: [] }
    } else {
      return null
    }
  }

  public targetStops() {
    if (!this.selectedStations) {
      return null
    }
    const stops = Array.from(this.selectedStations)
    if (stops.length >= 2) {
      let flag = false
      this.route.stations.forEach(s => {
        // @ts-ignore
        if (this.selectedStations.has(s)) {
          flag = !flag
        } else if (flag) {
          stops.push(s)
        }
      })
    }
    return new Set(stops)
  }

  public stopsGeoJSON() {
    console.log(this.selectedStations)
    if (!this.selectedStations) {
      return null
    }
    const features : GeoJSON.Feature<GeoJSON.Point>[] = []
    const stops = Array.from(this.selectedStations)
    if (stops.length >= 2) {
      let flag = false
      this.route.stations.forEach(s => {
        // @ts-ignore
        if (this.selectedStations.has(s)) {
          flag = !flag
        } else if (flag) {
          stops.push(s)
        }
      })
    }
    stops.forEach(s => {
      features.push({
        id: this.indexedStations[s].id,
        type: 'Feature',
        geometry: {
          type: 'Point',
          coordinates: this.indexedStations[s].coordinates
        },
        properties: {}
      })
    })
    return features
  }
}
