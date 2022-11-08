import {Action, getModule, Module, Mutation, VuexModule} from "vuex-module-decorators";
import store from "@/store";
import {Conflict, RouteGroup, RouteGroupCollection} from "@/utils/RouteGroup";
import {RouteList, SRoute, Station, Trip} from "@/utils/MCTSTypes";
import CandidatesList from "@/store/modules/CandidatesList";
import Exploration from "@/store/modules/Exploration";
import {GeoJSON} from "geojson";
import _ from 'lodash'
import {gqlQuery} from "@/utils/GraphqlAPI";
import {F_MATRIX, TRIPS_NUM_BETWEEN_STATIONS} from "@/query";

@Module({name: 'Evaluation', dynamic: true, namespaced: true, store})
class Evaluation extends VuexModule {
  private _start = false
  private _groupCollection: RouteGroupCollection | null = null
  private _conflicts: Conflict[] = []
  private _conflictsLineFeatures : GeoJSON.Feature<GeoJSON.LineString>[] = []
  private _conflictsPointFeatures : GeoJSON.Feature<GeoJSON.Point>[] = []
  private _conflictResolvedNum = -1
  private _currentRoute: number[] = []
  private _geoJsonUpdateWatcher = 0
  private _highlightRoutesWatcher = 0
  private _highlightRoutesGeoJSON : GeoJSON.FeatureCollection<GeoJSON.LineString> = {type: "FeatureCollection", features: []}
  private _currentRouteMatrix: {[id: number]: {[id: number]: number}} | null = null
  private _currentRouteCheckin: {[id: number]: Trip[]} = {}
  private _currentRouteCheckout: {[id: number]: Trip[]} = {}

  public get currentRouteCheckin () {
    return this._currentRouteCheckin
  }

  public get currentRouteCheckout () {
    return this._currentRouteCheckout
  }

  public get highlightRoutesGeoJSON () {
    return this._highlightRoutesGeoJSON
  }

  public get highlightRoutesWatcher () {
    return this._highlightRoutesWatcher
  }

  public get currentRouteMatrix () {
    return this._currentRouteMatrix
  }

  public get currentRoute () {
    return this._currentRoute
  }

  public get geoJsonUpdateWatcher () {
    return this._geoJsonUpdateWatcher
  }

  public get start() {
    return this._start
  }

  public get groupCollection() {
    return this._groupCollection
  }

  public get conflicts () {
    return this._conflicts
  }

  public get conflictGraphGeoJSON () {
    return {
      line: {
        type: 'FeatureCollection',
        features: this._conflictsLineFeatures
      },
      point: {
        type: 'FeatureCollection',
        features: this._conflictsPointFeatures
      }
    }
  }

  public get conflictResolvedNum () {
    return this._conflictResolvedNum
  }

  @Action({commit: '_updateConflictGeoJson'})
  public updateConflictGeoJson() {
    return undefined
  }

  @Mutation
  private _updateConflictGeoJson() {
    console.log('_updateConflictGeoJson')
    this._conflictsLineFeatures = (this._groupCollection as RouteGroupCollection).linkGeoJSONFeatures()
    this._conflictsPointFeatures = (this._groupCollection as RouteGroupCollection).pointGeoJSONFeatures()
    this._geoJsonUpdateWatcher++
  }

  @Action({commit: '_createRouteGroupCollection'})
  public createRouteGroupCollection(args : { routes: SRoute[], orderedStations: number[]}) {
    return {routes: args.routes, orderedStations: args.orderedStations}
  }

  @Action({commit: '_setHighlightRoutes'})
  public setHighLightRoutesByStation(sid: number) {
    let routes : SRoute[] = []
    if (this._groupCollection) {
      const stationIndex = this._groupCollection.stationIndex
      this._groupCollection.routeGroups.forEach(group => {
        if (group.stationVector[stationIndex[sid]] === 1) {
          routes = _.concat(routes, group.routes)
        }
      })
    }
    return routes
  }

  @Action({commit: '_setHighlightRoutes'})
  public setHighLightRoutesByGroup(gid: number) {
    if (this._groupCollection) {
      return this._groupCollection.indexRouteGroups[gid].routes
    }
  }

  @Mutation
  public clearHighlightRoutes() {
    this._highlightRoutesGeoJSON.features = []
    this._highlightRoutesWatcher++
  }

  @Mutation
  private _setHighlightRoutes(routes: SRoute[]) {
    const features: GeoJSON.Feature<GeoJSON.LineString>[] = []
    if (routes.length > 0) {
      routes.forEach(route => {
        features.push({
          id: route.id,
          type: 'Feature',
          geometry: {
            type: 'LineString',
            coordinates: route.coordinates
          },
          properties: {}
        })
      })
    }
    console.log(features)
    this._highlightRoutesGeoJSON.features = features
    this._highlightRoutesWatcher++
  }

  @Mutation
  private async _createRouteGroupCollection(args : { routes: SRoute[], orderedStations: number[]}) {
    this._start = true
    this._groupCollection = new RouteGroupCollection(args.routes, args.orderedStations, Exploration.indexedStations)
    this._conflicts = this._groupCollection.searchConflicts()
    this._currentRoute = this._groupCollection.currentRoute
    console.log("current route: " + this._currentRoute)
    CandidatesList.setConflictsToLineup(this._conflicts)
    this._conflictsLineFeatures = this._groupCollection.linkGeoJSONFeatures()
    this._conflictsPointFeatures = this._groupCollection.pointGeoJSONFeatures()
    this._conflictResolvedNum++
    // @ts-ignore
    this._currentRouteMatrix = await this._groupCollection.computeCurrentMatrix()
    const {checkin: cin, checkout: cout} = await this._groupCollection.computeCurrentRouteTrips()
    this._currentRouteCheckout = cout
    this._currentRouteCheckin = cin
  }

  // Select Group on Map
  @Action({commit: '_constructNewGroupCollectionByRoutes'})
  public selectGroup (id: number) {
    if (this._groupCollection) {
      let routes: SRoute[] = []
      this._groupCollection.routeGroups.forEach(group => {
        // @ts-ignore
        if (group.stationVector[this._groupCollection.stationIndex[id]] === 1) {
          routes = _.concat(routes, group.routes)
        }
      })
      return routes
    }
  }

  // Select Group By Group ID
  @Action({commit: '_constructNewGroupCollectionByRoutes'})
  public selectGroupByGroupID (gid: number) {
    if (this._groupCollection) {
      return this._groupCollection.indexRouteGroups[gid].routes
    }
  }

  @Mutation
  private async _constructNewGroupCollectionByRoutes(routes: SRoute[]) {
    if (this._groupCollection) {
      this._groupCollection = new RouteGroupCollection(
        routes,
        this._groupCollection.orderedStations,
        Exploration.indexedStations
      )
      console.log('confict group number', this._groupCollection.routeGroups.length)
      this._conflicts = this._groupCollection.searchConflicts()
      this._currentRoute = this._groupCollection.currentRoute
      this._conflictsLineFeatures = this._groupCollection.linkGeoJSONFeatures()
      this._conflictsPointFeatures = this._groupCollection.pointGeoJSONFeatures()
      CandidatesList.setConflictsToLineup(this._conflicts)
      this._conflictResolvedNum++
      // @ts-ignore
      this._currentRouteMatrix = await this._groupCollection.computeCurrentMatrix()
      const {checkin: cin, checkout: cout} = await this._groupCollection.computeCurrentRouteTrips()
      this._currentRouteCheckout = cout
      this._currentRouteCheckin = cin
    }
  }

  @Mutation
  public _setHighLightGroup(gid: number) {
    if (this._groupCollection) {
      this._groupCollection.routeGroups.forEach(group => {
        if (group.id !== gid) {
          group.highlight = false
        }
      })
    }
  }

  @Mutation
  public _clearHighLightGroup(gid: number) {
    if (this._groupCollection) {
      this._groupCollection.routeGroups.forEach(group => {
        if (group.id !== gid) {
          group.highlight = true
        }
      })
    }
  }
}

export default getModule(Evaluation)
