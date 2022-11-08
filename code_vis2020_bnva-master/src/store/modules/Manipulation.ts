import {Action, getModule, Module, Mutation, VuexModule} from "vuex-module-decorators";
import store from "@/store";
import {ADD_STOP_RUNTIME, BUILD_GRAPH, CONTINUE_MCTS, GET_CURRENT_ROUTES, PAUSE_MCTS, STOP_MCTS} from "@/query";
import {MonterCarloTree, RouteList, SRoute, StationGraph, StationNode} from "@/utils/MCTSTypes";
import _ from 'lodash'
import {gqlQuery} from "@/utils/GraphqlAPI";
import {GeoJSON} from "geojson";
import CandidatesList, {Candidate} from "@/store/modules/CandidatesList";
import Evaluation from "@/store/modules/Evaluation";
import Exploration from "@/store/modules/Exploration";
import {ConstructionCost, ServiceCost} from "@/utils/Criterion";
import {TargetRoute} from "@/utils/targetRoute";

const SNAPSHOTS_NUM = 10

@Module({name: 'Manipulation.ts', dynamic: true, namespaced: true, store})
class Manipulation extends VuexModule {
  private _start = false
  private _optimizingRouteID = -1
  private _origin = 6842 // mock
  private _destination = 5664 // mock
  private _stops : Set<number> = new Set<number>()
  private _stationGraph: StationGraph | null = null
  private _routeList: RouteList | null = null
  private _sessionID = ''
  private _running = false
  private _receiveDataCount = 0
  private _routeListSnapshots: GeoJSON.FeatureCollection<GeoJSON.LineString>[] = []
  private _currentSnapshots = -1
  private _checkSnapshot = false
  private _filterSet : Set<number> | null = null
  private _stationOrderMap: {[id: number]: number} = {}
  private _serviceCostModel: ServiceCost = new ServiceCost()
  private _constructConstModel: ConstructionCost = new ConstructionCost()

  public get routeList () {
    return this._routeList
  }

  public get stationOrderMap () {
    return this._stationOrderMap
  }

  public get currentSnapshotNum () {
    return this._currentSnapshots
  }

  public get checkSnapshot () {
    return this._checkSnapshot
  }

  public get start () {
    return this._start
  }

  public get running() {
    return this._running
  }

  public get receiveDataCount() {
    return this._receiveDataCount
  }

  public get currentSnapShot() {
    if (this._routeListSnapshots.length > 0 && this._currentSnapshots >= 0 && this._currentSnapshots < this._routeListSnapshots.length) {
      return this._routeListSnapshots[this._currentSnapshots]
    }
  }

  public get routesList () {
    return this._routeList
  }

  public get routesListGeoJSON () {
    const features: GeoJSON.Feature<GeoJSON.LineString>[] = []
    if (this._routeList) {
      _.each(this._routeList.routes, r => {
        if ((this._filterSet && this._filterSet.has(r.id)) || !this._filterSet) {
          features.push({
            id: r.id,
            type: 'Feature',
            geometry: {
              type: 'LineString',
              coordinates: r.coordinates
            },
            properties: {}
          })
        }
        // const coordinates = _.map(r.r, s => {
        //   return [s.lon, s.lat]
        // })
      })
      const listGeoJSON : GeoJSON.FeatureCollection<GeoJSON.LineString> = {
        type: 'FeatureCollection',
        features: features
      }
      return listGeoJSON
    }
  }

  // public get stationGraphGeoJSON() {
  //   return this._stationGraphGeoJSON
  // }

  public get stationGraphGeoJSON () {
    const features: GeoJSON.Feature<GeoJSON.Point>[] = []
    const linkFeatures: GeoJSON.Feature<GeoJSON.LineString>[] = []
    if (this._stationGraph) {
      const indexedNodes : {[id: number]: StationNode } = {}
      _.each(this._stationGraph.nodes, n => {
        let isOD = false
        // @ts-ignore
        if (this._stationGraph.originNode.s.id === n.s.id || this._stationGraph.destNode.s.id === n.s.id) {
          isOD = true
        }
        indexedNodes[n.s.id] = n
        if (n.next.length > 0 || n.prev.length > 0) {
          features.push({
            id: n.s.id,
            type: 'Feature',
            geometry: {
              type: 'Point',
              coordinates: [n.s.lon, n.s.lat]
            },
            properties: {
              isOD: isOD
            }
          })
        }
      })
      const graphNodesGeoJSON: GeoJSON.FeatureCollection<GeoJSON.Point> = {
        type: 'FeatureCollection',
        features: features
      }
      // let id = 0
      // _.each(this._stationGraph.nodes, n => {
      //   const nc = [n.s.lon, n.s.lat]
      //   _.each(n.next, next => {
      //     const nextc = [indexedNodes[next.s.id].s.lon, indexedNodes[next.s.id].s.lat]
      //     linkFeatures.push({
      //       id: id,
      //       type: 'Feature',
      //       geometry: {
      //         type: 'LineString',
      //         coordinates: [nc, nextc]
      //       },
      //       properties: {}
      //     })
      //     id++
      //   })
      // })
      const graphLinksGeoJSON: GeoJSON.FeatureCollection<GeoJSON.LineString> = {
        type: 'FeatureCollection',
        features: linkFeatures
      }
      return { nodes: graphNodesGeoJSON, links: graphLinksGeoJSON}
    }
  }

  @Action({commit: '_createStationGraph'})
  public createStationGraph(route: TargetRoute | null) {
    const params = route?.optimizingParameters()
    if (params) {
      return params
    }
  }

  @Mutation
  public applyFiltering() {
    if (this._start) {
      this._filterSet = new Set<number>()
      CandidatesList.filterCandidates.forEach(c => {
        // @ts-ignore
        this._filterSet.add(c.routeID)
      })
      this._receiveDataCount++
    }
  }

  @Mutation
  public setOptimizingRouteID(id: number) {
    this._optimizingRouteID = id
  }

  @Mutation
  private async _createStationGraph(input: {origin: number, dest: number, stops: number[]}) {
    this._start = true
    const {origin, dest, stops} = input
    this._origin = origin
    this._destination = dest
    _.each(stops, s => { this._stops.add(s) })
    const ss = Array.from(this._stops)
    const {data: {createStationGraph: {username, graph, randomSeed}}} =
      await gqlQuery<{ data: {createStationGraph: { username: string, graph: StationGraph, randomSeed: number } } }>(
        'BuildGraph',
        BUILD_GRAPH,
        {origin, dest, stops: ss, seed: 5}
      )
    console.log('Current random seed: ', randomSeed)
    if (graph) {
      console.log("the number of station graph nodes: " + graph.nodes.length)
      graph.nodes.forEach(node => {
        this._stationOrderMap[node.s.id] = node.order
      })
    }
    this._sessionID = username
    this._stationGraph = graph
    this._running = true
  }

  @Mutation
  public async queryCurrentRouteLists() {
    const data : {data: {plannings: RouteList}} = await gqlQuery<{data: {plannings: RouteList}}>(
      'getCurrentPlanning',
      GET_CURRENT_ROUTES,
      {username: this._sessionID})
    if (data === null) {
      this._running = false
      return
    }
    const list = data.data.plannings
    list.indexedRoutes = {}
    let idx = 0
    _.each(list.routes, route => {
      route.weight = _.map(route.criteria, c => { return 1 })
      route.id = idx
      route.serviceCost = this._serviceCostModel.value(route.criteria[0] / 3600)
      route.constructCost = this._constructConstModel.value(route.r.length)
      list.indexedRoutes[idx] = route
      idx++
    })
    this._routeList = list
    this._receiveDataCount++
  }

  @Mutation
  public async pausePlanning() {
    this._running = false
    await gqlQuery<{data}>('pause', PAUSE_MCTS, {username: this._sessionID})
  }

  @Mutation
  public async continuePlanning() {
    await gqlQuery<{data}>('continue', CONTINUE_MCTS, {username: this._sessionID})
    this._running = true
  }

  @Mutation
  public async stopPlanning() {
    await gqlQuery<{data}>('stopMCTS', STOP_MCTS, {username: this._sessionID})
  }

  @Mutation
  public async addStopInRunTime (sid: number) {
    await gqlQuery('addStop', ADD_STOP_RUNTIME, {username: this._sessionID, stop: sid})
    const {data} = await gqlQuery<{data: {plannings: RouteList}}>(
      'getCurrentPlanning',
      GET_CURRENT_ROUTES,
      {username: this._sessionID})
    const {data: {plannings: list}} = await gqlQuery<{data: {plannings: RouteList}}>(
      'getCurrentPlanning',
      GET_CURRENT_ROUTES,
      {username: this._sessionID})
    this._routeList = list
    this._receiveDataCount++
    this._checkSnapshot = false
  }

  @Mutation
  public saveCurrentPlaning (lists : GeoJSON.FeatureCollection<GeoJSON.LineString>) {
    if (this._routeListSnapshots.length >= SNAPSHOTS_NUM) {
      this._routeListSnapshots.shift()
    }
    if (lists) {
      this._routeListSnapshots.push(lists)
    }
    this._currentSnapshots = this._routeListSnapshots.length - 1
  }

  @Mutation
  public getLastSnapshot () {
    if (this._currentSnapshots > 0) {
      this._checkSnapshot = true
      this._currentSnapshots--
    }
  }

  @Mutation
  public getNextSnapshot () {
    if (this._currentSnapshots < this._routeListSnapshots.length) {
      this._currentSnapshots++
    }
    if (this._currentSnapshots >= this._routeListSnapshots.length - 1) {
      this._checkSnapshot = false
    }
  }

  @Mutation
  public investigateConflicts () {
    if (this._routeList) {
      console.log('the number of routes', this._routeList.routes.length)
      const routes: SRoute[] = []
      const routeSet = new Set<number>()
      console.log('the number of routes in filterCandidates ', CandidatesList.filterCandidates.length)
      _.each(CandidatesList.filterCandidates, c => {
        routeSet.add(c.routeID)
      })
      console.log('the number of routes in filter set ', routeSet.size)
      _.each(this._routeList.routes, r => {
        if (routeSet.has(r.id)) {
          routes.push(r)
        }
      })
      // console.log(CandidatesList.filterCandidates)
      const stationSet = new Set<number>()
      _.each(routes, a => {
        _.each(a.r, s => {
          stationSet.add(s.id)
        })
      })
      const sortedStationIDs = _.sortBy(Array.from(stationSet), sid => {
        return this._stationOrderMap[sid]
      })

      Evaluation.createRouteGroupCollection({
        routes: routes,
        orderedStations: sortedStationIDs
      })
    }
  }
}

export default getModule(Manipulation)
