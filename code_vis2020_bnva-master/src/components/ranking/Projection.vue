<template>
  <div id="routes-projection">
    <svg id="projection-svg-panel"
         pointer-events="all"
    >
      <rect
        visibility="hidden"
        id="selection-rect"
        fill="gray"
        opacity="0.1"
        stroke-width="2"
        stroke-dasharray="10"
        stroke-opacity="0.7"
        stroke="black"
      ></rect>
      <g clip-path="url(#clip)" class="link-group" transform="translate(0, 0)">
        <line
          v-for="link in stationLinks"
          :key="link.station1 + link.station2"
          :x1="xScale(indexedCenters[link.station1][0])"
          :y1="yScale(indexedCenters[link.station1][1])"
          :x2="xScale(indexedCenters[link.station2][0])"
          :y2="yScale(indexedCenters[link.station2][1])"
          :stroke-width="linkScale(link.routes.length)"
          stroke="#ef8a62"
          @mouseover="mouseOverLink(link.routes)"
          @mouseleave="mouseLeaveLink"
          @click="mouseClickLink(link.routes)"
        ></line>
      </g>
      <g clip-path="url(#clip)" class="points-group" transform="translate(0, 0)">
        <circle
          v-for="point in stationCenters"
          :key="point.id"
          :cx="xScale(point.px)"
          :cy="yScale(point.py)"
          :r="sizeScale(stationClusters[point.id].length)"
          stroke="#2166ac"
          stroke-width="3"
          fill="#67a9cf"
          :fill-opacity="opacityScale(point.checkin + point.checkout)"
          @mouseover="mouseOverCircle(parseInt(point.id))"
          @mouseleave="mouseLeaveCircle"
        ></circle>
      </g>
    </svg>
  </div>
</template>

<script lang="ts">
import {Component, Vue, Watch} from 'vue-property-decorator'
import * as d3 from 'd3'
import $ from 'jquery'
import _ from 'lodash'
import Projection from '@/store/modules/Projection'
import Exploration from '@/store/modules/Exploration'
import CandidateList from '@/store/modules/CandidatesList';
import { ProjectionPoint } from '@/utils/Types'

@Component
export default class ProjectionView extends Vue {
  public svgWidth = 0
  public svgHeight = 0
  public svg = d3.select('svg')
  public xScale = d3.scaleLinear()
  public yScale = d3.scaleLinear()
  public opacityScale = d3.scaleLinear()
  public sizeScale = d3.scaleLinear()
  public linkScale = d3.scaleLinear()
  public xDomain = [Infinity, -Infinity]
  public yDomain = [Infinity, -Infinity]
  public selectPoints : number[] = []
  public usingAllData = true
  public click = false
  public startPos : number[] = [0, 0]
  public endPos : number[] = [0, 0]

  public NODE_SPACE = 5
  public MOVE_DIST = 0.03

  mouseOverLink (routes: number[]) {
    if (this.stationLinks) {
      Exploration.setHighlightRoutesGeoJSON(routes)
    }
  }

  mouseLeaveLink () {
    Exploration.clearHighlightRoutes()
  }

  mouseOverCircle (clusterID: number) {
    if (this.stationClusters) {
      Exploration.setSelectedStationsGeoJSON(this.stationClusters[clusterID])
    }
  }

  mouseLeaveCircle () {
    Exploration.clearSelectedStation()
  }

  mouseClickLink (routes: number[]) {
    const rSet = new Set(routes)
    CandidateList.clearCandidates()
    CandidateList.addCandidates({ indexedRoutes: Exploration.indexedRoutes, newCandidates: Array.from(rSet) })
  }

  onClick (e : { offsetX: number, offsetY: number }) {
    console.log('click: ' + this.click)
    if (this.click) {
      this.click = false
      this.endPos = [e.offsetX, e.offsetY]
      this.onSelectionChange()
      d3.select('#selection-rect')
        .attr('visibility', 'hidden')
      CandidateList.clearCandidates()
      CandidateList.addCandidates({ indexedRoutes: Exploration.indexedRoutes, newCandidates: this.selectPoints})
    } else {
      this.click = true
      this.selectPoints = []
      const x = e.offsetX
      const y = e.offsetY
      this.startPos = [x, y]
      this.endPos = [x, y]
      d3.select('#selection-rect')
        .attr('visibility', 'visible')
      this.setSelectionRect()
    }
  }

  onMouseMove(e: { offsetX: number, offsetY: number }) {
    if (this.click) {
      this.endPos = [e.offsetX, e.offsetY]
      this.setSelectionRect()
    }
  }

  setSelectionRect() {
    d3.select('#selection-rect')
      .attr('x', this.x())
      .attr('y', this.y())
      .attr('width', this.width())
      .attr('height', this.height())
  }

  getCircleStroke (x : number, y: number) {
    if (this.click) {
      return 0
    }
    const x2 = this.x() + this.width()
    const y2 = this.y() + this.height()
    if (x < x2 && x > this.x() && y < y2 && y > this.y()) {
      return 2
    }
    return 0
  }

  x() { return Math.min(this.startPos[0], this.endPos[0]) }
  y() { return Math.min(this.startPos[1], this.endPos[1]) }
  width() { return Math.abs(this.startPos[0] - this.endPos[0]) }
  height() { return Math.abs(this.startPos[1] - this.endPos[1]) }

  get routeTransferTimes () {
    const data : { [id: number]: number } = {}
    _.each(
      Exploration.indexedRouteBranches,
      route => {
        data[route.route_id] = _.sumBy(route.branches, branch => branch.records.length)
      }
    )
    return data
  }

  get routeData () {
    if (!this.usingAllData) {
      return []
    }
    const data : ProjectionPoint[] = []
    _.each(
      Exploration.indexedRoutes,
      (route) => {
        data.push({
          id: route.id,
          checkin: route.checkin,
          checkout: route.checkout,
          px: parseFloat(route.projection[0]),
          py: parseFloat(route.projection[1])
        })
      }
    )
    return data
  }

  get indexedCenters () {
    return Projection.centers
  }

  get stationCenters () {
    const centers = Projection.centers
    const data : { id: string, px: number, py: number }[] = []
    _.each(
      centers,
      (v, k) => {
        data.push({
          id: k,
          px: v[0],
          py: v[1]
        })
      }
    )
    return data
  }

  get stationLinks () {
    const links: {station1: string, station2: string, routes: number[]}[] = []
    _.each(
      Projection.links,
      (cluster, ci) => {
        _.each(
          cluster,
          (link, cj) => {
            if (link.length > 0) {
              links.push({
                station1: ci,
                station2: cj,
                routes: link
              })
            }
          }
        )
      }
    )
    return links
  }

  get indexedStationLinks () {
    return Projection.links
  }

  get stationClusters () {
    return Projection.clusters
  }

  // move centers to avoid occlusion
  centersRepulsion () {
    if (!this.stationClusters) {
      return
    }
    const centers = _.cloneDeep(this.stationCenters)
    let flag = true
    let i = 0
    while (flag && i < 1000) {
      flag = false
      _.each(
        centers,
        c1 => {
          _.each(
            centers,
            c2 => {
              if (c1.id !== c2.id) {
                const s1 = this.sizeScale(this.stationClusters[c1.id].length)
                const s2 = this.sizeScale(this.stationClusters[c1.id].length)
                const dist = Math.sqrt(Math.pow((c1.px - c2.px), 2) + Math.pow((c1.py - c2.py), 2))
                if (s1 + s2 + this.NODE_SPACE > dist) {
                  flag = true
                  const { dx, dy } = this._unitDist(c1.px, c1.py, c2.px, c2.py)
                  c2.px += dx * this.MOVE_DIST
                  c2.py += dy * this.MOVE_DIST
                  c1.px -= dx * this.MOVE_DIST
                  c1.py -= dy * this.MOVE_DIST
                }
              }
            }
          )
        }
      )
      i++
    }
    if (i > 1) {
      Projection.setCenters(centers)
    }
  }

  _unitDist (x1: number, y1: number, x2: number, y2: number) {
    const dx = x2 - x1
    const dy = y2 - y1
    const dist = this._dist(x1, y1, x2, y2)
    return {dx: dx / dist, dy: dy / dist}
  }

  _dist (x1: number, y1: number, x2: number, y2: number) {
    return Math.sqrt(Math.pow((x1 - x2), 2) + Math.pow((y1 - y2), 2))
  }

  @Watch('stationLinks', {immediate: true})
  onStationLinksChanged(val: {station1: string, station2: string, routes: number[]}[]) {
    const linkDomain = [Infinity, -Infinity]
    _.each(
      this.stationLinks,
      v => {
        if (v.routes.length > linkDomain[1]) {
          linkDomain[1] = v.routes.length
        }
        if (v.routes.length < linkDomain[0]) {
          linkDomain[0] = v.routes.length
        }
      }
    )
    this.linkScale.domain(linkDomain).range([1, 10])
  }

  @Watch('stationClusters', { immediate: true })
  onStationClustersChanged(val: { [clusterID: number] : number[] } ) {
    const sizeDomain = [Infinity, -Infinity]
    _.each(
      val,
      v => {
        if (v.length > sizeDomain[1]) {
          sizeDomain[1] = v.length
        }
        if (v.length < sizeDomain[0]) {
          sizeDomain[0] = v.length
        }
        this.sizeScale.domain(sizeDomain).range([7, 15])
      }
    )
  }

  // @Watch('routeTransferTimes', { immediate: true })
  // onRouteTransferTimesChanged(val: { [id: number]: number }) {
  //   const sizeDomain = [Infinity, -Infinity]
  //   _.each(
  //     val,
  //     v => {
  //       if (v > sizeDomain[1]) {
  //         sizeDomain[1] = v
  //       }
  //       if (v < sizeDomain[0]) {
  //         sizeDomain[0] = v
  //       }
  //       this.sizeScale.domain(sizeDomain).range([2, 4])
  //     }
  //   )
  // }

  onSelectionChange() {
    const x2 = this.x() + this.width()
    const y2 = this.y() + this.height()
    _.each(
      this.routeData,
      route => {
        const x = this.xScale(route.px)
        const y = this.yScale(route.py)
        if (x < x2 && x > this.x() && y < y2 && y > this.y()) {
          this.selectPoints.push(route.id)
        }
      }
    )
  }

  @Watch('stationCenters', { immediate: true })
  onClusterDataChanged(val: { id: string, px: number, py: number }[]) {
    this.xDomain = [Infinity, -Infinity]
    this.yDomain = [Infinity, -Infinity]
    _.each(
      val,
      point => {
        if (point.px > this.xDomain[1]) {
          this.xDomain[1] = point.px
        }
        if (point.px < this.xDomain[0]) {
          this.xDomain[0] = point.px
        }
        if (point.py > this.yDomain[1]) {
          this.yDomain[1] = point.py
        }
        if (point.py < this.yDomain[0]) {
          this.yDomain[0] = point.py
        }
      }
    )
    const xDelta = (this.xDomain[1] - this.xDomain[0]) / 10
    const yDelta = (this.xDomain[1] - this.xDomain[0]) / 10
    this.xDomain[0] -= xDelta
    this.xDomain[1] += xDelta
    this.yDomain[0] -= yDelta
    this.yDomain[1] += yDelta
    this.xScale = d3.scaleLinear().domain(this.xDomain).range([0, this.svgWidth])
    this.yScale = d3.scaleLinear().domain(this.yDomain).range([this.svgHeight, 0])
    this.centersRepulsion()
  }

  // @Watch('routeData', { immediate: true })
  // onRouteDataChanged(val: ProjectionPoint[], oldVal: ProjectionPoint[]) {
  //   const opacityDomain = [Infinity, -Infinity]
  //   _.each(
  //     val,
  //     point => {
  //       if (point.px > this.xDomain[1]) {
  //         this.xDomain[1] = point.px
  //       }
  //       if (point.px < this.xDomain[0]) {
  //         this.xDomain[0] = point.px
  //       }
  //       if (point.py > this.yDomain[1]) {
  //         this.yDomain[1] = point.py
  //       }
  //       if (point.py < this.yDomain[0]) {
  //         this.yDomain[0] = point.py
  //       }
  //       const flow = point.checkout + point.checkout
  //       if (flow > opacityDomain[1]) {
  //         opacityDomain[1] = flow
  //       }
  //       if (flow < opacityDomain[0]) {
  //         opacityDomain[0] = flow
  //       }
  //     }
  //   )
  //   this.xScale = d3.scaleLinear().domain(this.xDomain).range([0, this.svgWidth])
  //   this.yScale = d3.scaleLinear().domain(this.yDomain).range([0, this.svgHeight])
  //   this.opacityScale = d3.scaleLinear().domain(opacityDomain).range([0.3, 0.9])
  //   // draw data points
  // }

  mounted (): void {
    this.svg = d3.select('svg')
    const svg = $('#projection-svg-panel')
    if (svg.height() && svg.width()) {
      this.svgHeight = svg.height()!
      this.svgWidth = svg.width()!
    }

    // create a clipping region
    this.svg.append('defs').append('clipPath')
      .attr('id', 'clip')
      .append('rect')
      .attr('width', this.svgWidth)
      .attr('height', this.svgHeight)
  }
}
</script>

<style scoped>
#projection-svg-panel {
  display: flex;
  flex: 0 0;
  position: relative;
  width: 100%;
  height: 100%;
  background-color: white;
  border: 1px solid #eee;
  border-radius: 3px;
}
</style>
