<template>
  <div class="column-headers"
    :class="{filtering: showFilter}"
    :style="{'flex-basis': px(showFilter? 120 : 50)}"
    @mousemove="mouseDidMoveOnColumnHeaders"
    @mouseup="mouseDidReleaseColumnHeaders">
    <div v-show="routeOptimized && !searching" class="progressive-svg">
      <div class="route-list-length">{{routeListLengthMax}}</div>
    </div>
    <transition-group :name="headerResizing ? 'flip-list-non-animated' : 'flip-list'"
      tag="div" class="flip-div">
      <div class="header" v-for="attr in attributes"
        :key="attr.key"
        :class="{
          grouped: attr.group
        }"
        :style="{
          'flex-basis': px(attr.width)
        }"
        @dblclick="(e) => mouseDBLClickOnAttribute(e, attr)"
        @contextmenu="(e) => { mouseRightClickOnAttribute(e, attr) }"
      >
        <div class="name-wrap">
          <font-awesome-icon
              @click="onReorderHeaders(attr)"
              icon="grip-vertical"
              class="name-icon" />
          <div class="name">
            {{ attr.name }}
          </div></div>
          <font-awesome-icon
            :icon="(attr.sorted && attr.sorted === 'up') ? `sort-amount-up-alt` : `sort-amount-up`"
            :class="['sorting', { 'active-sorting': attr.sorted }]" />

        <div class="filter" v-if="showFilter && attr.attrDistribution"
          @mousedown.left="(e) => onStartFilter(e, attr)">
          <svg width="178" height="70" style="position: absolute; transform-origin: 0 0;"
            :transform="`translate(7, 0) scale(${attr.width / 180} 1)`">
            <defs>
              <clipPath :id="`clipPath_${attr.key}`">
                  <rect :x="attr.attrDistribution.validRange[0] * 178"
                    y="0" :width="(attr.attrDistribution.validRange[1] - attr.attrDistribution.validRange[0]) * 178" height="70"/>
              </clipPath>
            </defs>
            <path :d="attr.attrDistribution.path" fill="#dae2ec" />
            <path :d="attr.attrDistribution.path" fill="#b6d4e6" :clip-path="`url(#clipPath_${attr.key})`"/>
          </svg>
          <div class="masking-cont" :ref="`maskings_${attr.key}`"
            @dblclick="(e) => onResetFilter(e, attr)">
            <div class="masking-left" :style="{ width: `calc(${attr.attrDistribution.validRange[0] * 100}%)` }" >
              <div class="value">{{getLeftValue(attr)}}</div>
            </div>
            <div class="masking-right"
              :style="{
                width: 0,
                left: `calc(${attr.attrDistribution.validRange[1] * 100}%)`
              }">
              <div class="value">{{getRightValue(attr)}}</div>
            </div>
          </div>
          <!-- <div class="bar-wrapper">
            <div class="bar-cont">
              <div class="bar" v-for="(opt, i) in attr.filter.options"
                  :key="i"
                  :style="{
                  left: prec(opt.styles.left),
                  width: prec(opt.styles.width),
                  height: prec(opt.styles.height)
                }"
              >
                <div class="shadow"
                    :style="{
                    height: prec(opt.styles.shadow)
                  }"></div>
              </div>
              <div class="range left"
                  :class="{active: ranger && ranger.target === attr && ranger.type === 'left'}"
                  :style="{ left: prec(attr.filter.range[0]) }"
                  @mousedown="(e) => mousePressOnRanger(e, attr, 'left')"></div>
              <div class="range right"
                  :class="{active: ranger && ranger.target === attr && ranger.type === 'right'}"
                  :style="{ left: prec(attr.filter.range[1]) }"
                  @mousedown="(e) => mousePressOnRanger(e, attr, 'right')"></div>
              <div class="range-axis left"
                  :style="{
                  width: prec(attr.filter.range[0])
                }"></div>
              <div class="range-axis right"
                  :style="{
                  width: prec(1 - attr.filter.range[1])
                }"></div>
            </div>
          </div> -->
        </div>
        <div class="handle"
          @mousedown="onStartHeaderResize($event, attr)" />
      </div>
    </transition-group>
  </div>
</template>

<script lang="ts">
import { Component, Vue, Watch, Ref } from 'vue-property-decorator'
import CandidatesList from '@/store/modules/CandidatesList'
import { prec, px } from '@/utils/Formatter'
import { Attribute } from '@/utils/Attribute'
import _ from 'lodash'
import $ from 'jquery'
import * as d3 from 'd3'
import Manipulation from '../../../store/modules/Manipulation'

@Component
export default class ColumnHeaders extends Vue {
  public px = px
  public prec = prec
  public ranger: { type: string, cursorX: number, target: Attribute, width: number } | null = null

  private pageXWhenStartResize = 0
  private headerWidthIntial = 0
  private headerResized: Attribute | null = null
  private headerGroupWidthIntial = 0
  private routeListLengthMax = 0

  private updates = 0
  private chart: any = ''

  public getLeftValue (attr) {
    if (attr.fixNumber) {
      return (attr.attrDistribution.valueRange[0] / attr.unitScale).toFixed(attr.fixNumber) + attr.unit
    }
    return Math.round(attr.attrDistribution.valueRange[0] / attr.unitScale) + attr.unit
  }

  public getRightValue (attr) {
    if (attr.fixNumber) {
      return (attr.attrDistribution.valueRange[1] / attr.unitScale).toFixed(attr.fixNumber) + attr.unit
    }
    return Math.round(attr.attrDistribution.valueRange[1] / attr.unitScale) + attr.unit
  }

  public get showFilter () {
    return CandidatesList.showFilter
  }

  get headerResizing () {
    return CandidatesList.headerResizing
  }

  get attributes () {
    return CandidatesList.attributes
  }

  get searching () {
    return CandidatesList.searching
  }

  onReorderHeaders (attr: Attribute) {
    CandidatesList.reorderHeaders(attr)
  }

  mouseDBLClickOnAttribute(e: Event, attr: Attribute) {
    // if (this.showFilter) {
    //   return
    // }

    if (CandidatesList.searching) {
      CandidatesList.sortConflict(attr)
    } else {
      CandidatesList.sortCandidates(attr)
    }
  }

  mouseRightClickOnAttribute(e: Event, attr: Attribute) {
    console.log('right click')
    e.preventDefault()
    CandidatesList.ToggleAttributeGrouping(attr)
  }

  onResetFilter (e: MouseEvent, attr: Attribute) {
    e.preventDefault()
    e.stopPropagation()
    attr.setFilterLeft(0)
    attr.setFilterRight(1)
    if (this.searching) {
      CandidatesList.applyAttributeFilterWhenSearching()
    } else {
      CandidatesList.applyAttributeFilter()
    }
  }

  onStartFilter (e: MouseEvent, attr : Attribute) {
    e.preventDefault()
    e.stopPropagation()
    const rect = (this.$refs[`maskings_${attr.key}`][0] as HTMLElement).getBoundingClientRect()
    const left = rect.left
    const width = rect.width
    attr.setFilterLeft((e.pageX - left) / width)
    attr.setFilterRight((e.pageX - left) / width)

    const movingFunction = (e) => this.onChangeFilterRange(e, attr, width, left)

    const stopFn = () => {
      if (this.searching) {
        CandidatesList.applyAttributeFilterWhenSearching()
      } else {
        CandidatesList.applyAttributeFilter()
      }
      document.removeEventListener('mouseup', stopFn)
      document.removeEventListener('mousemove', movingFunction)
    }

    document.addEventListener('mouseup', stopFn)
    document.addEventListener('mousemove', movingFunction)
  }

  onChangeFilterRange (e : { pageX: number }, attr : Attribute, width: number, left: number) {
    attr.setFilterRight((e.pageX - left) / width)
  }

  mousePressOnRanger (e : { pageX: number }, attr : Attribute, type : string) {
    const width = $('.filter .bar-cont').width()
    if (width) {
      this.ranger = {
        type: type,
        cursorX: e.pageX,
        target: attr,
        width: width
      }
    }
  }

  mouseDidMoveOnColumnHeaders (e : { pageX: number }) {
    if (this.ranger) {
      const delta = (e.pageX - this.ranger.cursorX) / this.ranger.width
      CandidatesList.moveAttributeFilterRange({
        left: this.ranger.type === 'left',
        attr: this.ranger.target,
        delta
      })
      this.ranger.cursorX = e.pageX
    }
  }

  mouseDidReleaseColumnHeaders () {
    if (this.ranger) {
      CandidatesList.applyAttributeFilter()
      this.ranger = null
    }
  }

  resizeHeader (e : { pageX: number }) {
    const addedWidth = e.pageX - this.pageXWhenStartResize
    if (this.headerResized) {
      this.headerResized.updateWeightAndWidth(addedWidth + this.headerWidthIntial)

      if (this.headerResized.group) {
        this.headerResized.group.updateWeightAndWidth(addedWidth + this.headerGroupWidthIntial)
      }
    }
  }

  onStartHeaderResize (e: MouseEvent, attr: Attribute) {
    CandidatesList.changeHeaderResizing()

    this.headerWidthIntial = attr.width
    if (attr.group) {
      this.headerGroupWidthIntial = attr.group.width
    }
    this.pageXWhenStartResize = e.pageX
    this.headerResized = attr

    const stopFn = () => {
      CandidatesList.changeHeaderResizing()
      this.pageXWhenStartResize = 0
      document.removeEventListener('mouseup', stopFn)
      document.removeEventListener('mousemove', this.resizeHeader)
    }

    document.addEventListener('mouseup', stopFn)
    document.addEventListener('mousemove', this.resizeHeader)
  }

  // Get a data set.  10 points.
  private _getData () {
    const points = 20
    return d3.range(points + 2).map((i) => this._getDataPoint(i))
  }

  // Get a single point of data.
  private _getDataPoint(i) {
    const points = 20
    return {x: (i) / points, y: Math.random() / 5}
  }

  private addData (n: number) { // Add a data point to the right of the graph and slide the line to the left.
    // Grab the path
    const path = d3.select(".dengzikun")
    const data = path._groups[0][0].__data__
    // Grab the data from the path
    // const data = path[0][0].__data__

    const rate = 500 // The scroll and update speed of the graph (ms)
    const points = 20 // The number of points to generate
    const width = 200 // Chart width
    const height = 100 // Chart height

    this.updates++
    data.push({x: (this.updates + points) / points, y: n})
    if (n > this.routeListLengthMax) {
      this.routeListLengthMax = n
    }

    const x = d3.scaleLinear().domain([0, 1]).range([0, width])
    const y = d3.scaleLinear().domain([0, this.routeListLengthMax + 1]).range([height, 0])
    //
    // console.log('debug', data)

    // Slap a new random point to the end of the data
    // data.push(this._getDataPoint(data.length))

    // Increment the total number of updates.  This lets us know how much to
    // move the path.  This is a sub-optimal solution.

    // Apply the new data to the path and re-draw.
    path.data([data])
      .transition()
      .duration(rate)
      // Use a linear easing to keep an even scroll
      .ease(d3.easeLinear)
      .attr("d", d3.line()
        .x((d, i) => x(d.x - this.updates / points))
        .y((d) => y(d.y))
        // I'm not sure if this is the interpolation that works best, but I
        // can't find a better one...
        .curve(d3.curveBasis)
      )
  }

  public get routeOptimized () {
    return CandidatesList.routeOptimized
  }

  @Watch('routeOptimized')
  public drawLinechart () {
    if (!this.routeOptimized) {
      return
    }
    const rate = 500 // The scroll and update speed of the graph (ms)
    const points = 20 // The number of points to generate
    const width = 200 // Chart width
    const height = 100 // Chart height
    const padding = 20 // Visual padding for the axis labels
    // let updates = 0 // The number of updates that have been performed

    // These appear to be factories for generaring values.  Will look more at
    // it later.
    const x = d3.scaleLinear().domain([0, 1]).range([0, width])
    const y = d3.scaleLinear().domain([0, 1]).range([height, 0])

    // Create the chart object to the desired parameters.
    const initialData = new Array(22).fill(0).map((d, i) => ({ x: i / points, y: 0 }))
    console.log(initialData)
    const chart = d3.select(".progressive-svg")
      // .data(initialData)
      .data([initialData])
      .append("svg:svg")
      .attr('class', 'wengdi')
      .attr("width", width)
      .attr("height", height)
      .append("svg:g")
      // I have no idea what this does yet.
    this.chart = chart

    console.log(this._getData())

    // Creating rules for the background grid.

    // Now add the line.  This should be last so it sits on top of the
    // reference rulers.
    chart
      .append("svg:path")
      .attr("class", "line dengzikun")
      .attr("d", d3.line()
        .x((d) => x(d.x))
        .y((d) => y(d.y))
        .curve(d3.curveBasis)
      )
    // this.addData()

    // Make it a "live" updating chart.
    // window.setInterval(this.addData, rate)
  }

  public get routeListLength () {
    return CandidatesList.candidateRoutes.length
  }

  public get receiveDataCount () {
    return Manipulation.receiveDataCount
  }

  @Watch('receiveDataCount')
  public onRouteListLengthChanged () {
    if (this.routeOptimized && this.routeListLength > 0) {
      this.addData(this.routeListLength)
    }
  }
}
</script>

<style lang="scss">
@import '../../../style/Constants.scss';
.column-headers {
  padding-left: $RANKING_NAME_CELL_WIDTH;
  height: $RANKING_COLUMN_HEADER_HEIGHT;
  // height: calc(100% - #{$RANKING_CONTENT_HEIGHT});
  // flex: 0 0 $RANKING_COLUMN_HEADER_HEIGHT;
  flex-direction: row;
  display: flex;
  position: relative;
  transition: flex-basis 200ms;
  background-color: #e9edf1;
  border-bottom: 1px solid #e6e6e6;
  flex: 0 0 100px !important;

  .progressive-svg {
    position: absolute;
    width: 200px;
    height: 100%;
    top: 0px;
    left: 0px;

    .route-list-length {
      position: absolute;
      top: 2px;
      left: 2px;
      font-size: 12px;
      color: #777;
    }

    .rule line {
      stroke: #eee;
      shape-rendering: crispEdges;
    }

    .line {
      fill: none;
      stroke: steelblue;
      stroke-width: 1.5px;
    }
  }

  .header {
    padding: 7px 7px 6px 5px;
    position: relative;
    background-color: #dce7f2;

    &.no-transition {
      transition: flex-basis 0s, width 0s;
    }
    &.resizing {
      background-color: $GRAY2;
    }
    &.grouped {
      background-color: $GROUPED;
    }

    &:first-child {
      border-left: 1px solid $GRAY2;
    }

    & > div {
      cursor: default;
    }

    .name-wrap {
      position: absolute;
      left: 10px;
      top: 10px;
      height: 15px;
      width: calc(100% - 30px);

      .name-icon {
        float: left;
        color: #b1c1d1;
      }

      .name {
        color: #808080;
        float: left;
        height: 15px;
        line-height: 15px;;
        font-size: 15px;
        margin-right: 30px;
        padding-left: 5px;
        user-select: none;
        overflow: hidden;
        text-overflow: ellipsis;
        max-width: calc(100% - 50px);
        white-space: nowrap;
      }
    }

    .handle {
      position: absolute;
      width: 2px;
      left: calc(100% - 2px);
      height: 100%;
      top: 0px;
      cursor: col-resize;
      background-color: #fff;
    }

    .delete-btn, .sorting, .grouping, .accept-btn, .reset-btn {
      position: absolute;
      padding: 3px;
      font-size: 15px;
      right: 5px;
      color: $GRAY3;
      width: 20px;
      text-align: center;
    }

    .sorting {
      top: 5px;
      font-size: 14px;
      line-height: 15px;;
      font-size: 15px;
    }

    .active-sorting {
      color: #bdd2e5 !important;
    }

    .grouping {
      top: 5px;
      right: 23px;
      font-size: 14px;
    }

    .filter {
      position: absolute;
      height: 70px;
      left: 0px;
      right: 0px;
      top: 30px;

      .masking-cont {
        position: absolute;
        left: 3%;
        width: calc(93%);
        height: 100%;
        top: 0px;

        .masking-left {
          position: absolute;
          left: 0px;
          top: 0px;
          height: 100%;
          border-right: 1px solid #8ca6ba;
          background-color: rgba(255,255,255,0);

          .value {
            position: absolute;
            left: calc(100% + 2px);
            height: 12px;
            line-height: 12px;
            top: calc(100% - 14px);
            color: #808080;
            font-size: 12px;
            user-select: none;
            pointer-events: none;
          }
        }

        .masking-right {
          position: absolute;
          top: 0px;
          height: 100%;
          border-left: 1px solid #8ca6ba;
          background-color: rgba(255,255,255,0);

          .value {
            position: absolute;
            width: 100px;
            left: -102px;
            height: 12px;
            line-height: 12px;
            top: calc(100% - 14px);
            color: #808080;
            text-align: right;
            font-size: 12px;
            user-select: none;
            pointer-events: none;
          }
        }
      }

      .bar-wrapper {
        height: 50px;
        margin: 0 10px 10px 10px;
        border-bottom: 3px solid #FFB300;
        position: relative;
        padding: 1px 10px;
        box-sizing: border-box;

        .bar-cont {
          width: 100%;
          height: 100%;
          position: relative;
        }

        .bar {
          position: absolute;
          width: 25%;
          bottom: 0;
          background-color: $GRAY2;
          overflow: hidden;
          border-top-left-radius: 3px;
          border-top-right-radius: 3px;
          transition: background-color 200ms, height 200ms;

          .shadow {
            position: absolute;
            left: 0;
            width: 100%;
            bottom: 0;
            background-color: #FFB300;
            transition: background-color 200ms, height 200ms;

            &:hover {
              background-color: lighten(#FFB300, 10%);
            }
          }
        }

        @for $i from 1 through 5 {
          .bar:nth-child(#{$i}) {
            left: percentage(0 + ($i - 1) * 0.2);
            width: 18.5%;
          }
        }

        .range {
          position: absolute;
          // box-sizing: content-box;
          top: calc(100% + 4px);
          transform: translate(-50%, 0);
          border-left: 7px solid transparent;
          border-right: 7px solid transparent;
          border-bottom: 8px solid #FF7043;
          // transition: border-bottom 200ms, left 200ms;
          cursor: pointer;
          user-select: none;

          &:hover, &.active {
            border-bottom: 8px solid lighten(#FF7043, 10%);
          }
        }
      }

      .range-axis {
        position: absolute;
        height: 3px;
        transform: translateY(1px);
        top: 100%;
        background-color: #FF7043;
        border-left: 10px solid #FF7043;
        // transition: width 200ms;
        // border-bottom: 2px solid #607D8B;

        width: 10px;

        &.left {
          left: -10px;
        }

        &.right {
          right: -10px;
        }
      }

      .label {
        display: flex;
        padding: 0 10px;

        .text {
          flex: 1;
          font-size: 12px;
          color: $GRAY5;
          text-shadow: 0 0 1px $GRAY3;
          &.maximum {
            text-align: right;
          }
        }
      }
    }
  }
}

.flip-div {
  position: relative;
  height: 100%;
  width: 100%;
  display: flex;
}

.flip-list-move {
  transition: transform .3s;
}

</style>
