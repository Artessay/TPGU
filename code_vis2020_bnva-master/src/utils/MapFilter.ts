import _ from 'lodash'
import { Candidate} from "@/store/modules/CandidatesList";
import {ClusterLink, StationClusters} from "@/store/modules/Projection";
import {GeoJSON} from "geojson";
import * as d3 from "d3";

export class MapFilter {
  public selectedClusters: Set<number> = new Set()
  public selectedLinks: Set<number> = new Set()
  public clusterRoutes: { [cid: number]: Set<number> } = {}
  public indexedLinks: { [rid: number]: { c1: number, c2: number, routes: Set<number> } } = {}
  public links: { c1: number, c2: number, routes: Set<number> }[] = []

  public get mapLinks () {
    const mapLinksGeoJSON : GeoJSON.FeatureCollection<GeoJSON.LineString> = {
      type: 'FeatureCollection',
      features: []
    }
    const features: GeoJSON.Feature<GeoJSON.LineString>[] = []
    // @ts-ignore
    const max = _.maxBy(this.links, f => f.routes.size).routes.size
    // @ts-ignore
    const min = _.minBy(this.links, f => f.routes.size).routes.size
    const sizeScale = d3.scaleLinear().domain([min, max]).range([5, 15])
    _.each(this.links, (link, id) => {
      if (link.routes.size > 7) {
        const s1 = this.mapCenter[link.c1]
        const s2 = this.mapCenter[link.c2]

        const feature = {
          id: id,
          type: 'Feature',
          geometry: {
            type: 'LineString',
            coordinates: this.chunkLink(s1, s2, 0.07)
          },
          properties: {
            cluster1: s1,
            cluster2: s2,
            key: id,
            value: sizeScale(link.routes.size),
            routes: Array.from(link.routes)
          }
        }
        // @ts-ignore
        features.push(feature)
      }
    })
    mapLinksGeoJSON.features = features
    return mapLinksGeoJSON
  }

  chunkLink(c1: [number, number], c2: [number, number], r: number) {
    const retC11 = c1[0] + (c2[0] - c1[0]) * r
    const retC12 = c1[1] + (c2[1] - c1[1]) * r
    const retC21 = c2[0] + (c1[0] - c2[0]) * r
    const retC22 = c2[1] + (c1[1] - c2[1]) * r
    return [[retC11, retC12], [retC21, retC22]]
  }

  constructor (
    public stationClusters: StationClusters,
    public level: number,
    public mapCenter: {[id: number]: [number, number]}) {
    this.computeClusterRoutes(stationClusters.cluster_links[level])
    this.computeLinks()
  }

  computeClusterRoutes (links: ClusterLink) {
    _.each(links, (l, id) => { this.clusterRoutes[parseInt(id)] = new Set<number>() })
    _.each(links, (link, cid1) => {
      _.each(link, (routes, cid2) => {
        _.each(routes, r => {
          this.clusterRoutes[parseInt(cid1)].add(r)
          this.clusterRoutes[parseInt(cid2)].add(r)
        })
      })
    })
  }

  computeLinks () {
    let idx = 0
    _.each(this.stationClusters.cluster_links[this.level], (cluster, ci) => {
      _.each(cluster, (link, cj) => {
        if (link.length > 0) {
          const l = {
            c1: parseInt(ci),
            c2: parseInt(cj),
            routes: new Set<number>(link)
          }
          this.links.push(l)
          this.indexedLinks[idx] = l
          idx++
        }
      })
    })
  }

  public toggleCluster(cid: number) {
    if (this.selectedClusters.has(cid)) {
      this.selectedClusters.delete(cid)
      return false
    }
    this.selectedClusters.add(cid)
    return true
  }

  public toggleLink(lid: number) {
    if (this.selectedLinks.has(lid)) {
      this.selectedLinks.delete(lid)
      return false
    }
    this.selectedLinks.add(lid)
    return true
  }

  getRoutesByStationCluster(routesSet: Set<number>) {
    this.selectedClusters.forEach(c => {
      this.clusterRoutes[c].forEach(r => {
        routesSet.add(r)
      })
    })
    return routesSet
  }

  getRoutesByLink(routesSet: Set<number>) {
    this.selectedLinks.forEach(l => {
      this.links[l].routes.forEach(r => {
        routesSet.add(r)
      })
    })
    return routesSet
  }

  filterRoutesByClusters(routesSet: Set<number>) {
    let retSet = new Set<number>()
    let values = new Set<number>()
    let first = true
    this.selectedClusters.forEach(c => {
      if (first) {
        values = this.clusterRoutes[c]
        first = false
      } else {
        this.clusterRoutes[c].forEach(r => {
          if (values.has(r)) {
            retSet.add(r)
          }
        })
        values = retSet
        retSet = new Set<number>()
      }
    })
    routesSet.forEach(r => {
      if (values.has(r)) {
        retSet.add(r)
      }
    })
    return retSet
  }

  public get candidates() {
    const routesSet = new Set<number>()
    if (this.selectedLinks.size === 0 && this.selectedClusters.size === 0) {
      return routesSet
    }
    if (this.selectedLinks.size > 0 && this.selectedClusters.size === 0) {
      return this.getRoutesByLink(routesSet)
    }
    if (this.selectedClusters.size > 0 && this.selectedLinks.size === 0) {
      return this.filterRoutesByClusters(this.getRoutesByStationCluster(routesSet))
    }
    return this.filterRoutesByClusters(this.getRoutesByLink(routesSet))
  }
}
