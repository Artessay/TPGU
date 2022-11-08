import {Stage} from "@/utils/Types";
import {Stage} from "@/utils/Types";
<template>
  <div id="map"
    @mousemove="mousePosition = { x: $event.clientX, y: $event.clientY }"
    @mouseleave="mousePosition = null"
  >
    <div id="mapbox"></div>
    <ConflictGraph :map="map"></ConflictGraph>
    <MCTree :map="map"></MCTree>
    <MapTooltip :pinned="tooltipPinned" :mouse-position="mousePosition" :route-features="hoveredRouteFeatures" ref="tooltip"></MapTooltip>
  </div>
</template>

<script lang="ts">
import {Component, Ref, Vue, Watch} from 'vue-property-decorator'
import mapboxgl from 'mapbox-gl'
import _ from 'lodash'
import * as d3 from 'd3'

import MapTooltip from '@/components/explore/MapTooltip.vue'
import {DeferredPromise} from '@/utils'
import Dataset from '@/store/modules/Dataset'
import Exploration from '@/store/modules/Exploration'
import Projection from '@/store/modules/Projection'
import CandidatesList from '@/store/modules/CandidatesList'
import {Stage} from '@/utils/Types'
import Manipulation from '@/store/modules/Manipulation'
import MCTree from '@/components/manipulation/MCTree.vue';
import ConflictGraph from "@/components/evaluation/ConflictGraph.vue";
import Evaluation from "@/store/modules/Evaluation";

mapboxgl.accessToken = 'pk.eyJ1Ijoic2tpZXM0NTciLCJhIjoiY2s1NWdzYWk3MHU0azNrcjV1YjI3Z3p0bCJ9.CRPi6rGEW0_LkNKLXqKzqQ'

@Component({
  components: {
    ConflictGraph,
    MapTooltip,
    MCTree
  }
})
export default class Map extends Vue {
  public map: DeferredPromise<mapboxgl.Map> = new DeferredPromise()
  public hoveredRouteFeatures: mapboxgl.MapboxGeoJSONFeature[] = []
  public mousePosition: { x: number, y: number } | null = null
  public tooltipPinned = false
  public highlightClusterLinkID : string | null = null
  public highlightClusterID : number | string= -1
  public linkClickTimes = 0
  public svg!: d3.Selection<SVGSVGElement, unknown, null, undefined>
  public stage : Stage = Stage.EXPLORATION
  public dots: any = []
  public lineupHoverRouteID = -1

  @Ref('tooltip') public tooltip!: MapTooltip

  public get currentStage () {
    if (Manipulation.start && Evaluation.start) {
      this.stage = Stage.MANIPULATION
    } else if (CandidatesList.highlightCandidate) {
      this.stage = Stage.HIGHLIGHT
    } else {
      this.stage = Stage.EXPLORATION
    }
    return this.stage
  }

  public get voronoiPolygons () {
    return Projection.voronoiPolygons
  }

  public get stationExtends () {
    return Exploration.coordinatesMaxAndMinValue
  }

  public get selectedRoutesGeoJSON () {
    return { GeoJSON: Exploration.selectedRoutesGeoJSON, hidden: Exploration.matrixRouteSelected }
  }

  public get routeGeoJSON () {
    return Dataset.routeGeoJSON
  }

  public get stationGeoJSON () {
    return Dataset.stationGeoJSON
  }

  public get selectedStations () {
    return Exploration.selectedStationGeoJSON
  }

  public get selectedStationTimes () {
    return Exploration.selectedStationTimes
  }

  public get highlightRoutesTimes () {
    return Exploration.highlightRouteTimes
  }

  public get highlightRoutes () {
    return Exploration.highlightRoutes
  }

  public get clusterCenters () {
    return Projection.mapCenters
  }

  public get clusterLinks () {
    return Projection.mapLinks
  }

  public get contour () {
    return { contour: Projection.contourGeoJSON, hidden: Exploration.matrixRouteSelected }
  }

  public get matrixRoute () {
    return Exploration.matrixHighLightRoutes
  }

  public get matrixStations() {
    return Exploration.matrixHighLightStations
  }

  public get hoverRoute () {
    return CandidatesList.hoverRoute
  }

  public mounted () {
    const map = new mapboxgl.Map({
      container: 'mapbox',
      style: 'mapbox://styles/skies457/ck7hnlzqy2pyn1inrpza4rdnk',
      center: [116.37, 39.90],
      zoom: 11
    })
    const container = map.getCanvasContainer()
    this.svg = d3.select(container).append("svg")
      .attr('class', 'map-svg')
      .attr('transform', 'translate(0, -20)') // hack
    map.on('viewreset', this._myRender)
    map.on('move', this._myRender)
    map.on('load', () => {
      map.getStyle()!.layers!.forEach(function(thisLayer) {
        if (thisLayer.type === 'symbol') {
          map.setLayoutProperty(thisLayer.id, 'text-field', ['get', 'name_en'])
        }
      })
      this.map.resolve(map)
    })
    map.on('mousemove', e => this._onMapMouseMove(e))
    map.on('click', () => this._onMapClick())
    // this.drawRoutes()
    this.drawStations()
  }

  public async getD3(): Promise<[d3.GeoProjection, number]> {
    const map = await this.map.get()
    const bbox = document.body.getBoundingClientRect();
    const center = map.getCenter();
    const zoom = map.getZoom();
    // 512 is hardcoded tile size, might need to be 256 or changed to suit your map config
    const scale = (512) * 0.5 / Math.PI * Math.pow(2, zoom);

    const d3projection = d3.geoMercator()
      .center([center.lng, center.lat])
      .translate([bbox.width / 2, bbox.height / 2])
      .scale(scale);

    return [d3projection, zoom]
  }

  @Watch('currentStage', {immediate: true, deep: true})
  public async changeStage (val: Stage, oldVal: Stage) {
    const map = await this.map.get()
    if (val !== Stage.EXPLORATION) {
      const targetLayer = map.getLayer('targetRoute')
      if (targetLayer) {
        map.setLayoutProperty('targetRoute', 'visibility', 'none')
      }
      const stationLayer = map.getLayer('matrixStations')
      if (stationLayer) {
        map.setLayoutProperty('matrixStations', 'visibility', 'none')
      }
      const heatSource = map.getSource('stations')
      if (heatSource) {
        map.removeLayer('stations-heat')
        map.removeSource('stations')
      }
      const matrixRouteSource = map.getSource('matrixRoute')
      if (matrixRouteSource) {
        map.removeLayer('matrixRoute')
        map.removeSource('matrixRoute')
      }
      _.each(this.voronoiPolygons, (cluster, id) => {
        const oldSource = map.getSource('cluster' + id)
        if (oldSource) {
          map.removeLayer('cluster-polygon' + id)
          // map.removeLayer('cluster-line' + id)
          map.removeSource('cluster' + id)
        }
      })
      const oldSource = map.getSource('selected-routes')
      if (oldSource) {
        map.removeLayer('selected-routes')
        map.removeSource('selected-routes')
      }
      const linkSource = map.getSource('cluster-links')
      if (linkSource) {
        map.removeLayer('cluster-links')
        map.removeSource('cluster-links')
      }
      const conSource = map.getSource('con')
      if (conSource) {
        map.removeLayer('con')
        map.removeSource('con')
      }
      d3.selectAll(".map-svg > *").remove();
    }
  }

  @Watch('clusterCenters', { immediate: true, deep: true })
  public async drawGlyphWithSvg () {
    if (this.clusterCenters.features.length === 0) {
      Projection.setCenterGeoJSON()
      return
    }

    const d3Projection = this.getD3()[0]
    if (this.clusterCenters && this.svg) {
      const map = await this.map.get()
      const dots = this.svg.selectAll("g")
        .data(this.clusterCenters.features)
        .enter().append("g")
        .attr('class', 'test-dot-class')
        .on('click', function (d) {
          if (d.properties && d.id) {
            if (Projection.selectedCenters) {
              if (Projection.selectedCenters.has(parseInt(d.properties.key))) {
                d3.select(this).select('.glyph-background').attr('stroke', '#e6e6e6')
                map.setFeatureState(
                  { source: 'cluster-centers', id: d.id.toString() },
                  { hover: false }
                );
              } else {
                d3.select(this).select('.glyph-background').attr('stroke', '#8856a7')
                map.setFeatureState(
                  { source: 'cluster-centers', id: d.id.toString() },
                  { hover: true }
                );
              }
            }
            Projection.toggleCluster(parseInt(d.properties.key))
          }
        })
        .on('mouseenter', (d) => {
          if (d.id) {
            map.setPaintProperty(
              'cluster-polygon' + d.id,
              'fill-opacity',
              ['interpolate',
                ['linear'],
                ['zoom'],
                9,
                0.1,
                15,
                ['get', 'value']])
            this.highlightClusterID = d.id
          }
        })
        .on('mouseleave', (d) => {
          if (this.highlightClusterID > 0) {
            map.setPaintProperty('cluster-polygon' + this.highlightClusterID, 'fill-opacity', 0)
            this.highlightClusterID = -1
          }
        })
        .call((g) => {
          const arc360 = 2 * Math.PI
          const radarRadius = 14
          const innerRadius = 19
          const outerRadius = 24

          // 最外层白底
          g.append('circle')
            .attr('class', 'glyph-background')
            .attr('r', 25)
            .attr('fill', '#fff')
            .attr('stroke-width', 1)
            .attr('stroke', '#e6e6e6')

          g.append("path")
            .attr("fill", "#ffd178")
            .attr("d", (d) => {
              if (d.id) {
                if (Projection.clusterVol) {
                  const max = _.max(Projection.clusterVol[d.id].out)
                  const min = _.min(Projection.clusterVol[d.id].out)
                  const series : number[] = []
                  Projection.clusterVol[d.id].out.forEach(val => {
                    // @ts-ignore
                    series.push((val - min) / (max - min))
                  })
                  series.push(series[0])
                  const seriesFormatted = series.map((v, i) => [v, i]) as [number, number][]
                  const x = d3.scaleLinear()
                    .range([0, arc360])
                    .domain([0, series.length - 1])
                  const y = d3.scaleSqrt()
                    .domain([0, 1])
                    .range([innerRadius, outerRadius])
                  const lineStr = d3.lineRadial()
                    .angle((d) => x(d[1]))
                    .radius((d) => y(d[0]))
                    .curve(d3.curveCardinal)(seriesFormatted)
                  return lineStr as string
                }
              }
              return ''
            })

          // 绿色圈
          g.append('circle')
            .attr('r', (d) => {
              const max = _.max(Projection.clusterVol[d.id].in)
              const min = _.min(Projection.clusterVol[d.id].in)
              if (min === max) {
                return 0
              }
              return innerRadius
            })
            .attr('fill', '#b3e074')

          g.append("path")
            .attr("fill", "#fff")
            .attr("d", (d) => {
              if (d.id) {
                if (Projection.clusterVol) {
                  const max = _.max(Projection.clusterVol[d.id].in)
                  const min = _.min(Projection.clusterVol[d.id].in)
                  const series2 : number[] = []
                  Projection.clusterVol[d.id].in.forEach(val => {
                    // @ts-ignore
                    series2.push((val - min) / (max - min))
                  })
                  series2.push(series2[0])
                  const series2Formatted = series2.map((v, i) => [v, i]) as [number, number][]
                  const x2 = d3.scaleLinear()
                    .range([0, arc360])
                    .domain([0, series2.length - 1])
                  const y2 = d3.scaleSqrt()
                    .domain([1, 0])
                    .range([radarRadius, innerRadius])
                  const lineStr2 = d3.lineRadial()
                    .angle(d => x2(d[1]))
                    .radius(d => y2(d[0]))
                    .curve(d3.curveCardinal)(series2Formatted)
                  return lineStr2 as string
                }
              }
              return ''
            })

          // 雷达图
          const values = [0.5, 0.7, 0.2, 0.3, 0.6, 1]
          const nSector = values.length
          const onePiece = arc360 / nSector
          // 雷达图框框
          g.append('g')
            .attr('class', 'radar-circles-g')
            .selectAll('circle')
            .data([1, 2, 3])
            .enter().append('circle')
            .attr('r', d => (radarRadius - 1) * d / 3 )
            .attr('fill', 'none')
            .attr('stroke-width', 0.5)
            .style("stroke-dasharray", ("1, 1"))
            .attr('stroke', '#dce7f2')
          // 雷达图线
          g.append('g')
            .attr('class', 'radar-lines-g')
            .selectAll('line')
            .data(new Array(nSector).fill(0))
            .enter().append('line')
            .attr('x1', 0)
            .attr('y1', 0)
            .attr('x2', (d, i) => radarRadius * (Math.sin(i * onePiece)))
            .attr('y2', (d, i) => radarRadius * (Math.cos(i * onePiece)))
            .attr('stroke-width', 0.5)
            .style("stroke-dasharray", ("1, 1"))
            .attr('stroke', '#dce7f2')
          // 雷达图文字
          const attrText = ['AL', "PV", "NS", "RL", "SC", "DR"]
          g.append('g')
            .attr('class', 'radar-text-g')
            .selectAll('text')
            .data(new Array(nSector).fill(0))
            .enter().append('text')
            .text((d, i) => attrText[i]) // 放属性缩写的地方
            .attr('x', 0.5)
            .attr('y', -9.2)
            .attr('font-size', 4)
            .attr('fill', '#dce7f2')
            .attr('transform', (d, i) => `rotate(${i * onePiece * 180 / Math.PI})`)
          // 雷达图区域
          let areaStr2 = ''
          values.forEach((v, i) => {
            const xLocal = radarRadius * v * (Math.sin(i * onePiece))
            const yLocal = radarRadius * v * (Math.cos(i * onePiece))
            areaStr2 += xLocal + ',' + yLocal + ' ';
          })
          const radar = g.append('g')
            .attr('class', 'radar')
            .append('polygon')
            .attr('points', (d) => {
              if (d.id && Projection.clusterAttrs) {
                const values = Projection.clusterAttrs[d.id]
                let areaStr2 = ''
                values.forEach((v, i) => {
                  const xLocal = radarRadius * v * (Math.sin(i * onePiece))
                  const yLocal = radarRadius * v * (Math.cos(i * onePiece))
                  areaStr2 += xLocal + ',' + yLocal + ' ';
                })
                return areaStr2
              }
              return areaStr2
            })
            .attr('stroke', '#c8d2dc')
            .attr('stroke-width', 0.5)
            .attr('fill', 'rgba(146,197,222,0.5)')
        })

      this.dots = dots

      this._myRender()
    }
  }

  public async _myRender () {
    const [d3Projection, zoom] = await this.getD3()
    this.dots
      .attr('transform', (d) => {
        const x = (d3Projection(d.geometry.coordinates) as [number, number])[0]
        const y = (d3Projection(d.geometry.coordinates) as [number, number])[1]
        const referenceS = Math.pow(2, 20 - zoom)
        const s = Math.pow(2, 9)
        return `translate(${x},${y}) scale(${1.5 * s / referenceS})`
      })
  }

  @Watch('stationExtends', {immediate: true, deep: true})
  public computeVoronoiDigrame (val: number[][]) {
    if (val) {
      Projection.computeVoronoiForStations({stations: _.values(Exploration.indexedStations), extent: val })
    }
  }

  public get targetRoute() {
    return CandidatesList.changeTargetWatcher
  }

  public get inOutStations() {
    return Exploration.matrixInOutWatcher
  }

  @Watch('inOutStations')
  public async drawInOutStations () {
    console.log('[Map] Redrawing In-and-Out Stations...')
    const map = await this.map.get()
    if (Exploration.inOutGeoJSON) {
      const oldSource = map.getSource('inOutStations')
      if (oldSource) {
        map.removeLayer('inOutStations')
        map.removeSource('inOutStations')
      }
      map.addSource('inOutStations', {
        type: 'geojson',
        data: Exploration.inOutGeoJSON
      })
      map.addLayer({
        id: 'inOutStations',
        type: 'circle',
        source: 'inOutStations',
        paint: {
          'circle-radius': 5,
          'circle-stroke-width': 1.5,
          'circle-stroke-color': '#7585d7',
          'circle-color': ['get', 'color']
        }
      })
    }
  }

  @Watch('targetRoute', {immediate: true, deep: true})
  public async drawTargetRoute() {
    console.log('[Map] Redrawing Target Route...')
    const map = await this.map.get()
    if (CandidatesList.targetRoute && CandidatesList.targetRoute.stopsGeoJSON()) {
      const data = {
        type: 'FeatureCollection',
        features: CandidatesList.targetRoute.stopsGeoJSON()
      }
      const oldRouteSource = map.getSource('targetRoute')
      if (oldRouteSource) {
        oldRouteSource.setData(data)
        if (map.getLayer('conflictGraphLink')) {
          map.moveLayer('targetRoute', 'conflictGraphLink')
        }
        return
      }
      // @ts-ignore
      map.addSource('targetRoute', {
        type: 'geojson',
        data: data
      })
      map.addLayer({
        id: 'targetRoute',
        type: 'circle',
        source: 'targetRoute',
        paint: {
          'circle-radius': 7,
          'circle-stroke-width': 3,
          'circle-stroke-color': '#67a9cf',
          'circle-color': 'white'
        }
      })
    } else {
      const oldRouteSource = map.getSource('targetRoute')
      if (oldRouteSource) {
        map.removeLayer('targetRoute')
        map.removeSource('targetRoute')
      }
    }
  }

  @Watch('matrixRoute', {immediate: true, deep: true})
  public async drawMatrixRoutes () {
    if (!this.displayMatrixRoute) {
      return
    }
    console.log('[Map] Redrawing Voronoi Polygons...')
    const map = await this.map.get()

    const oldRouteSource = map.getSource('matrixRoute')
    if (oldRouteSource) {
      map.removeLayer('matrixRoute')
      map.removeSource('matrixRoute')
    }
    const oldStationSource = map.getSource('matrixStations')
    if (oldStationSource) {
      map.removeLayer('matrixStations')
      map.removeSource('matrixStations')
    }

    if (this.matrixRoute) {
      const {routes, stations} = this.matrixRoute
      map.addSource('matrixRoute', {
        type: 'geojson',
        data: routes
      })
      map.addSource('matrixStations', {
        type: 'geojson',
        data: stations
      })
      map.addLayer({
        id: 'matrixStations',
        type: 'circle',
        source: 'matrixStations',
        paint: {
          'circle-color': 'white',
          'circle-opacity': 1,
          'circle-radius': 5,
          'circle-stroke-width': [
            'case',
            ['boolean', ['feature-state', 'hover'], false],
            3,
            1
          ],
          'circle-stroke-color': '#7585d7'
        }
      })
      map.on('click', 'matrixStations', (e) => {
        if (e.features && e.features[0].id) {
          const succ = CandidatesList.targetRoute?.setSelectedStation(parseInt(e.features[0].id.toString()))
          if (succ) {
            CandidatesList.incChangeTargetWatcher()
          }
        }
      })
      map.addLayer({
        id: 'matrixRoute',
        type: 'line',
        source: 'matrixRoute',
        paint: {
          'line-color': [
            'case',
            ['get', 'evaluate'],
            '#888',
            '#7585d7'
          ],
          'line-width': 3,
          'line-opacity': [
            'case',
            ['get', 'focus'],
            1,
            0.3
          ]
        }
      }, 'matrixStations')
      if (map.getLayer('conflictGraphLink')) {
        map.moveLayer('matrixRoute', 'conflictGraphLink')
        map.moveLayer('matrixStations', 'conflictGraphLink')
      }
    }
  }

  public get displayMatrixRoute () {
    return Exploration.displayMatrixRoute
  }

  @Watch('displayMatrixRoute', {immediate: true, deep: true})
  public async toggleDisplayMatrix (val: boolean) {
    const map = await this.map.get()
    if (val) {
      const routeLayer = map.getLayer('matrixRoute')
      const stationLayer = map.getLayer('matrixStations')
      const targetLayer = map.getLayer('targetRoute')
      if (routeLayer) {
        map.setLayoutProperty('matrixRoute', 'visibility', 'visible')
      }
      if (stationLayer) {
        map.setLayoutProperty('matrixStations', 'visibility', 'visible')
      }
      if (targetLayer) {
        map.setLayoutProperty('targetRoute', 'visibility', 'visible')
      }
    } else {
      const routeLayer = map.getLayer('matrixRoute')
      const stationLayer = map.getLayer('matrixStations')
      const targetLayer = map.getLayer('targetRoute')
      if (routeLayer) {
        map.setLayoutProperty('matrixRoute', 'visibility', 'none')
      }
      if (stationLayer) {
        map.setLayoutProperty('matrixStations', 'visibility', 'none')
      }
      if (targetLayer) {
        map.setLayoutProperty('targetRoute', 'visibility', 'none')
      }
    }
  }

  @Watch('matrixStations', {immediate: true, deep: true})
  public async matrixStationsHighlight() {
    const map = await this.map.get()
    if (Exploration.matrixRouteSelected) {
      this.matrixStations.forEach(s => {
        map.setFeatureState(
          {source: 'matrixStations', id: s.toString()},
          {hover: true}
        );
      })
    }
  }

  @Watch('voronoiPolygons', {immediate: true, deep: true})
  public async drawVoronoiPolygons () {
    console.log('[Map] Redrawing Voronoi Polygons...')
    const map = await this.map.get()

    _.each(this.voronoiPolygons, (cluster, id) => {
      const oldSource = map.getSource('cluster' + id)
      if (oldSource) {
        map.removeLayer('cluster-polygon' + id)
        // map.removeLayer('cluster-line' + id)
        map.removeSource('cluster' + id)
      }
    })

    if (this.voronoiPolygons) {
      _.each(this.voronoiPolygons, (cluster, id) => {
        map.addSource('cluster' + id, {
          type: 'geojson',
          data: cluster
        })
        map.addLayer({
          id: 'cluster-polygon' + id,
          type: 'fill',
          source: 'cluster' + id,
          paint: {
            'fill-color': '#8ca6ba',
            'fill-opacity': 0
          }
        })
      })
    }
  }

  @Watch('selectedRoutesGeoJSON', {immediate: true, deep: true})
  public async drawSelectedRoutes () {
    console.log('[Map] Redrawing selected routes...')
    const map = await this.map.get()

    const oldSource = map.getSource('selected-routes')
    if (oldSource) {
      map.removeLayer('selected-routes')
      map.removeSource('selected-routes')
    }

    if (this.selectedRoutesGeoJSON) {
      if (this.selectedRoutesGeoJSON.hidden) {
        return
      }
      const geojson = this.selectedRoutesGeoJSON.GeoJSON
      map.addSource('selected-routes', {
        type: 'geojson',
        data: geojson
      })
      map.addLayer({
        id: 'selected-routes',
        type: 'line',
        source: 'selected-routes',
        paint: {
          'line-color': [
            'case',
            ['boolean', ['feature-state', 'hover'], false],
            '#060506',
            '#7585d7'
          ],
          'line-opacity': [
            'case',
            ['boolean', ['feature-state', 'hover'], false],
            1,
            0.2
          ],
          'line-width': 1.5
        }
      })
    }
  }

  // @Watch('highlightRoutesTimes', {immediate: true})
  // public async drawHighLightRoutes () {
  //   console.log('[Map] Redrawing highlight routes...')
  //   const map = await this.map.get()
  //
  //   const allRoutes = map.getSource('routes')
  //   if (allRoutes) {
  //     map.removeLayer('all-routes')
  //     map.removeSource('routes')
  //   }
  //
  //   const oldSource = map.getSource('highlight-routes')
  //   if (oldSource) {
  //     map.removeLayer('highlight-routes')
  //     map.removeSource('highlight-routes')
  //   }
  //
  //   if (this.routeGeoJSON) {
  //     map.addSource('highlight-routes', {
  //       type: 'geojson',
  //       data: this.highlightRoutes
  //     })
  //     map.addLayer({
  //       id: 'highlight-routes',
  //       type: 'line',
  //       source: 'highlight-routes',
  //       paint: {
  //         'line-color': '#2c7fb8',
  //         'line-opacity': 1,
  //         'line-width': 1.5
  //       }
  //     })
  //   }
  // }

  @Watch('selectedStations', {immediate: true, deep: true})
  public async drawSelectedStations () {
    console.log('[Map] Redrawing selected stations...')
    const map = await this.map.get()

    const oldSource = map.getSource('selected-stations')
    if (oldSource) {
      map.removeLayer('selected-stations')
      map.removeSource('selected-stations')
    }

    if (this.stationGeoJSON) {
      map.addSource('selected-stations', {
        type: 'geojson',
        data: this.selectedStations
      })
      map.addLayer({
        id: 'selected-stations',
        type: 'circle',
        source: 'selected-stations',
        paint: {
          'circle-radius': 3.5,
          'circle-color': '#8c96c6'
        }
      })
    }
  }

  public async drawStations () {
    console.log('[Map] Redrawing stations...')
    const map = await this.map.get()

    const oldSource = map.getSource('stations')
    if (oldSource) {
      map.removeLayer('all-stations')
      map.removeLayer('stations-heat')
      map.removeSource('stations')
    }

    if (this.stationGeoJSON) {
      map.addSource('stations', {
        type: 'geojson',
        data: this.stationGeoJSON
      })
      map.addLayer({
        id: 'stations-heat',
        type: 'heatmap',
        source: 'stations',
        maxzoom: 14,
        paint: {
          'heatmap-weight': [
            'interpolate',
            ['linear'],
            ['get', 'std_flow'],
            0.1,
            0,
            7,
            1
          ],
          'heatmap-intensity': [
            'interpolate',
            ['linear'],
            ['zoom'],
            0,
            1,
            13,
            3
          ],
          'heatmap-color': [
            'interpolate',
            ['linear'],
            ['heatmap-density'],
            0.1,
            'rgba(33,102,172,0)',
            0.25,
            '#4575b4',
            0.56,
            '#91bfdb',
            0.72,
            '#e0f3f8',
            0.86,
            '#ffffbf',
            0.92,
            '#fee090',
            0.96,
            '#fc8d59',
            1.0,
            '#d73027'
          ],
          'heatmap-radius': [
            'interpolate',
            ['linear'],
            ['zoom'],
            5,
            5,
            15,
            50
          ],
          'heatmap-opacity': [
            'interpolate',
            ['linear'],
            ['zoom'],
            10,
            0.5,
            11,
            0
          ]
        }
      })
    }
  }

  @Watch('clusterLinks', { immediate: true })
  public async drawClusterLinks () {
    console.log('[Map] Redrawing cluster Links...')
    const map = await this.map.get()

    const oldSource = map.getSource('cluster-links')
    if (oldSource) {
      map.removeLayer('cluster-links')
      map.removeSource('cluster-links')
    }

    if (this.clusterCenters) {
      map.addSource('cluster-links', {
        type: 'geojson',
        data: this.clusterLinks
      })
      map.addLayer({
        id: 'cluster-links',
        type: 'line',
        source: 'cluster-links',
        paint: {
          'line-width': ['get', 'value'],
          'line-color': '#b6d4e6',
          'line-opacity': 0.5
          // [
          //   'interpolate',
          //   ['linear'],
          //   ['zoom'],
          //   9,
          //   [
          //     'case',
          //     ['boolean', ['feature-state', 'hover'], false],
          //     1,
          //     0.5
          //   ],
          //   15,
          //   0
          // ]
        }
      })
      // map.on('mouseenter', 'cluster-links', (e) => {
      //   if (e.features && e.features[0].properties && e.features[0].id) {
      //     const routes = e.features[0].properties.routes
      //     this.highlightClusterLinkID = e.features[0].id.toString()
      //     map.setFeatureState(
      //       { source: 'cluster-links', id: this.highlightClusterLinkID },
      //       { hover: true }
      //     );
      //     Exploration.setHighlightRoutesGeoJSON({routes: JSON.parse(routes), selected: false})
      //   }
      // })
      // map.on('mouseleave', 'cluster-links', () => {
      //   if (this.highlightClusterLinkID) {
      //     if (Projection.selectedLinks && Projection.selectedLinks.has(parseInt(this.highlightClusterLinkID))) {
      //       return
      //     }
      //     Exploration.clearHighlightRoutes()
      //     map.setFeatureState(
      //       {source: 'cluster-links', id: this.highlightClusterLinkID},
      //       {hover: false}
      //     );
      //     this.highlightClusterLinkID = null
      //   }
      // })
      // map.on('click', 'cluster-links', (e) => {
      //   if (this.linkClickTimes === 0) {
      //     this.linkClickTimes++
      //     return;
      //   }
      //   this.linkClickTimes = 0
      //   // toggle link selection
      //   if (e.features && e.features[0].properties && e.features[0].id) {
      //     Projection.toggleLink(parseInt(e.features[0].properties.key))
      //     if (Projection.selectedLinks) {
      //       if (Projection.selectedLinks.has(e.features[0].properties.key)) {
      //         map.setFeatureState(
      //           { source: 'cluster-links', id: e.features[0].id.toString() },
      //           { hover: false }
      //         );
      //       } else {
      //         map.setFeatureState(
      //           { source: 'cluster-links', id: e.features[0].id.toString() },
      //           { hover: true }
      //         );
      //       }
      //     }
      //   }
      // })
    }
  }

  // @Watch('clusterCenters', { immediate: true, deep: true })
  // public async drawClusterCenters () {
  //   if (this.clusterCenters.features.length === 0) {
  //     Projection.setCenterGeoJSON()
  //     return
  //   }
  //   console.log('[Map] Redrawing cluster Centers...')
  //   const map = await this.map.get()

  //   const oldSource = map.getSource('cluster-centers')
  //   if (oldSource) {
  //     map.removeLayer('cluster-centers')
  //     map.removeSource('cluster-centers')
  //   }

  //   if (this.clusterCenters) {
  //     console.log(this.clusterCenters)
  //     map.addSource('cluster-centers', {
  //       type: 'geojson',
  //       data: this.clusterCenters
  //     })
  //     map.addLayer({
  //       id: 'cluster-centers',
  //       type: 'circle',
  //       source: 'cluster-centers',
  //       paint: {
  //         'circle-radius': ['get', 'value'],
  //         'circle-color': [
  //           'case',
  //           ['boolean', ['feature-state', 'hover'], false],
  //           '#8856a7', '#3182bd'],
  //         'circle-stroke-opacity': 1,
  //         'circle-stroke-color': '#deebf7',
  //         'circle-stroke-width': 2,
  //         'circle-opacity': [
  //           'interpolate',
  //           ['linear'],
  //           ['zoom'],
  //           11,
  //           1,
  //           15,
  //           0
  //         ]
  //       }
  //     })
  //     map.on('mouseenter', 'cluster-centers', (e) => {
  //       if (e.features && e.features[0].id) {
  //         map.setPaintProperty(
  //           'cluster-polygon' + e.features[0].id,
  //           'fill-opacity',
  //           ['interpolate',
  //             ['linear'],
  //             ['zoom'],
  //             9,
  //             0,
  //             15,
  //             ['get', 'value']])
  //         this.highlightClusterID = e.features[0].id
  //       }
  //     })
  //     map.on('mouseleave', 'cluster-centers', (e) => {
  //       if (this.highlightClusterID > 0) {
  //         map.setPaintProperty('cluster-polygon' + this.highlightClusterID, 'fill-opacity', 0)
  //         this.highlightClusterID = -1
  //       }
  //     })
  //     map.on('click', 'cluster-centers', (e) => {
  //       if (e.features && e.features[0].properties && e.features[0].id) {
  //         if (Projection.selectedCenters) {
  //           if (Projection.selectedCenters.has(parseInt(e.features[0].properties.key))) {
  //             console.log('style concell on center')
  //             map.setFeatureState(
  //               { source: 'cluster-centers', id: e.features[0].id.toString() },
  //               { hover: false }
  //             );
  //           } else {
  //             map.setFeatureState(
  //               { source: 'cluster-centers', id: e.features[0].id.toString() },
  //               { hover: true }
  //             );
  //           }
  //         }
  //         Projection.toggleCluster(parseInt(e.features[0].properties.key))
  //       }
  //     })
  //   }
  // }

  @Watch('contour', {immediate: true, deep: true})
  public async drawContour() {
    console.log('[Map] Redrawing contour...', this.contour)
    const map = await this.map.get()

    const oldSource = map.getSource('con')
    if (oldSource) {
      map.removeLayer('con')
      map.removeSource('con')
    }
    if (this.contour) {
      const {contour, hidden} = this.contour
      if (hidden) {
        return
      }
      map.addSource('con', {
        type: 'geojson',
        data: contour
      })
      map.addLayer({
        id: 'con',
        type: 'line',
        source: 'con',
        paint: {
          'line-color': 'gray',
          'line-opacity': 0.3,
          'line-width': 1
        }
      })
    }
  }

  public async drawRoutes () {
    console.log('[Map] Redrawing routes...')
    const map = await this.map.get()

    const oldSource = map.getSource('routes')
    if (oldSource) {
      map.removeLayer('all-routes')
      map.removeSource('routes')
    }

    if (this.routeGeoJSON) {
      map.addSource('routes', {
        type: 'geojson',
        data: this.routeGeoJSON
      })
      map.addLayer({
        id: 'all-routes',
        type: 'line',
        source: 'routes',
        paint: {
          'line-color': '#7585d7',
          'line-opacity': 0.8,
          'line-width': 15
        }
      })
    }
  }

  @Watch('stationGeoJSON')
  private async _onStationGeoJSONChanged () {
    await this.drawStations()
  }

  private async _onMapMouseMove (e: mapboxgl.MapMouseEvent) {
    this.hoveredRouteFeatures = _((await this.map.get()).queryRenderedFeatures(e.point))
      .filter({ source: 'routes' })
      .uniqBy('properties.id')
      .valueOf()
  }

  private async _onMapClick () {
    if (this.tooltip.visible) {
      this.tooltipPinned = !this.tooltipPinned
    }
  }
}
</script>

<style lang="scss">
#map {
  position: absolute;
  width: 100%;
  height: 100%;

  #mapbox {
    width: 100%;
    height: 100%;

    canvas {
      cursor: crosshair;
    }

    svg {
      position: absolute;
      width: 100%;
      height: 100%;
      cursor: crosshair;
    }
  }
}
</style>
