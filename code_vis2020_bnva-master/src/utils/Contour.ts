import _ from 'lodash'
import {Station} from "@/store/modules/Exploration";
import {GeoJSON} from "geojson";
import * as d3 from "d3";

export class Contour {
  public indexedStationPolygons: { [id: number]: [number, number][] } = {}
  public edgeStationsInClusters: { [id: number]: Set<number> } = {}
  public filteredEdgeLines: [number, number][][] = []
  public clusters: { [id: number]: GeoJSON.FeatureCollection<GeoJSON.Polygon> } = {}

  constructor(public voronoiPolygons: { coordinates: [number, number][], station: Station }[],
              public voronoiLinks: { source: Station, target: Station }[],
              public stationMapToCluster: { [sid: number]: { [level: number]: number } },
              public level: number,
              public clusterLevels: { [level: number]: { [clusterID: number]: number[] } }) {
    _.each(clusterLevels[level], (s, cid) => {
      this.edgeStationsInClusters[parseInt(cid)] = new Set<number>()
    })
    _.each(voronoiPolygons, p => {
      if (p.coordinates) {
        this.indexedStationPolygons[p.station.id] = p.coordinates
      }
    })
    this.findEdgeStations()
    this.filteredEdgeLines = this.filterInnerPoints()
    this.computeVoronoiPolygonClusters()
  }

  public get polygonPointsGeoJSON() {
    const points: GeoJSON.FeatureCollection<GeoJSON.Point> = {
      type: 'FeatureCollection',
      features: []
    }
    const features: GeoJSON.Feature<GeoJSON.Point>[] = []
    _.each(this.voronoiPolygons, p => {
      p.coordinates.forEach(c => {
        features.push({
          id: 1,
          type: 'Feature',
          geometry: {
            type: 'Point',
            coordinates: c
          },
          properties: []
        })
      })
    })
    points.features = features
    return points
  }

  public get contourGeoJSON() {
    const contour: GeoJSON.FeatureCollection<GeoJSON.LineString> = {
      type: 'FeatureCollection',
      features: []
    }
    if (this.filteredEdgeLines.length > 0) {
      let i = 0
      // @ts-ignore
      contour.features = _.map(this.filteredEdgeLines, line => {
        i++
        return {
          id: i,
          type: 'Feature',
          geometry: {
            type: 'LineString',
            coordinates: line
          },
          properties: []
        }
      })
    }
    return contour
  }

  findEdgeStations() {
    _.each(this.voronoiLinks, link => {
      const source: Station = link.source
      const target: Station = link.target
      const sourceClusterID = this.stationMapToCluster[source.id][this.level]
      const targetClusterID = this.stationMapToCluster[target.id][this.level]
      if (sourceClusterID !== targetClusterID) {
        this.edgeStationsInClusters[sourceClusterID].add(source.id)
        this.edgeStationsInClusters[targetClusterID].add(target.id)
      }
    })
  }

  private _addLineEdge(
    edgeInCluster: { [source: number]: { [target: number]: number } },
    source: number, target: number) {
    if (edgeInCluster[source] && edgeInCluster[source][target]) {
      edgeInCluster[source][target]++
      edgeInCluster[target][source]++
      return
    }
    if (!(source in edgeInCluster)) {
      edgeInCluster[source] = {}
    }
    if (!(target in edgeInCluster[source])) {
      edgeInCluster[source][target] = 1
    }
    if (!(target in edgeInCluster)) {
      edgeInCluster[target] = {}
    }
    if (!(source in edgeInCluster[target])) {
      edgeInCluster[target][source] = 1
    }
  }

  filterInnerPoints() {
    // the lon and the lat in lonLatClusterDict are enlarged
    const lonLatClusterDict:
      {
        [lon: number]: {
          [lat: number]: {
            set: Set<number>
            id: number
          }
        }
      } = {}
    // edges in the same cluster
    const edgeInClusters: { [id: number]: { [source: number]: { [target: number]: number } } } = {}
    // build coordinates to cluster idx map
    let positionIdx = 0
    let lastIdx = -1
    const filteredLines: [number, number][][] = []
    _.each(this.clusterLevels[this.level], (stations, cid) => {
      edgeInClusters[parseInt(cid)] = {}
      stations.forEach((s) => {
        const polygon = this.indexedStationPolygons[s]
        _.each(polygon, coordinate => {
          const lon = Math.round(coordinate[0] * 10000)
          const lat = Math.round(coordinate[1] * 10000)
          if (!lonLatClusterDict[lon] || !lonLatClusterDict[lon][lat]) {
            if (!(lon in lonLatClusterDict)) {
              lonLatClusterDict[lon] = {}
            }
            if (!(lat in lonLatClusterDict[lon])) {
              lonLatClusterDict[lon][lat] = {set: new Set<number>(), id: positionIdx}
            }
            positionIdx++
          }
          lonLatClusterDict[lon][lat].set.add(parseInt(cid))
          if (lastIdx >= 0) {
            this._addLineEdge(edgeInClusters[parseInt(cid)], lastIdx, lonLatClusterDict[lon][lat].id)
          }
          lastIdx = lonLatClusterDict[lon][lat].id
        })
        lastIdx = -1
      })
    })
    _.each(this.edgeStationsInClusters, (stations, cid) => {
      stations.forEach((s) => {
        let line: [number, number][] = []
        const polygon = this.indexedStationPolygons[s]
        lastIdx = -1
        _.each(polygon, coordinate => {
          const lon = Math.round(coordinate[0] * 10000)
          const lat = Math.round(coordinate[1] * 10000)
          if (lonLatClusterDict[lon][lat].set.size > 1) {
            if (lastIdx >= 0 && edgeInClusters[parseInt(cid)][lonLatClusterDict[lon][lat].id][lastIdx] > 1) {
              // split line
              if (line.length > 1) {
                filteredLines.push(line)
              }
              line = []
            }
            line.push(coordinate)
            lastIdx = lonLatClusterDict[lon][lat].id
          } else {
            filteredLines.push(line)
            line = []
          }
        })
        filteredLines.push(line)
      })
    })
    return filteredLines
  }

  computeVoronoiPolygonClusters() {
    if (this.clusterLevels) {
      // @ts-ignore
      const max = _.maxBy(_.values(this.clusterLevels[this.level]), f => f.length).length
      // @ts-ignore
      const min = _.minBy(_.values(this.clusterLevels[this.level]), f => f.length).length
      const sizeScale = d3.scaleLinear().domain([min, max]).range([0.01, 0.5])
      _.each(this.clusterLevels[this.level], (c, k) => {
        // @ts-ignore
        this.clusters[parseInt(k)] = this.clusterPolygons(parseInt(k), sizeScale)
      })
    }
  }

  clusterPolygons(clusterID: number, sizeScale: any) {
    if (this.clusterLevels) {
      const clusterPolygons = _.filter(
        this.clusterLevels[this.level][clusterID],
        c => this.indexedStationPolygons[c])
      const features = _.map(clusterPolygons, c => {
        // @ts-ignore
        return {
          id: c,
          type: 'Feature',
          geometry: {
            type: 'Polygon',
            coordinates: [this.indexedStationPolygons[c]]
          },
          properties: {
            id: c,
            value: sizeScale(this.clusterLevels[this.level][clusterID].length)
          }
        }
      })
      return {
        id: clusterID.toString(),
        type: 'FeatureCollection',
        features: features
      }
    }
  }
}
