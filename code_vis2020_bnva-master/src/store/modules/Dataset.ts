import { Action, Module, Mutation, VuexModule, getModule } from 'vuex-module-decorators'
import { GeoJSON } from 'geojson'
import _ from 'lodash'

import store from '@/store'
import Exploration, { Route, Station, TransferData } from '@/store/modules/Exploration'
import Projection, { StationClusters } from "@/store/modules/Projection"
import CandidatesList from "@/store/modules/CandidatesList";
const REPOSITORY_URL = 'data/sources.json'
const STATION_GEOJSON_URL = '/bus_stations03.geojson'
const ROUTE_GEOJSON_URL = '/bus_routes07.geojson'
const TRANSFER_JSON_URL = '/transfer_records.json'
const CLUSTER_JSON_URL = '/station_cluster03.json'

export interface DataSource {
  id: string
  name: string
}

@Module({ dynamic: true, namespaced: true, name: 'DatasetStore', store })
class Dataset extends VuexModule {
  private _dataSources: DataSource[] = []
  private _selectedDataSource: DataSource | null = null
  private _stationGeoJSON: GeoJSON.FeatureCollection<GeoJSON.Point> | null = null
  private _routeGeoJSON: GeoJSON.FeatureCollection<GeoJSON.LineString> | null = null
  private _transferJSON: TransferData | null = null
  private _clusterJSON: StationClusters | null = null

  public get dataSources () {
    return this._dataSources
  }

  public get selectedDataSource () {
    return this._selectedDataSource
  }

  public get routeGeoJSON () {
    return this._routeGeoJSON
  }

  public get stationGeoJSON () {
    return this._stationGeoJSON
  }

  public get transferJson () {
    return this._transferJSON
  }

  @Action
  async loadDataSources () {
    const resp = await fetch(REPOSITORY_URL)
    this.context.commit('_setDataSources', await resp.json())
    await this.context.dispatch('loadSelectedDataSource')
  }

  @Action({ commit: '_setStationGeoJSON' })
  async loadStationGeoJSON () {
    let geojson: GeoJSON.FeatureCollection<GeoJSON.Point> | null = null
    if (this._selectedDataSource == null) {
      console.error('[Dataset::loadStationGeoJSON] No data source is selected')
    } else {
      const dataSourceURL = `data/${this._selectedDataSource.id}`
      const resp = await fetch(dataSourceURL + STATION_GEOJSON_URL)
      geojson = await resp.json() as GeoJSON.FeatureCollection<GeoJSON.Point>
    }
    return geojson
  }

  @Action({ commit: '_setRouteGeoJSON' })
  async loadRouteGeoJSON () {
    let geojson: GeoJSON.FeatureCollection<GeoJSON.LineString> | null = null
    if (this._selectedDataSource == null) {
      console.error('[Dataset::loadRouteGeoJSON] No data source is selected')
    } else {
      const dataSourceURL = `data/${this._selectedDataSource.id}`
      const resp = await fetch(dataSourceURL + ROUTE_GEOJSON_URL)
      geojson = await resp.json() as GeoJSON.FeatureCollection<GeoJSON.LineString>
    }
    return geojson
  }

  @Action({ commit: '_setTransferJSON' })
  async loadTransferJSON () {
    let json: TransferData | null = null
    if (this._selectedDataSource == null) {
      console.error('[Dataset::loadTransferJSON] No data source is selected')
    } else {
      const dataSourceURL = `data/${this._selectedDataSource.id}`
      const resp = await fetch(dataSourceURL + TRANSFER_JSON_URL)
      json = await resp.json() as TransferData
    }

    return json
  }

  @Action({ commit: '_setClusterJSON' })
  async loadClusterJSON () {
    let json: StationClusters | null = null
    if (this._selectedDataSource == null) {
      console.error('[Dataset::loadClusterJSON] No data source is selected')
    } else {
      const dataSourceURL = `data/${this._selectedDataSource.id}`
      const resp = await fetch(dataSourceURL + CLUSTER_JSON_URL)
      json = await resp.json() as StationClusters
    }

    return json
  }

  @Action
  async loadSelectedDataSource () {
    await this.context.dispatch('loadStationGeoJSON')
    await this.context.dispatch('loadRouteGeoJSON')
    await this.context.dispatch('loadTransferJSON')
    await this.context.dispatch('loadClusterJSON')

    if (this._stationGeoJSON && this._routeGeoJSON && this._transferJSON && this._clusterJSON) {
      const stations = this._stationGeoJSON.features.map(f => f.properties) as Station[]
      Exploration.setStations(stations)
      const routes = this._routeGeoJSON.features.map(f => _.defaults(f.properties, f.geometry)) as unknown as Route[]
      Exploration.setRoutes(routes)
      // Exploration.setIndexedRouteBranches(this._transferJSON.route_branches)
      Exploration.computeRoutesTransfers({ records: this._transferJSON.transfer_records, routes: routes })
      Projection.setClusters(this._clusterJSON)
      Projection.computeClusterGeoCenters(this._clusterJSON)
      Projection.setCenterGeoJSON()
      Exploration.setRoutesGeoJSON(this._routeGeoJSON)
    }
  }

  @Mutation
  private _setDataSources (dataSources: DataSource[]) {
    this._dataSources = dataSources
    this._selectedDataSource = dataSources[0]
  }

  @Mutation
  private _setSelectedDataSource (dataSource: DataSource) {
    this._selectedDataSource = dataSource
  }

  @Mutation
  private _setStationGeoJSON (stationGeoJSON: GeoJSON.FeatureCollection<GeoJSON.Point>) {
    this._stationGeoJSON = stationGeoJSON
  }

  @Mutation
  private _setRouteGeoJSON (routeGeoJSON: GeoJSON.FeatureCollection<GeoJSON.LineString>) {
    this._routeGeoJSON = routeGeoJSON
  }

  @Mutation
  private _setTransferJSON (transferJson: TransferData) {
    this._transferJSON = transferJson
  }

  @Mutation
  private _setClusterJSON (clusters: StationClusters) {
    this._clusterJSON = clusters
  }
}

export default getModule(Dataset)
