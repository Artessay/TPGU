import { Action, Module, Mutation, VuexModule, getModule } from 'vuex-module-decorators'
import _ from 'lodash'
import CandidatesList from "@/store/modules/CandidatesList";
import Vue from 'vue'
import store from '@/store'
import {GeoJSON} from "geojson";
import Projection from "@/store/modules/Projection";
import Evaluation from "@/store/modules/Evaluation";
import {TargetRoute} from "@/utils/targetRoute";
import {RouteTransfers, RouteTransfersToRouteBranches, BarchartInMatrix} from "@/utils/Types";

const TRANSFER_JSON_URL = 'data/beijing/new_check_sequence.json'

export interface Station {
  id: number
  name: string
  coordinates: [number, number]
}

export interface Route {
  id: number
  name: string
  stations: number[]
  length: number
  time: number
  load: number
  flow: {
    checkin: number
    checkout: number
  }[]
  directness: number
  states: {
    selected: boolean
  }
  matrix: {
    [origin: number]: {
      [dest: number]: number
    }
  }
  projection: string[]
  center: number[]
  checkin: number
  checkout: number
  coordinates: number[][]
}

export interface TransferData {
  transfer_records: Transfer[]
  route_branches: RouteBranches[]
}

export interface Transfer {
  trip_id: number // the trip id
  from_route_id: number
  from_station_id: number
  from_stop_num: number
  to_route_id: number
  to_station_id: number
  to_stop_num: number
}

export interface Branch {
  route_id: number
  from_stop_num: number
  to_stop_num: number
  from_station_id: number
  to_station_id: number
  records: number[]
}

export interface RouteBranches {
  route_id: number
  branches: Branch[]
  checkinBranches: Branch[]
  checkoutBranches: Branch[]
}

export interface CheckSequence {
  day: {
    checkin: {
      [routeId: number]: DayDataFlow[]
    }
    checkout: {
      [routeId: number]: DayDataFlow[]
    }
  }
  hour: {
    checkin: {
      [routeId: number]: HourDataFlow[]
    }
    checkout: {
      [routeId: number]: HourDataFlow[]
    }
  }
  weekday: {
    checkin: {
      [routeId: number]: WeekdayDataFlow[]
    }
    checkout: {
      [routeId: number]: WeekdayDataFlow[]
    }
  }
}

export interface SingleRouteCheckSequence {
  checkin: SingleCheckSequence
  checkout: SingleCheckSequence
}

export interface SingleCheckSequence {
  day: DayDataFlow[]
  hour: HourDataFlow[]
  weekday: WeekdayDataFlow[]
}

export interface DayDataFlow {
  station: number
  day: string
  times: number
}

export interface HourDataFlow {
  station: number
  hour: string
  times: number
}

export interface WeekdayDataFlow {
  station: number
  weekday: string
  times: number
}

export interface CheckSequenceEachRoute {
  [station: number]: {
    day: DayDataFlow[]
    hour: HourDataFlow[]
    weekday: WeekdayDataFlow[]
  }
}

export interface MatrixG {
  mainG: any
  inG: any
  outG: any
  contentG: any
  rectsG: any
  inRoute: any
  outRoute: any
  flowG: any
  stationG: any
  lineG: any
  selectedG: any
}
export interface Selected {
  xKey: number
  yKey: number
  flag: boolean
}
export interface MatrixWave {
  x: number
  y: number
  box: any
  reverse: boolean
  parId: number
  routeId: number
  direction: number // 上右下左 1234
  stations: number[]
  matrixG: MatrixG // the only svg element
  stationTextsObj: any
  selected: Selected
  horizonChart: {
    data: {checkin: CheckSequenceEachRoute, checkout: CheckSequenceEachRoute}
    current: string // day hour weekday
  }
  barChart: BarchartInMatrix
  horizonSelected: {
    in: Selected
    out: Selected
  }
  sibling: {
    left: MatrixBranch
    right: MatrixBranch
  }
}
export interface MatrixBranch {
  id: number
  branch: Branch | null
}
export interface Position {
  x: number
  y: number
  box: any
}
export interface Point {
  x: number
  y: number
}
export interface OverviewInfo {
  parentID: number
  childrenID: {
    right: number
    left: number
    num: number
  }
  matrixWaveID: number
  x: number
  y: number
  type: string
}

@Module({ dynamic: true, namespaced: true, name: 'ExplorationStore', store })
class Exploration extends VuexModule {
  private _indexedStations: { [id: number]: Station } = {}
  private _indexedRoutes: { [id: number]: Route } = {}
  private _selectedRoutes: Route[] = []
  private _indexedTransfers: { [id: number]: Transfer } = {};
  private _indexedRouteBranches: { [id: number]: RouteBranches} = {}
  private _selectedStationsGeoJSON: GeoJSON.FeatureCollection<GeoJSON.Point> = {type: 'FeatureCollection', features: []}
  private _selectedStationTimes = 0
  private _highlightRouteTimes = 0
  private _checkSequence: CheckSequence | null = null
  private _highlightRoutesGeoJSON: GeoJSON.FeatureCollection<GeoJSON.LineString> = {type: 'FeatureCollection', features: []}
  private _selectedRoutesGeoJSON: GeoJSON.FeatureCollection<GeoJSON.LineString> = {type: 'FeatureCollection', features: []}
  private _matrixHighLightRoute: {routes: number[], focus: number} = { routes: [], focus: -1}
  private _matrixHighLightStation : number[] = []
  private _routesTransfers: { [id: number]: RouteTransfers } = {}
  private _highlightInOutStations: GeoJSON.FeatureCollection<GeoJSON.Point> = {type: 'FeatureCollection', features: []}
  private _matrixInOutWatcher = 0
  private _displayMatrixRoute = true

  public get displayMatrixRoute () {
    return this._displayMatrixRoute
  }

  @Mutation
  public toggleDisplayMatrixRoute () {
    this._displayMatrixRoute = !this._displayMatrixRoute
  }

  public get matrixInOutWatcher () {
    return this._matrixInOutWatcher
  }

  public get inOutGeoJSON() {
    return this._highlightInOutStations
  }

  @Mutation
  public setInOutStation (args: { in: number | null, out: number | null}) {
    const {in: inStationID, out: outStationID} = args
    this._highlightInOutStations.features = []
    if (inStationID !== null) {
      this._highlightInOutStations.features.push({
        id: inStationID,
        type: 'Feature',
        geometry: {
          type: 'Point',
          coordinates: this._indexedStations[inStationID].coordinates
        },
        properties: {
          color: '#94b366'
        }
      })
    }
    if (outStationID !== null) {
      this._highlightInOutStations.features.push({
        id: outStationID,
        type: 'Feature',
        geometry: {
          type: 'Point',
          coordinates: this._indexedStations[outStationID].coordinates
        },
        properties: {
          color: '#cca766'
        }
      })
    }
    this._matrixInOutWatcher++
  }

  @Action({commit: '_addMatrixHighLightRoute'})
  public addMatrixHighLightRoute(rid: number) {
    return rid
  }

  @Mutation
  private _addMatrixHighLightRoute(rid: number) {
    this._matrixHighLightRoute.routes.push(rid)
  }

  @Action({commit: '_deleteMatrixHighLightRoute'})
  public deleteMatrixHighLightRoute(rid: number) {
    return rid
  }

  @Mutation
  private _deleteMatrixHighLightRoute(rid: number) {
    const s = new Set(this._matrixHighLightRoute.routes)
    if (s.has(rid)) {
      s.delete(rid)
    }
    this._matrixHighLightRoute.routes = Array.from(s)
  }

  @Mutation
  public emptyMatrixHighLightRoute() {
    this._matrixHighLightRoute.routes = []
  }

  @Mutation
  public setMatrixFocusRoute(focus: number) {
    this._matrixHighLightRoute.focus = focus
  }

  @Action({commit: '_toggleMatrixHighLightRoute'})
  public toggleMatrixHighLightRoute(rid?: number) {
    return rid
  }

  @Mutation
  private _toggleMatrixHighLightRoute (rid?: number) {
    if (rid) {
      // this._matrixHighLightRoute.routes = [rid]
      this._matrixHighLightRoute.focus = rid
    } else {
      this._matrixHighLightRoute.routes = []
    }
  }

  @Mutation
  public addMatrixHighlightStation(sid: number) {
    const s = new Set(this._matrixHighLightStation)
    s.add(sid)
    this._matrixHighLightStation = Array.from(s)
  }

  @Mutation
  public deleteMatrixHighLightStation(sid: number) {
    const s = new Set(this._matrixHighLightStation)
    if (s.has(sid)) {
      s.delete(sid)
    }
    this._matrixHighLightStation = Array.from(s)
  }

  public get matrixRouteSelected () {
    return this._matrixHighLightRoute.routes.length > 0;
  }

  public get matrixHighLightRoutes () {
    if (this._matrixHighLightRoute.routes.length >= 0) {
      const routes = _.map(this._matrixHighLightRoute.routes, r => { return this._indexedRoutes[r] })
      const stations : Set<number> = new Set<number>()
      const matrixRouteGeoJSON : GeoJSON.Feature<GeoJSON.LineString>[] = _.map(routes, r => {
        const focus = this._matrixHighLightRoute.focus === r.id
        _.each(r.stations, s => stations.add(s))
        return {
          type: 'Feature',
          geometry: {
            type: 'LineString',
            coordinates: r.coordinates
          },
          properties: {
            stations: r.stations,
            focus: focus,
            evaluate: Evaluation.start
          }
        }
      })
      const routeRouteGeoJSON: GeoJSON.FeatureCollection<GeoJSON.LineString> = {
        type: 'FeatureCollection',
        features: matrixRouteGeoJSON
      }
      const routeStationGeoJSON: GeoJSON.FeatureCollection<GeoJSON.Point> = {
        type: 'FeatureCollection',
        features: []
      }
      stations.forEach(s => {
        // @ts-ignore
        routeStationGeoJSON.features.push({
          id: s,
          type: 'Feature',
          geometry: {
            type: 'Point',
            coordinates: this._indexedStations[s].coordinates
          },
          properties: {}
        })
      })

      return { routes: routeRouteGeoJSON, stations: routeStationGeoJSON }
    }
  }

  public get matrixHighLightStations () {
    return this._matrixHighLightStation
  }

  public get matrixHighLightRoute () {
    return this._matrixHighLightRoute
  }

  public get selectedStationTimes() {
    return this._selectedStationTimes
  }

  public get selectedStationGeoJSON () {
    return this._selectedStationsGeoJSON
  }

  public get selectedRoutesGeoJSON () {
    return this._selectedRoutesGeoJSON
  }

  public get checkSequence () {
    return this._checkSequence
  }

  public get indexedStations () {
    return this._indexedStations
  }

  public get indexedRoutes () {
    return this._indexedRoutes
  }

  public get indexedRouteBranches() {
    return this._indexedRouteBranches
  }

  public get selectedRoutes () {
    return this._selectedRoutes
  }

  public get highlightRoutes () {
    return this._highlightRoutesGeoJSON
  }

  public get highlightRouteTimes () {
    return this._highlightRouteTimes
  }

  public get routesTransfers () {
    return this._routesTransfers
  }

  public get coordinatesMaxAndMinValue () {
    if (_.values(this._indexedStations).length > 0) {
      const stations = _.values(this._indexedStations)
      // @ts-ignore
      const lonMax = _.maxBy(stations, s => s.coordinates[0]).coordinates[0]
      // @ts-ignore
      const latMax = _.maxBy(stations, s => s.coordinates[1]).coordinates[1]
      // @ts-ignore
      const lonMin = _.minBy(stations, s => s.coordinates[0]).coordinates[0]
      // @ts-ignore
      const latMin = _.minBy(stations, s => s.coordinates[1]).coordinates[1]
      return [[lonMin, latMin], [lonMax, latMax]]
    }
  }

  @Action({ commit: '_setIndexedStations' })
  public setStations (stations: Station[]) {
    const index: { [id: number]: Station } = {}
    for (const s of stations) {
      index[s.id] = _.clone(s)
    }
    return index
  }

  @Action({ commit: '_setIndexedRoutes' })
  public setRoutes (routes: Route[]) {
    const index: { [id: number]: Route } = {}
    // const ids = []
    for (const r of routes) {
      index[r.id] = _.clone(r)
      index[r.id].states = {
        selected: false
      }
      // ids.push(r.id)
    }
    // CandidatesList.addCandidates({indexedRoutes: index, newCandidates: ids})
    return index
  }

  @Action({ commit: '_setIndexedTransfers' })
  public setIndexedTransfers (transfers: Transfer[]) {
    const index: { [id: number]: Transfer} = {}
    for (const t of transfers) {
      index[t.trip_id] = _.clone(t)
    }
    return index
  }

  @Action({ commit: '_setIndexedRouteBranches' })
  public setIndexedRouteBranches (branches: RouteBranches[]) {
    const index: { [id: number]: RouteBranches} = {}
    for (const b of branches) {
      index[b.route_id] = _.clone(b)
    }
    return index
  }

  @Action
  public toggleRoute (route: Route) {
    if (route.states.selected) {
      this.context.commit('_deselectRoute', route)
    } else {
      this.context.commit('_selectRoute', route)
    }
  }

  @Action({ commit: '_setHighlightRoutesGeoJSON'})
  public setHighlightRoutesGeoJSON ({routes, selected}: {routes: number[], selected: boolean}) {
    const features: GeoJSON.Feature<GeoJSON.LineString>[] = []
    _.each(
      routes,
      r => {
        const route = this.indexedRoutes[r]
        const feature = {
          id: route.id,
          type: "Feature",
          geometry: {
            type: 'LineString',
            coordinates: this.indexedRoutes[route.id].coordinates
          },
          properties: {}
        }
        // @ts-ignore
        features.push(feature)
      }
    )
    return {features, selected}
  }

  @Action({ commit: '_setSelectedStationsGeoJSON' })
  public setSelectedStationsGeoJSON (stations: number[]) {
    const features: GeoJSON.Feature<GeoJSON.Point>[] = []
    _.each(
      stations,
      s => {
        const station = this.indexedStations[s]
        const feature = {
          id: station.id,
          type: "Feature",
          geometry: {
            type: 'Point',
            coordinates: station.coordinates
          },
          properties: {}
        }
        // @ts-ignore
        features.push(feature)
      }
    )
    return features
  }

  @Action({ commit: '_setCheckSequence' })
  async loadCheckSequence () {
    let json: CheckSequence | null = null
    const resp = await fetch(TRANSFER_JSON_URL)
    json = await resp.json() as CheckSequence
    return json
  }

  @Action({ commit: '_clearSelectedStation' })
  public clearSelectedStation () {
    // @types/ignore

  }

  @Action({commit: '_clearHighlightRoutes' })
  public clearHighlightRoutes () {
    // @types/ignore

  }

  @Mutation
  private _setCheckSequence (checkSequence: CheckSequence) {
    this._checkSequence = checkSequence
  }

  @Mutation
  private _setIndexedStations (indexedStations: { [id: number]: Station }) {
    this._indexedStations = indexedStations
  }

  @Mutation
  private _setIndexedRoutes (indexedRoutes: { [id: number]: Route }) {
    this._indexedRoutes = indexedRoutes
  }

  @Mutation
  private _setIndexedRouteBranches (indexedRouteBranches: { [id: number]: RouteBranches }) {
    this._indexedRouteBranches = indexedRouteBranches
  }

  @Mutation
  private _setIndexedTransfers (indexedTransfers: { [id: number]: Transfer}) {
    this._indexedTransfers = indexedTransfers
  }

  @Mutation
  private _selectRoute (route: Route) {
    console.log(`[Exploration] Selecting route ${route.name}...`)
    route.states.selected = true
    this._selectedRoutes.push(route)
  }

  @Mutation
  private _deselectRoute (route: Route) {
    console.log(`[Exploration] Deselecting route ${route.name}...`)
    route.states.selected = false
    this._selectedRoutes = this._selectedRoutes.filter(r => r !== route)
  }

  @Mutation
  private _setSelectedStationsGeoJSON (features : GeoJSON.Feature<GeoJSON.Point>[]) {
    this._selectedStationsGeoJSON.features = features
    this._selectedStationTimes += 1
  }

  @Mutation
  private _setHighlightRoutesGeoJSON ({features, selected}: { features : GeoJSON.Feature<GeoJSON.LineString>[], selected: boolean }) {
    if (selected) {
      // this._selectedRoutesGeoJSON.features = _.concat(this._selectedRoutesGeoJSON.features, features)
      this._selectedRoutesGeoJSON.features = features
      // Vue.set(this._selectedRoutesGeoJSON, 'features', features)
      console.log(this._selectedRoutesGeoJSON)
    } else {
      this._highlightRoutesGeoJSON.features = features
      this._highlightRouteTimes += 1
    }
  }

  @Mutation
  public setRoutesGeoJSON(routeGeoJSON: GeoJSON.FeatureCollection<GeoJSON.LineString>) {
    this._selectedRoutesGeoJSON = routeGeoJSON
  }

  @Mutation
  private _clearSelectedStation() {
    this._selectedStationsGeoJSON.features = []
    this._selectedStationTimes -= 1
  }

  @Mutation
  private _clearHighlightRoutes () {
    this._highlightRoutesGeoJSON.features = []
    this._highlightRouteTimes -= 1
  }

  @Mutation
  public computeRoutesTransfers (args: { records: Transfer[], routes: Route[] }) {
    this._indexedRouteBranches = {}
    const {records, routes} = args
    // initialization
    _.each(routes, route => {
      const checkinCount = {}
      const checkoutCount = {}
      const checkinDetails = {}
      const checkoutDetails = {}
      _.each(route.stations, s => {
        checkinCount[s] = 0
        checkoutCount[s] = 0
        checkinDetails[s] = {}
        checkoutDetails[s] = {}
      })
      this._routesTransfers[route.id] = {
        checkinCount: checkinCount,
        checkoutCount: checkoutCount,
        checkinDetail: checkinDetails,
        checkoutDetail: checkoutDetails
      }
    })
    const indexedTransfers : {[id: number]: Transfer} = {}
    // check all transfer records
    _.each(records, record => {
      indexedTransfers[record.trip_id] = record
      // check out route
      this._routesTransfers[record.from_route_id].checkoutCount[record.from_station_id] += 1
      if (!(record.to_route_id in this._routesTransfers[record.from_route_id].checkoutDetail[record.from_station_id])) {
        this._routesTransfers[record.from_route_id].checkoutDetail[record.from_station_id][record.to_route_id] = []
      }
      this._routesTransfers[record.from_route_id].checkoutDetail[record.from_station_id][record.to_route_id].push(record.trip_id)
      // check in route
      this._routesTransfers[record.to_route_id].checkinCount[record.to_station_id] += 1
      if (!(record.from_route_id in this._routesTransfers[record.to_route_id].checkinDetail[record.to_station_id])) {
        this._routesTransfers[record.to_route_id].checkinDetail[record.to_station_id][record.from_route_id] = []
      }
      this._routesTransfers[record.to_route_id].checkinDetail[record.to_station_id][record.from_route_id].push(record.trip_id)
    })
    console.log('routes transfer', this._routesTransfers)
    this._indexedRouteBranches = RouteTransfersToRouteBranches(this._routesTransfers, indexedTransfers)
    console.log('route branches', this._indexedRouteBranches)
  }
}

export default getModule(Exploration)
