import {Action, getModule, Module, Mutation, VuexModule} from "vuex-module-decorators"
import store from "@/store"
import Exploration, {Station} from "@/store/modules/Exploration";
import _ from 'lodash'
import * as d3 from 'd3'
import {GeoJSON} from "geojson";
import {Contour} from "@/utils/Contour";
import {MapFilter} from "@/utils/MapFilter";
import CandidatesList from "@/store/modules/CandidatesList";
import {ServiceCost} from "@/utils/Criterion";
import Dataset from "@/store/modules/Dataset";

export interface StationClusters {
  cluster_center: {
    [level: number]: {
      [clusterID: number]: number[]
    }
  }
  level_connection: {
    [level: number]: {
      [clusterID: number]: number[]
    }
  }
  cluster_levels: {
    [level: number]: {
      [clusterID: number]: number[]
    }
  }
  cluster_links: {
    [level: number]: ClusterLink
  }
  cluster_vol: {
    [level: number]: {
      [clusterID: number]: { in: number[], out: number[] }
    }
  }
  clusterAttr: {
    [level: number]: {
      [clusterID: number]: number[]
    }
  }
}

export interface ClusterLink {
  [clusterID: number]: {
    [clusterID: number]: number[]
  }
}

@Module({ name: 'ProjectionStore', dynamic: true, namespaced: true, store })
class Projection extends VuexModule {
  private _level = 5
  private _clusters : StationClusters | null = null
  private _mapFilter: MapFilter | null = null
  private _mapCenters: {[id: number]: [number, number]} = {}
  private _voronoiPolygons : {[id: number]: GeoJSON.FeatureCollection<GeoJSON.Polygon>} = {}
  private _inversedStationClusterIndex: {[sid: number]: {[level: number]: number} } = {}
  private _contour: Contour | null = null
  private _selectedClusterCenters: Set<number> = new Set<number>()
  private _mapCenterGeoJSON : GeoJSON.FeatureCollection<GeoJSON.Point> = {
    type: 'FeatureCollection',
    features: []
  }

  public get contourGeoJSON () {
    if (this._contour) {
      return this._contour.contourGeoJSON
    }
  }

  public get indexedPolygons () {
    if (this._contour) {
      return this._contour.indexedStationPolygons
    }
  }

  public get mapLinks () {
    if (this._mapFilter) {
      return this._mapFilter.mapLinks
    }
  }

  public get mapCenters () {
    return this._mapCenterGeoJSON
  }

  public get centers () {
    if (this._clusters) {
      return this._clusters.cluster_center[this._level]
    }
  }

  public get links () {
    if (this._clusters) {
      return this._clusters.cluster_links[this._level]
    }
  }

  public get clusters () {
    if (this._clusters) {
      return this._clusters.cluster_levels[this._level]
    }
  }

  public get clusterVol () {
    if (this._clusters) {
      return this._clusters.cluster_vol[this._level]
    }
  }

  public get clusterAttrs () {
    if (this._clusters) {
      return this._clusters.clusterAttr[this._level]
    }
  }

  private get stations () {
    return _.values(Exploration.indexedStations)
  }

  public get voronoiPolygons () {
    return this._voronoiPolygons
  }

  public get selectedLinks () {
    if (this._mapFilter) {
      return this._mapFilter.selectedLinks
    }
  }

  public get selectedCenters () {
    if (this._mapFilter) {
      return this._mapFilter.selectedClusters
    }
  }

  @Action({ commit: '_setClusters' })
  public setClusters (clusters: StationClusters) {
    return clusters
  }

  @Action({ commit: '_setCenters'})
  public setCenters (centers: { id: string, px: number, py: number }[]) {
    return centers
  }

  @Mutation
  private _setClusters(clusters: StationClusters) {
    const serviceCost = new ServiceCost()
    _.each(clusters.cluster_levels, (cs, l) => {
      _.each(cs, (c, clusterID) => {
        _.each(c, s => {
          if (!this._inversedStationClusterIndex[s]) {
            this._inversedStationClusterIndex[s] = {}
          }
          this._inversedStationClusterIndex[s][parseInt(l)] = parseInt(clusterID)
        })
      })
    })
    clusters.clusterAttr = {}
    // compute cluster attributes
    _.each(clusters.cluster_links, (cs, l) => {
      clusters.clusterAttr[l] = {}
      _.each(cs, (c, clusterID) => {
        const rSet = new Set<number>()
        _.each(c, rs => {
          rs.forEach(r => { rSet.add(r) })
        })
        _.each(cs, (otherC, otherID) => {
          _.each(otherC, (rs, cid) => {
            if (cid === clusterID) {
              rs.forEach(r => { rSet.add(r) })
            }
          })
        })
        const routes = Array.from(rSet)
        const dist = _.meanBy(routes, (r) => { return Exploration.indexedRoutes[r].length })
        const flow = _.meanBy(routes, (r) => { return Exploration.indexedRoutes[r].checkin + Exploration.indexedRoutes[r].checkout })
        const load = _.meanBy(routes, (r) => { return Math.log(Exploration.indexedRoutes[r].load + 1) })
        const directness = _.meanBy(routes, (r) => { return Exploration.indexedRoutes[r].directness })
        const stationNum = _.meanBy(routes, (r) => { return Exploration.indexedRoutes[r].stations.length })
        const sCost = _.meanBy(routes, (r) => { return serviceCost.value(Exploration.indexedRoutes[r].time) })
        clusters.clusterAttr[l][clusterID] = [dist, stationNum, flow, load, directness, sCost]
      })
    })
    _.each(clusters.clusterAttr, (cs, l) => {
      // HARD CODE
      for (let i = 0; i < 6; i++) {
        const max = _.maxBy(_.values(clusters.clusterAttr[l]), attrs => { return attrs[i] })[i]
        const min = _.minBy(_.values(clusters.clusterAttr[l]), attrs => { return attrs[i] })[i]
        _.each(clusters.clusterAttr[l], (attr, j) => {
          if (max === min) {
            attr[i] = 1
          } else if (i === 4 || i === 5) {
            attr[i] = (max - attr[i]) / (max - min)
          } else {
            attr[i] = (attr[i] - min) / (max - min)
          }
        })
      }
    })
    this._clusters = clusters
  }

  @Action({ commit: '_computeClusterGeoCenters'})
  public computeClusterGeoCenters (clusters: StationClusters) {
    return clusters
  }

  @Mutation
  private _computeClusterGeoCenters(clusters: StationClusters) {
    const computeGeoCenter = function (clusterID: number, clusters: StationClusters, level: number) {
      const center: [number, number] = [0, 0]
      if (clusters) {
        const len = clusters.cluster_levels[level][clusterID].length
        _.each(clusters.cluster_levels[level][clusterID], c => {
          const s = Exploration.indexedStations[c]
          center[0] += s.coordinates[0]
          center[1] += s.coordinates[1]
        })
        center[0] /= len
        center[1] /= len
      }
      return center
    }

    if (clusters) {
      _.each(clusters.cluster_center[this._level], (c, k) => {
        this._mapCenters[parseInt(k)] = computeGeoCenter(parseInt(k), clusters, this._level)
      })
    }
    this._mapFilter = new MapFilter(clusters, this._level, this._mapCenters)
    CandidatesList.addCandidates({
      indexedRoutes: Exploration.indexedRoutes,
      newCandidates: _.map(Exploration.indexedRoutes, r => { return r.id })
    })
  }

  @Mutation
  private _setCenters(centers: { id: string, px: number, py: number }[]) {
    if (this._clusters) {
      _.each(
        centers,
        center => {
          // @ts-ignore
          this._clusters.cluster_center[this._level][center.id] = [center.px, center.py]
        }
      )
    }
  }

  @Mutation
  public computeVoronoiForStations({stations, extent}: {stations: Station[], extent: number[][]}) {
    if (!this._clusters) {
      return
    }
    const voronoiExtent : [[number, number], [number, number]] = [[0, 0], [0, 0]]
    voronoiExtent[0][0] = extent[0][0] - 1e-2
    voronoiExtent[0][1] = extent[0][1] - 1e-2
    voronoiExtent[1][0] = extent[1][0] + 1e-2
    voronoiExtent[1][1] = extent[1][1] + 1e-2
    // @ts-ignore
    const voronoi = d3.voronoi().extent(voronoiExtent)
      .x((d: Station) => d.coordinates[0])
      .y((d: Station) => d.coordinates[1])
    const links = voronoi.links(stations)
    const polygons = voronoi.polygons(stations)
    // calculate the contour
    this._contour = new Contour(
      _.filter(polygons, p => { return p })
        .map(p => { return {coordinates: p, station: p.data} } ),
      links,
      this._inversedStationClusterIndex,
      this._level,
      this._clusters.cluster_levels)
    this._voronoiPolygons = this._contour.clusters
  }

  @Mutation
  public setSelectedClusterCenters(centers: number[]) {
    centers.forEach(c => {
      this._selectedClusterCenters.add(c)
    })
  }

  @Mutation
  public setCenterGeoJSON () {
    const features: GeoJSON.Feature<GeoJSON.Point>[] = []
    if (this._clusters) {
      const cluster = this._clusters
      // @ts-ignore
      const max = _.maxBy(_.values(cluster.cluster_levels[this._level]), f => f.length).length
      // @ts-ignore
      const min = _.minBy(_.values(cluster.cluster_levels[this._level]), f => f.length).length
      const sizeScale = d3.scaleLinear().domain([min, max]).range([5, 15])
      _.each(this._mapCenters, (c, id) => {
        let highlight = false
        if (this._selectedClusterCenters.has(parseInt(id))) {
          console.log('set highlight')
          highlight = true
        }
        const feature = {
          id: id,
          type: 'Feature',
          geometry: {
            type: 'Point',
            coordinates: c
          },
          properties: {
            key: id,
            highlight: highlight,
            value: sizeScale(cluster.cluster_levels[this._level][parseInt(id)].length)
          }
        }
        // @ts-ignore
        features.push(feature)
      })
    }
    this._mapCenterGeoJSON.features = features
  }

  @Action({ commit: '_toggleLink' })
  public toggleLink(lid: number) {
    return lid
  }

  @Mutation
  private _toggleLink(lid: number) {
    if (this._mapFilter) {
      if (this._mapFilter.toggleLink(lid)) {
        // the link is selected
        console.log('the link ', lid, ' is selected')
      } else {
        // the link is unselected
        console.log('the link ', lid, ' is unselected')
      }
      CandidatesList.clearCandidates()
      CandidatesList.addCandidates({
        indexedRoutes: Exploration.indexedRoutes,
        newCandidates: Array.from(this._mapFilter.candidates)
      })
      Exploration.setHighlightRoutesGeoJSON({
        routes: Array.from(this._mapFilter.candidates),
        selected: true
      })
    }
  }

  @Mutation
  public toggleCluster(cid: number) {
    if (this._mapFilter) {
      this._mapFilter.toggleCluster(cid)
      CandidatesList.clearCandidates()
      CandidatesList.addCandidates({
        indexedRoutes: Exploration.indexedRoutes,
        newCandidates: Array.from(this._mapFilter.candidates)
      })
      Exploration.setHighlightRoutesGeoJSON({
        routes: Array.from(this._mapFilter.candidates),
        selected: true
      })
    }
  }
}

export default getModule(Projection)
