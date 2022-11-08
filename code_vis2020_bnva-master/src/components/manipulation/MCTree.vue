<template>
  <div></div>
</template>

<script lang="ts">
import 'reflect-metadata'
import {Component, Prop, Vue, Watch} from 'vue-property-decorator';
import Manipulation from '@/store/modules/Manipulation'
import { DeferredPromise } from '@/utils';
import mapboxgl from 'mapbox-gl'
import CandidatesList from '@/store/modules/CandidatesList'
import Evaluation from "@/store/modules/Evaluation";

@Component
export default class MCTree extends Vue {
  private routesListInterval : any

  get remove () {
    return Evaluation.start
  }

  get checkSnapshot () {
    return Manipulation.checkSnapshot
  }

  get currentSnapshot () {
    return Manipulation.currentSnapshotNum
  }

  @Prop()
  map!: DeferredPromise<mapboxgl.Map>

  get routeListsGeoJSON() {
    return Manipulation.routesListGeoJSON
  }

  get receiveDataCount() {
    return Manipulation.receiveDataCount
  }

  get running() {
    return Manipulation.running
  }

  get stationGraphGeoJSON () {
    return Manipulation.stationGraphGeoJSON
  }

  @Watch('remove', {immediate: true, deep: true})
  public async removeMCTree(val: boolean) {
    console.log('[MCTree] remove layers...')
    const map = await this.map.get()
    if (val) {
      const nodesSource = map.getSource('stationGraphNodes')
      const linksSource = map.getSource('stationGraphLink')
      if (nodesSource) {
        console.log('[MCTree] remove node layers...')
        map.removeLayer('stationGraphNodes')
        map.removeSource('stationGraphNodes')
      }
      if (linksSource) {
        map.removeLayer('stationGraphLink')
        map.removeSource('stationGraphLink')
      }
      const listSource = map.getSource('routesList')
      if (listSource) {
        console.log('[MCTree] remove list layers...')
        map.removeLayer('routesList')
        map.removeSource('routesList')
      }
    }
  }

  @Watch('currentSnapshot', {immediate: true})
  public async drawSnapshot () {
    console.log('[MCTree] Redrawing snapshot...')
    const map = await this.map.get()

    const oldSource = map.getSource('routesList')
    if (oldSource) {
      if (this.checkSnapshot && Manipulation.currentSnapShot) {
        oldSource.setData(Manipulation.currentSnapShot)
      } else {
        oldSource.setData(this.routeListsGeoJSON)
      }
    }
  }

  @Watch('stationGraphGeoJSON', {immediate: true, deep: true})
  public async drawStationGraph () {
    if (!this.stationGraphGeoJSON) {
      return
    }
    const {nodes, links} = this.stationGraphGeoJSON
    console.log('[MCTree] Redrawing station graph...')
    const map = await this.map.get()
    const oldSource = map.getSource('stationGraphNodes')
    const oldLinkSource = map.getSource('stationGraphLink')
    if (oldSource) {
      oldSource.setData(nodes)
      oldLinkSource.setData(links)
      return
    }

    if (nodes && links) {
      map.addSource('stationGraphLink', {
        type: 'geojson',
        data: links
      })
      map.addSource('stationGraphNodes', {
        type: 'geojson',
        data: nodes
      })
      map.addLayer({
        id: 'stationGraphLink',
        type: 'line',
        source: 'stationGraphLink',
        paint: {
          'line-color': '#bbb',
          'line-opacity': 0.1,
          'line-width': 3
        }
      })
      map.addLayer({
        id: 'stationGraphNodes',
        type: 'circle',
        source: 'stationGraphNodes',
        paint: {
          'circle-stroke-color': '#7585d7',
          'circle-stroke-width': [
            'case',
            ['boolean', ['feature-state', 'hover'], false],
            1.5,
            [
              'case',
              ['get', 'isOD'],
              1.5,
              0.5
            ]
          ],
          'circle-radius': [
            'case',
            ['boolean', ['feature-state', 'hover'], false],
            5,
            [
              'case',
              ['get', 'isOD'],
              5,
              4
            ]
          ],
          'circle-color': [
            'case',
            ['boolean', ['feature-state', 'hover'], false],
            'white',
            [
              'case',
              ['get', 'isOD'],
              'white',
              '#b6d4e6'
            ]
          ],
          'circle-opacity': [
            'case',
            ['boolean', ['feature-state', 'hover'], false],
            1,
            [
              'case',
              ['get', 'isOD'],
              1,
              0.6
            ]
          ]
        }
      })
      if (map.getLayer('routesList')) {
        map.moveLayer('stationGraphNodes', 'routesList')
      }
      map.on('click', 'stationGraphNodes', (e) => {
        if (e.features && e.features[0].id) {
          map.setFeatureState(
            { source: 'stationGraphNodes', id: e.features[0].id.toString() },
            { hover: true }
          );
          Manipulation.addStopInRunTime(parseInt(e.features[0].id.toString()))
        }
      })
    }
  }

  @Watch('running', {immediate: true})
  public setRouteListInterval(val: boolean, oldVal: boolean) {
    if (!oldVal && val) {
      this.routesListInterval = window.setInterval(() => {
        Manipulation.queryCurrentRouteLists()
      }, 300)
    } else if (oldVal && !val) {
      clearInterval(this.routesListInterval)
    }
  }

  @Watch('receiveDataCount', {immediate: true, deep: true})
  public async drawRoutesList () {
    if (Evaluation.start) {
      return
    }
    const map = await this.map.get()

    if (this.routeListsGeoJSON) {
      Manipulation.saveCurrentPlaning(this.routeListsGeoJSON)
      CandidatesList.clearCandidates()
      CandidatesList.addPossibleSolutions(Manipulation.routesList)
    }
    const oldSource = map.getSource('routesList')
    if (oldSource) {
      if (this.checkSnapshot && Manipulation.currentSnapShot) {
        oldSource.setData(Manipulation.currentSnapShot)
      } else {
        oldSource.setData(this.routeListsGeoJSON)
      }
      return
    }

    if (this.routeListsGeoJSON) {
      map.addSource('routesList', {
        type: 'geojson',
        data: this.routeListsGeoJSON
      })
      map.addLayer({
        id: 'routesList',
        type: 'line',
        source: 'routesList',
        paint: {
          'line-color': '#7585d7',
          'line-opacity': 0.2,
          'line-width': 1.5
        }
      })
      if (map.getLayer('stationGraphNodes')) {
        map.moveLayer('routesList', 'stationGraphNodes')
      }
    }
  }
}
</script>

<style scoped>

</style>
