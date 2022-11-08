<template>
  <div class="matrix-container" :style="{
    transform: `translate(${selectedRoute ? 0 : 50}vw, 0px)`
  }">
    <svg width="100%" height="100%" ref="svg" id="matrix">
      <g ref="matrices" transform="translate(0, 0) rotate(45, 0, 0)"></g>
    </svg>
    <div class="route-name-info">{{currentMatrixWave ? indexedRoutes[currentMatrixWave.routeId].name : ''}}</div>
    <div class="overview">
      <div class="left">
        <svg width="100%" height="100%" ref="overview" id="overview"></svg>
      </div>
      <!-- <div class="right">
        <span>Route #{{currentMatrixWave ? currentMatrixWave.routeId + '-' + indexedRoutes[currentMatrixWave.routeId].name : ''}}</span>
        <svg ref="linechart">
          <line x1="0" y1="5" :x2="overviewRightWidth" y2="5" stroke-dasharray="5,5" stroke="#B6D4E6"></line>
          <line x1="0" :y1="overviewRightHeight / 2" :x2="overviewRightWidth" :y2="overviewRightHeight / 2" stroke-dasharray="5,5" stroke="#B6D4E6"></line>
          <line x1="0" :y1="overviewRightHeight - 5" :x2="overviewRightWidth" :y2="overviewRightHeight - 5" stroke-dasharray="5,5" stroke="#B6D4E6"></line>
          <g :transform="'translate(0,' + overviewRightHeight / 2 + ')'">
            <polyline :points="lineChartPolyline" style="fill:none;stroke:#7ABDE6;stroke-width:1.5;"></polyline>
            <g v-for="(item, index) in lineChartPosition" :key="index">
              <circle :cx="item.x" :cy="item.y" r="4" style="fill:white;stroke:#7ABDE6;stroke-width:1.5;"></circle>
            </g>
          </g>
        </svg>
      </div> -->
    </div>
  </div>
</template>

<script lang="ts">
import { Component, Vue, Watch, Ref, Prop } from 'vue-property-decorator'
import _ from 'lodash'
import * as d3 from 'd3'
import svgPanZoom from 'svg-pan-zoom'
import Exploration, { MatrixG, MatrixWave, MatrixBranch, Position, Point, OverviewInfo, RouteBranches, Branch, DayDataFlow, HourDataFlow, WeekdayDataFlow, Route, SingleRouteCheckSequence, SingleCheckSequence, CheckSequenceEachRoute } from '@/store/modules/Exploration'
import Evaluation from '@/store/modules/Evaluation'
import { Conflict } from "@/utils/RouteGroup";
import CandidatesList from '@/store/modules/CandidatesList'
import { RouteTransfers, BarchartInMatrix } from '@/utils/Types'

interface MatrixEncoding {
  color: string
  height: number
  width: number
  margin: number
}

@Component
export default class Matrix extends Vue {
  @Prop({ default: false }) reverse!: boolean

  public selectedStationIDs: number[] = []
  public matrixContainer: MatrixWave[] = [] // 装大matrix的容器
  public panZoomTiger: any = null // pan and zoom的触发器
  public initialZoom = 1 // 初始的zoom
  public curFocusId = -1 // 当前focus的大matrix在matrixContainer中的index

  // 中间所有小matrix部分
  public singleMatrixWidth = 10 // 单个小matrix的大小
  public matrixMargin = 1 // 小matrix之间的margin
  public matrixEncoding = { // 控制小matrix的配置
    color: 'rgb(142,155,224)',
    height: this.singleMatrixWidth,
    width: this.singleMatrixWidth,
    margin: this.matrixMargin
  }

  // hover后的分支
  public branchTextFontSize = 6 // 分支名称的字体大小
  // 左右两边分支总和，会用在overview左部分
  public branchesSumLeft = 0
  public branchesSumRight = 0

  // overview的左部分
  public overviewContainer: OverviewInfo[] = [] // 存放overview中所有菱形信息的容器
  public overviewTiger = null as any // pan and zoom的触发器
  public overviewMatrixSize = 50 // overview中菱形的“半径”
  public overviewMatrixMargin = 5 // overview中菱形间的margin
  // overview的右部分
  public overviewRightWidth = 100 // overview的右部分的linechart的宽度
  public overviewRightHeight = 74 // overview的右部分的linechart的高度

  // station名称部分
  public stationFrontLineLength = 20 // station名字前面的线段的长度
  public stationPostLineLength = 30 // station名字后面的线段的长度
  public stationNameMaxLength = 28 // station名字的最长长度
  public stationTextFontSize = 6 // station名称的字体大小

  // horizon chart部分
  public outColors = ['#ffeccc', '#ffd078', '#cca766'] // in（绿色）部分的颜色
  public inColors = ['#dcebc7', '#b3e074', '#94b366'] // out（橙色）部分的颜色
  public horizonChartMargin = 10 // 每行horizon chart的左右两边的边距
  public horizonChatrWidth = 300 // horizon chart的宽度
  // public horizonChartPerBarWidth = 10 // horizon chart中没小个bar的宽度

  public maxTransferNum = 12

  @Ref('svg') public svg: any // 大matrix的svg的ref
  @Ref('matrices') public matrices: any
  @Ref('overview') public overview: any // overview左部分svg的ref
  @Ref('linechart') public linechart: any // overview右部分linechart的svg的ref

  public beforeCreate () {
    Exploration.loadCheckSequence()
  }

  public mounted () {
    // 更新overview右部分的大小
    const bound = this.linechart.getBoundingClientRect()
    this.overviewRightWidth = bound.width
    this.overviewRightHeight = bound.height
  }

  public get finished () {
    return CandidatesList.finished
  }

  public get searching () {
    return CandidatesList.searching
  }

  public get routesTransfers () {
    return Exploration.routesTransfers
  }

  public get routeMatrixWhenSearching() {
    return Evaluation.currentRouteMatrix
  }

  public get routeStationsWhenSearching() {
    return Evaluation.currentRoute
  }

  public get currentMatrixWave () { // 当前focus的大matrix的信息
    return this.matrixContainer[this.curFocusId]
  }

  public get lineChartPosition () { // linechart的点位置信息
    const currentMatrixWave = this.currentMatrixWave
    if (!currentMatrixWave) return []
    const xInterval = (this.overviewRightWidth - 10) / (currentMatrixWave.stations.length - 1)
    const barChart = currentMatrixWave.barChart
    let x = 5
    let currentFlow = 0
    let maxFlow = 0
    const result = currentMatrixWave.stations.map((item, index) => {
      x += (index === 0 ? 0 : xInterval)
      currentFlow = index === 0 ? barChart.checkin.data[item]
        : currentFlow + barChart.checkin.data[item] - barChart.checkout.data[item]
      if (Math.abs(currentFlow) > maxFlow) maxFlow = Math.abs(currentFlow)
      return { x, y: currentFlow }
    })
    result.forEach(item => { item.y = -item.y / maxFlow * (this.overviewRightHeight / 2 - 5) })
    return result
  }

  public get lineChartPolyline () { // linechart的线的points属性值
    let result = ''
    this.lineChartPosition.forEach(item => {
      result += `${item.x},${item.y} `
    })
    return result
  }

  public get indexedStations () { // 存放所有Station对象，通过stationID检索
    return Exploration.indexedStations
  }

  public get indexedRoutes () { // 存放所有的Route对象，通过routeID检索
    return Exploration.indexedRoutes
  }

  public get selectedRoute () { // 从lineup中点击选择的路线的Route对象
    const len = Exploration.selectedRoutes.length
    return len > 0 ? Exploration.selectedRoutes[len - 1] : null
  }

  public get conflicts () {
    return Evaluation.conflicts
  }

  public get openedConflictIndex () {
    return CandidatesList.openedConflictIndex
  }

  @Watch('openedConflictIndex')
  onOpenedConflictIndexChanged () {
    this.updateConflictHint()
  }

  // @Watch('conflicts')
  updateConflictHint () {
    this.clearConflictHint()
    if (this.conflicts.length === 0) {
      return
    }
    this.conflicts.forEach((conflict: Conflict, cIndex: number) => {
      const dest = conflict.dest
      const stations = this.routeStationsWhenSearching
      const index = stations.indexOf(dest)
      console.log(dest, index, stations)
      if (index < 1) {
        console.log('error in updateConflictHint')
      } else {
        const matrixSize = stations.length * this.matrixEncoding.width
        // this.drawTriangle(index - 1)
        const color = cIndex === this.openedConflictIndex ? '#ffd078' : '#ddd'
        this.drawDashedLine(index, matrixSize, color)
      }
    })
  }

  public clearConflictHint() {
    d3.select('.matrix').selectAll('.conflict-hint').remove()
  }

  // debug () {
  //   const rid = Exploration.matrixHighLightRoute.routes[0]
  //   const stations = Exploration.indexedRoutes[rid].stations
  //   const matrixSize = stations.length * this.matrixEncoding.width
  //   this.drawDashedLine(3, matrixSize)
  // }
  @Watch('finished')
  public onFinishedChanged() {
    if (this.finished) {
      d3.select('.matrix').selectAll('.conflict-hint').remove()
    }
  }

  public drawDashedLine(destStationIDIndex: number, matrixSize: number, color: string) {
    if (this.finished) {
      return
    }
    console.log(destStationIDIndex)
    console.log(this.matrixContainer, this.curFocusId)
    d3.select('.matrix').append('line')
      .attr('stroke-dasharray', '5, 5')
      .attr('y1', 0)
      .attr('y2', matrixSize)
      .attr('x1', -1)
      .attr('x2', -1)
      .attr('stroke', color)
      .attr('class', 'conflict-hint')
      .attr('transform', `translate(${this.horizonChatrWidth + destStationIDIndex * this.horizonChartMargin},${this.horizonChatrWidth})`)
    d3.select('.matrix').append('line')
      .attr('stroke-dasharray', '5, 5')
      .attr('y1', -1)
      .attr('y2', -1)
      .attr('x1', 0)
      .attr('x2', matrixSize)
      .attr('stroke', color)
      .attr('class', 'conflict-hint')
      .attr('transform', `translate(${this.horizonChatrWidth},${this.horizonChatrWidth + destStationIDIndex * this.horizonChartMargin})`)

    console.log(d3.select('.matrix'))
  }

  public drawTriangle(destStationIDIndex: number) {
    this.currentMatrixWave.matrixG.mainG.append('path')
      .attr('d', "M-3 -3 L0 -3 L-1.5 0 Z")
      .attr('fill', 'red')
      .attr('class', 'conflict-hint')
      .attr('transform', `translate(${this.horizonChatrWidth + destStationIDIndex * this.horizonChartMargin},${this.horizonChatrWidth})`)
    this.currentMatrixWave.matrixG.mainG.append('path')
      .attr('d', "M-3 -3 L0 -1 L-3 1 Z")
      .attr('fill', 'red')
      .attr('class', 'conflict-hint')
      .attr('transform', `translate(${this.horizonChatrWidth},${this.horizonChatrWidth + destStationIDIndex * this.horizonChartMargin})`)
  }

  // 新增大matrix
  public addNewMartix (parId: number, reverse: boolean, branch: Branch, direction: number, key: number) {
    if ([3, 4, 1, 2].indexOf(direction) + 1 === this.matrixContainer[parId].direction) {
      console.log('此方向不能展开')
      return
    }
    const item = this.matrixContainer.find(item => item && item.parId === parId && item.direction === direction)
    this.deleteMatrix(item as MatrixWave, parId)
    if (item && item.routeId === branch.route_id) return
    const data = this.indexedRoutes[branch.route_id].stations
    const parMatrixWave = this.matrixContainer[parId]
    // console.log('debug', 'call appendG in addNewMartix')
    const newMatrixG = this.appendG(this.matrixContainer.length, reverse)
    const tempObj = this.drawMatrixWave(this.matrixContainer.length, reverse, newMatrixG, data, parMatrixWave, direction, branch, key)
    let left = {id: -1, branch: null} as MatrixBranch
    let right = {id: -1, branch: null} as MatrixBranch
    if (direction === 1 || direction === 2) {
      left = {id: parId, branch}
      parMatrixWave.sibling.right = {id: this.matrixContainer.length, branch}
    } else if (direction === 3 || direction === 4) {
      right = {id: parId, branch}
      parMatrixWave.sibling.left = {id: this.matrixContainer.length, branch}
    }
    const selected = {xKey: -1, yKey: -1, flag: false}
    const inSelected = {xKey: -1, yKey: -1, flag: false}
    const outSelected = {xKey: -1, yKey: -1, flag: false}
    const newMatrixWave = {...tempObj, reverse, stations: data, routeId: branch.route_id, parId, direction, selected, sibling: {left, right}, horizonSelected: {in: inSelected, out: outSelected}}
    this.matrixContainer.push(newMatrixWave)
    Exploration.addMatrixHighLightRoute(branch.route_id)
    if (this.panZoomTiger) {
      this.focusToNewMatrix(newMatrixWave)
    }
  }

  // 为新的大matrix append预先的svg元素
  public appendG (id: number, reverse: boolean) {
    const mainG = d3.select(this.matrices).append('g').attr('transform', 'translate(0, 0)')
      .attr('id', `matrix${id}`).attr('class', `matrix`)
      .on('click', () => {
        const matrixWave = this.matrixContainer[id]
        matrixWave.selected.flag = !matrixWave.selected.flag
      })
    // this.drawToolbar(id, mainG, reverse)
    const inG = mainG.append('g').attr('height', `${this.horizonChatrWidth}`).attr('transform', `translate(${this.horizonChatrWidth}, 0)`)
    const outG = mainG.append('g').attr('width', `${this.horizonChatrWidth}`).attr('transform', `translate(0, ${this.horizonChatrWidth})`)
    inG.append('g').attr('id', 'in')
    inG.append('g').attr('id', 'content')
    outG.append('g').attr('id', 'out')
    outG.append('g').attr('id', 'content')
    const contentG = mainG.append('g').attr('class', 'mainG')
    const rectsG = contentG.append('g').attr('class', 'rectsG').attr('transform', `translate(${this.horizonChatrWidth}, ${this.horizonChatrWidth})`)
    const inRoute = rectsG.append('g').attr('class', 'inRoute')
    const outRoute = rectsG.append('g').attr('class', 'outRoute')
    const flowG = rectsG.append('g').attr('class', 'flowG')
    const stationG = rectsG.append('g').attr('class', 'stationG')
    const selectedG = contentG.append('g').attr('class', 'selectedG').attr('transform', `translate(${this.horizonChatrWidth}, ${this.horizonChatrWidth})`)
    const lineG = d3.select(this.matrices).append('g').attr('class', 'lineG')
    return { mainG, inG, outG, contentG, rectsG, inRoute, outRoute, flowG, stationG, lineG, selectedG } as MatrixG
  }

  // 画两个horizon chart之间的toolbar
  public drawToolbar (id: number, mainG: any, reverse: boolean, type: string) {
    const lineLength = 70
    const lineHeight = 10
    const [x, y] = [(this.horizonChatrWidth - lineLength) / 2, this.horizonChatrWidth - lineHeight - 5]
    const [x1, y1] = [0, 0]
    const [x2, y2] = [x1 + lineLength, y1]
    const changeG = mainG.append('g').attr('id', 'toolbar').attr('style', 'cursor:pointer;')
      .on('click', () => this.changeHorizonData(id))
    const transformTranslate1 = `translate(${x}, ${y})`
    changeG.append('line').attr('x1', x1)
      .attr('y1', y1)
      .attr('x2', x2)
      .attr('y2', y2)
      .attr('style', 'stroke-width:20;stroke:#d0d0e2;stroke-linecap:round;')
    changeG.append('circle').attr('cx', x1 + 5).attr('cy', y1).attr('r', 5)
      .attr('style', 'stroke:#6e7f80;stroke-width:1.5;fill:transparent;')
    changeG.append('line').attr('x1', x1 + 5).attr('y1', y1).attr('x2', x1 + 5).attr('y2', y1 - 3)
      .attr('style', 'stroke:#6e7f80;stroke-width:1.5;stroke-linecap:round;')
    changeG.append('line').attr('x1', x1 + 5).attr('y1', y1).attr('x2', x1 + 5 + 1.5).attr('y2', y1 + 1.5)
      .attr('style', 'stroke:#6e7f80;stroke-width:1.5;stroke-linecap:round;')
    changeG.append('text').attr('x', x1 + 5 + 10).attr('y', y1 + 4.5).text('HOURLY')
      .attr('style', 'font-size:14px;fill:#6e7f80;')
    const newChangeG = changeG.clone(true)
    newChangeG.on('click', () => this.changeHorizonData(id))
    const transformTranslate2 = `translate(${y}, ${x + lineLength}) rotate(-90)`
    const transformMatrix1 = this.getTransformMatrix(changeG.node(), 'y', reverse)
    const transformMatrix2 = this.getTransformMatrix(newChangeG.node(), 'y', reverse)
    changeG.attr('transform', `${transformTranslate1} ${transformMatrix1}`)
    newChangeG.attr('transform', `${transformTranslate2} ${transformMatrix2}`)
    mainG.append(() => newChangeG.node())
  }

  // 画大matrix，也就是matrix wave
  public drawMatrixWave (id: number, reverse: boolean, matrixG: MatrixG, stationsID: number[],
    parMatrixWave: MatrixWave, directon: number, branch: Branch, parKey: number) {
    // 当前route的checkin和checkout的row data
    const singleCheckSequence = this.getSingleCheckSquence(branch.route_id)
    const singleTransferObj: RouteTransfers = this.routesTransfers[branch.route_id]
    // console.log('debug', singleCheckSequence, this.getHorizonChartData(singleCheckSequence.checkin, stationsID))
    // 处理好的，用于horizon chart的checkin和checkout数据
    const horizonChart = {
      data: {
        checkin: this.getHorizonChartData(singleCheckSequence.checkin, stationsID),
        checkout: this.getHorizonChartData(singleCheckSequence.checkout, stationsID)
      },
      current: 'hour'
    }
    this.drawToolbar(id, matrixG.mainG, reverse, horizonChart.current)
    // 处理好的，用于line chart的checkin和checkout数据
    const barChart: BarchartInMatrix = {
      checkin: this.getBarChartData(singleCheckSequence.checkin, stationsID),
      checkout: this.getBarChartData(singleCheckSequence.checkout, stationsID),
      transferin: singleTransferObj.checkinCount,
      transferout: singleTransferObj.checkoutCount,
      transferoutDetail: singleTransferObj.checkoutDetail,
      transferinDetail: singleTransferObj.checkinDetail
    }
    // console.log('debug', 'drawing matrix', horizonChart.current, this.horizonChartPerBarWidth)
    // 画中间的小matrix和它对应的station部分，返回每个station名称的d3.selection
    let stationTextsObj: any = this.drawMatrix(id, stationsID, matrixG, this.matrixEncoding,
      matrixG.inRoute, matrixG.outRoute, reverse, branch.route_id, barChart)
    // 画in的horizon chart，并返回所有bar的d3.selection的数组
    const horizonChartIn = this.drawIn(id, stationsID, matrixG.inG.select('#content'), {
      color: 'rgb(122, 201, 67)',
      height: this.horizonChatrWidth,
      width: this.singleMatrixWidth,
      margin: 1
    }, horizonChart.data.checkin, horizonChart.current)
    // 画out的horizon chart，并返回所有bar的d3.selection的数组
    const horizonChartOut = this.drawOut(id, stationsID, matrixG.outG.select('#content'), {
      color: 'rgb(255, 147, 30)',
      height: this.singleMatrixWidth,
      width: this.horizonChatrWidth,
      margin: this.matrixMargin
    }, horizonChart.data.checkout, horizonChart.current)
    // 画好后获取这个大matrix的size
    const box = matrixG.mainG.node().getBBox()
    // 如果这个大matrix是反过来了，就用transform的matrix翻转一下
    if (reverse) matrixG.mainG.attr('transform', `matrix(0, -1, -1, 0, ${box.width},  ${box.height})`)
    let x = parMatrixWave.x
    let y = parMatrixWave.y
    let newTransform = ''
    const maxWidth = parMatrixWave.box ? Math.max(box.width, parMatrixWave.box.width) : 0
    const maxHeight = parMatrixWave.box ? Math.max(box.height, parMatrixWave.box.height) : 0
    if (directon > 0) {
      let inOrOut = 'in'
      if (directon === 1) {
        x = parMatrixWave.x - this.horizonChatrWidth
        y = parMatrixWave.y - maxHeight
        newTransform = `translate(${x}, ${y})`
        inOrOut = 'out'
      } else if (directon === 2) {
        x = parMatrixWave.x + maxWidth + this.horizonChatrWidth
        y = parMatrixWave.y + this.horizonChatrWidth
        newTransform = `translate(${-y}, ${-x})`
        inOrOut = 'out'
      } else if (directon === 3) {
        x = parMatrixWave.x + this.horizonChatrWidth
        y = parMatrixWave.y + maxHeight + this.horizonChatrWidth
        newTransform = `translate(${-y}, ${-x})`
      } else if (directon === 4) {
        x = parMatrixWave.x - maxWidth
        y = parMatrixWave.y - this.horizonChatrWidth
        newTransform = `translate(${x}, ${y})`
      }
      matrixG.mainG.attr('transform', `${matrixG.mainG.attr('transform')} ${newTransform}`)
      const containerG = parMatrixWave.matrixG[`${inOrOut}Route`]
      const target = containerG.select(`#station${parKey}`)
      const tempBranchG = matrixG.lineG.append('g').attr('transform', `translate(${this.horizonChatrWidth}, ${this.horizonChatrWidth})`)
      let flag = false
      target.selectAll('line').clone(true).nodes().forEach(item => {
        if (!flag) tempBranchG.append(() => item)
        if (item.id === `line${branch.route_id}`) flag = true
      })
      const targetBranch = target.select(`#route${branch.route_id}`).clone(true)
      const targetCircle = targetBranch.select('circle')
      const cx = Number(targetCircle.attr('cx')) | 0
      const cy = Number(targetCircle.attr('cy')) | 0
      this.deleteBranchText(containerG)
      this.drawBranches(id, branch.route_id, stationsID, stationTextsObj[`${inOrOut}StationTexts`],
        this.matrixEncoding, reverse, inOrOut, matrixG, barChart)
      const parP = { x: parMatrixWave.x, y: parMatrixWave.y, box: parMatrixWave.box }
      const curP = { x, y, box }
      const parTextPosition = this.getTextPosition(curP, parP, parMatrixWave.stationTextsObj, parKey, directon, 'par', {x: cx, y: cy})
      const curTextPosition = this.getTextPosition(curP, parP, stationTextsObj, branch.to_stop_num - 1, directon, 'cur')
      const branchRecoderNum = branch.records.length + 1
      // 画两个大matrix之间的折线
      matrixG.lineG.append('polyline')
        .attr('points', `${parTextPosition.curPoint.x},${parTextPosition.curPoint.y}
          ${parTextPosition.edgePoint.x},${parTextPosition.edgePoint.y}
          ${curTextPosition.edgePoint.x},${curTextPosition.edgePoint.y}
          ${curTextPosition.curPoint.x},${curTextPosition.curPoint.y}`)
        .attr('style', `fill:none;stroke:${this[`${inOrOut}Colors`][0]};stroke-width:1;`)
      const newTargetBranch = tempBranchG.append(() => targetBranch.node())
      tempBranchG.attr('transform', `${parMatrixWave.matrixG.mainG.attr('transform')} ${tempBranchG.attr('transform')}`)
      targetBranch.select('path').remove()
      // 画branch名称旁边的关闭按钮X
      newTargetBranch.append('line')
        .attr('x1', cx - 3 / Math.sqrt(2))
        .attr('y1', cy + 3 / Math.sqrt(2))
        .attr('x2', cx + 3 / Math.sqrt(2))
        .attr('y2', cy - 3 / Math.sqrt(2))
        .attr('stroke-linecap', 'round')
        .attr('stroke-width', 1.5)
        .attr('stroke', this[`${inOrOut}Colors`][2])
      newTargetBranch.append('line')
        .attr('x1', cx - 3 / Math.sqrt(2))
        .attr('y1', cy - 3 / Math.sqrt(2))
        .attr('x2', cx + 3 / Math.sqrt(2))
        .attr('y2', cy + 3 / Math.sqrt(2))
        .attr('stroke-linecap', 'round')
        .attr('stroke-width', 1.5)
        .attr('stroke', this[`${inOrOut}Colors`][2])
      newTargetBranch.attr('style', 'cursor:pointer;')
      newTargetBranch.on('click', () => {
        this.deleteMatrix(this.matrixContainer[id], -1, parMatrixWave)
        this.focusToNewMatrix(parMatrixWave)
      })
    } else {
      // 如果是只有一个大matrix的情况，画两边的树状branch
      this.drawBranches(id, branch.route_id, stationsID, stationTextsObj.outStationTexts,
        this.matrixEncoding, reverse, 'out', matrixG, barChart)
      this.drawBranches(id, branch.route_id, stationsID, stationTextsObj.inStationTexts,
        this.matrixEncoding, reverse, 'in', matrixG, barChart)
    }
    stationTextsObj = {...stationTextsObj, horizonChartIn, horizonChartOut}
    return { box, x, y, matrixG, stationTextsObj, horizonChart, barChart }
  }

  // 用于获取链接两个大matrix的线的位置
  public getTextPosition (curP: Position, parP: Position, textObj: any, index: number, direction: number, parOrCur: string, start: any = null) {
    let textG: any
    let reverse = false
    let type = 'in'
    const position = parOrCur === 'par' ? parP : curP
    let x = position.x
    let y = position.y
    let edgePoint = { x: 0, y: 0, box: null }
    let textPosition = [0, 0]
    if (direction === 1 || direction === 2) {
      textG = parOrCur === 'par' ? textObj.outStationTexts[index] : textObj.inStationTexts[index]
      type = parOrCur === 'par' ? 'out' : 'in'
    } else if (direction === 3 || direction === 4) {
      textG = parOrCur === 'par' ? textObj.inStationTexts[index] : textObj.outStationTexts[index]
      type = parOrCur === 'par' ? 'in' : 'out'
    }
    if (direction === 1 || direction === 4) {
      reverse = parOrCur === 'par'
    } else if (direction === 2 || direction === 3) {
      reverse = parOrCur !== 'par'
    }
    if (start) {
      textPosition = [start.x, start.y]
    } else {
      const textBox = textG.node().getBBox()
      edgePoint = { x: 0, y: 0, box: textBox }
      try {
        const textPositionStr = textG.attr('transform').split('translate(')[1].split(')')[0].split(', ')
        textPosition = [Number(textPositionStr[0]), Number(textPositionStr[1])]
      } catch (e) {
        console.error('text的transform属性不存在translate')
      }
      if (type === 'in') {
        textPosition[0] -= (textBox.height / 2)
        textPosition[1] += textBox.width
      } else if (type === 'out') {
        textPosition[0] += textBox.width
        textPosition[1] -= (textBox.height / 2)
      }
    }
    if (reverse) {
      x += (position.box.height - textPosition[1] - this.horizonChatrWidth)
      y += (position.box.width - textPosition[0] - this.horizonChatrWidth)
    } else {
      x += textPosition[0] + this.horizonChatrWidth
      y += textPosition[1] + this.horizonChatrWidth
    }
    if (direction === 1) {
      edgePoint.x = x
      edgePoint.y = parP.y + (curP.y - parP.y + curP.box.height) / 2
    } else if (direction === 2) {
      edgePoint.x = curP.x - (curP.x - parP.x - parP.box.width) / 2
      edgePoint.y = y
    } else if (direction === 3) {
      edgePoint.x = x
      edgePoint.y = curP.y + (parP.y - curP.y + parP.box.height) / 2
    } else if (direction === 4) {
      edgePoint.x = parP.x - (parP.x - curP.x - curP.box.width) / 2
      edgePoint.y = y
    }
    return {
      curPoint: { x, y, box: null } as Position,
      edgePoint
    }
  }

  // 获取某个route的check sequence，也就是checkin和checkout数据
  public getSingleCheckSquence (routeId: number) {
    const result: SingleRouteCheckSequence = {
      checkin: {day: [], hour: [], weekday: []},
      checkout: {day: [], hour: [], weekday: []}
    }
    if (Exploration.checkSequence) {
      result.checkin.day = Exploration.checkSequence.day.checkin[routeId]
      result.checkout.day = Exploration.checkSequence.day.checkout[routeId]
      result.checkin.hour = Exploration.checkSequence.hour.checkin[routeId]
      result.checkout.hour = Exploration.checkSequence.hour.checkout[routeId]
      result.checkin.weekday = Exploration.checkSequence.weekday.checkin[routeId]
      result.checkout.weekday = Exploration.checkSequence.weekday.checkout[routeId]
    }
    return result
  }

  // 根据check sequence处理成用于horizon chart 的数据
  public getHorizonChartData (rowData: SingleCheckSequence, stations: number[]) {
    const result: CheckSequenceEachRoute = {}
    const dateStrs: string[] = this.getDiffDate('2013-04-01', '2013-05-31')
    const hourStrs = new Array(24).fill('').map((item, index) => String(index))
    const weekdayStrs = new Array(7).fill('').map((item, index) => String(index))
    stations.forEach(item => {
      const tempDayDataFlow = rowData.day.filter(item1 => item1.station === item) || []
      const tempHourDataFlow = rowData.hour.filter(item1 => item1.station === item) || []
      const tempWeekdayDataFlow = rowData.weekday.filter(item1 => item1.station === item) || []
      const tempDayResult = dateStrs.map(day => {
        return tempDayDataFlow.find(item2 => item2.day === day) || {station: item, day, times: 0}
      })
      const tempHourResult = hourStrs.map(hour => {
        return tempHourDataFlow.find(item2 => item2.hour === hour) || {station: item, hour, times: 0}
      })
      const tempWeekdayResult = weekdayStrs.map(weekday => {
        return tempWeekdayDataFlow.find(item2 => item2.weekday === weekday) || {station: item, weekday, times: 0}
      })
      result[item] = {day: tempDayResult, hour: tempHourResult, weekday: tempWeekdayResult}
    })
    return result
  }

  // 根据check sequence处理成用于line chart 的数据
  public getBarChartData (rowData: SingleCheckSequence, stations: number[]) {
    const result: {data: {[station: number]: number}, maxTimes: number} = {data: {}, maxTimes: 0}
    const dateStrs: string[] = this.getDiffDate('2013-04-01', '2013-05-31')
    stations.forEach(item => {
      const tempDayDataFlow = rowData.day.filter(item1 => item1.station === item) || []
      let sum = 0
      tempDayDataFlow.forEach(item => {
        sum += item.times
      })
      result.data[item] = sum
      if (result.maxTimes < sum) result.maxTimes = sum
    })
    return result
  }

  public getHorizonChartPerBarWidth (horizonChartCurrent): number {
    const texts = ['M-DAILY', 'W-DAILY', 'HOURLY']
    const types = ['day', 'weekday', 'hour']
    const curTextIndex = (types.indexOf(horizonChartCurrent)) % 3
    const numObj = {day: 4, hour: 10, weekday: 32}
    // console.log('debug', numObj[types[curTextIndex]], curTextIndex, types[curTextIndex])
    return numObj[types[curTextIndex]]
  }

  // 更换horizon chart 的数据，在day、weekday和hour间更换
  public changeHorizonData (id: number) {
    const matrixWave = this.matrixContainer[id]
    const horizonChart = matrixWave.horizonChart
    // console.log('debug', horizonChart)
    const texts = ['M-DAILY', 'W-DAILY', 'HOURLY']
    const types = ['day', 'weekday', 'hour']
    const curTextIndex = (types.indexOf(horizonChart.current) + 1) % 3
    matrixWave.matrixG.mainG.selectAll('#toolbar').selectAll('text').text(texts[curTextIndex])
    horizonChart.current = types[curTextIndex]
    matrixWave.matrixG.inG.select('#content').selectAll('g').remove()
    matrixWave.matrixG.inG.select('#content').selectAll('rect').remove()
    matrixWave.matrixG.outG.select('#content').selectAll('g').remove()
    matrixWave.matrixG.outG.select('#content').selectAll('rect').remove()
    const numObj = {day: 4, hour: 10, weekday: 32}
    // this.horizonChartPerBarWidth = numObj[types[curTextIndex]]
    this.drawIn(id, matrixWave.stations, matrixWave.matrixG.inG.select('#content'), {
      color: 'rgb(122, 201, 67)',
      height: this.horizonChatrWidth,
      width: this.singleMatrixWidth,
      margin: this.matrixMargin
    }, horizonChart.data.checkin, types[curTextIndex])
    this.drawOut(id, matrixWave.stations, matrixWave.matrixG.outG.select('#content'), {
      color: 'rgb(255, 147, 30)',
      height: this.singleMatrixWidth,
      width: this.horizonChatrWidth,
      margin: this.matrixMargin
    }, horizonChart.data.checkout, types[curTextIndex])
  }

  // 画小matrix
  public drawMatrix (id: number, data: number[], matrixG: MatrixG, encoding: MatrixEncoding, inRoute: any, outRoute: any, reverse: boolean, routeId: number, barChart: Barchart) {
    console.log(matrixG)
    const inStationTexts: any[] = []
    const outStationTexts: any[] = []
    const matrixField: any[] = []
    let y = 0
    matrixG.contentG.on('mouseover', () => this.mouseOver(id, encoding))
      .on('mouseleave', () => this.mouseLeave(id))
    data.forEach((outStationId: number, yKey: number) => {
      let x = 0
      matrixField[yKey] = []
      data.forEach((inStationId: number, xKey: number) => {
        const opacity = this.getOpacity(this.indexedRoutes[routeId], inStationId, outStationId)
        const rect = matrixG.flowG.append('rect')
          .attr('height', (encoding.height - encoding.margin * 2))
          .attr('width', (encoding.width - encoding.margin * 2))
          .attr('transform', `translate(${x}, ${y})`)
          .attr('style', `fill:${encoding.color};fill-opacity:${opacity};`)
          .attr('class', `flow-cell`)
          .attr('id', `flowcell-${inStationId}_${outStationId}`)
        matrixField[yKey].push(rect)
        if (xKey === data.length - 1) {
          const stationInfoG = matrixG.flowG.append('g')
            .attr('id', `out${yKey}`)
            .attr('transform', `translate(0, -1)`)
          const xStation = this.indexedStations[outStationId]
          const tempText = stationInfoG.append('text').attr('height', encoding.height / 2)
            .attr('font-size', `${this.stationTextFontSize}px`)
            .attr('class', 'station-name')
            .attr('id', `inID-${inStationId}_outID-${outStationId}`)
            .text(this.abbreviateString(xStation.name, 8))
          const stationMatrix = this.getTransformMatrix(tempText.node(), 'y', reverse)
          const textX = x + encoding.width + encoding.margin + this.stationFrontLineLength + 2
          const textY = y + encoding.height - encoding.margin
          tempText.attr('transform', `translate(${textX}, ${textY - 1.5}) ${stationMatrix}`)
          outStationTexts.push(tempText)
          stationInfoG.append('line').attr('style', 'stroke:#B6D4E6;')
            .attr('x1', x + encoding.width + encoding.margin).attr('y1', y + encoding.height / 2)
            .attr('x2', x + encoding.width + encoding.margin + this.stationFrontLineLength).attr('y2', y + encoding.height / 2)
          const totalFlowRatio = this.stationFrontLineLength * barChart.checkout.data[outStationId] / barChart.checkout.maxTimes | 1
          stationInfoG.append('rect')
            .attr('height', (encoding.height - encoding.margin * 2))
            .attr('width', totalFlowRatio)
            .attr('transform', `translate(${x + encoding.width + encoding.margin}, ${y + encoding.margin})`)
            .attr('style', `fill:${this.outColors[0]};`)
          // const transferRatio = this.stationFrontLineLength * barChart.transferout[outStationId] / barChart.checkout.maxTimes
          // stationInfoG.append('rect')
          //   .attr('height', (encoding.height - encoding.margin * 2))
          //   .attr('width', transferRatio)
          //   .attr('transform', `translate(${x + encoding.width + encoding.margin}, ${y + encoding.margin})`)
          //   .attr('style', `fill:${this.outColors[2]};`)
          const radius = 4
          const labelx = textX + this.stationNameMaxLength + radius
          const labely = textY - encoding.height / 2 + encoding.margin
          const labelTransform = ''
          const branches = this.getNewStationBranches(yKey, routeId, data, 'out')
          stationInfoG.append('circle').attr('cx', labelx).attr('cy', labely)
            .attr('r', radius).attr('fill', '#cca766').attr('fill-opacity', barChart.transferout[outStationId] / this.maxTransferNum)
            .attr('transform', labelTransform)
          const num = stationInfoG.append('text').attr('x', labelx).attr('y', labely + radius - 1)
            .text(branches.length).attr('style', 'font-size:8px;')
            .attr('fill', 'white').attr('transform', labelTransform)
          num.attr('x', labelx - num.node().getBBox().width / 2)
          const labelMatrix = this.getTransformMatrix(num.node(), 'y', reverse)
          num.attr('transform', `${labelTransform} ${labelMatrix}`)
        }
        if (yKey === data.length - 1) {
          const stationInfoG = matrixG.flowG.append('g')
            .attr('id', `in${xKey}`)
            .attr('transform', `translate(-1, 0)`)
          const yStation = this.indexedStations[inStationId]
          const tempText = stationInfoG.append('text').attr('height', encoding.height / 2)
            .attr('font-size', `${this.stationTextFontSize}px`)
            .attr('text-anchor', 'end')
            .attr('class', 'station-name')
            .attr('id', `inID-${inStationId}_outID-${outStationId}`)
            .text(this.abbreviateString(yStation.name, 8))
          const stationMatrix = this.getTransformMatrix(tempText.node(), 'y', reverse)
          const textX = x + encoding.width - encoding.margin
          const textY = y + encoding.height + encoding.margin + this.stationFrontLineLength + 2
          tempText.attr('transform', `translate(${textX - 1.5}, ${textY}) rotate(-90, 0, 0) ${stationMatrix}`)
          inStationTexts.push(tempText)
          stationInfoG.append('line').attr('style', 'stroke:#B6D4E6;')
            .attr('x1', x + encoding.width / 2).attr('y1', y + encoding.height + encoding.margin)
            .attr('x2', x + encoding.width / 2).attr('y2', y + encoding.height + encoding.margin + this.stationFrontLineLength)
          const totalFlowRatio = this.stationFrontLineLength * barChart.checkin.data[inStationId] / barChart.checkin.maxTimes | 1
          stationInfoG.append('rect')
            .attr('height', totalFlowRatio)
            .attr('width', encoding.height - encoding.margin * 2)
            .attr('transform', `translate(${x + encoding.margin}, ${y + encoding.height + encoding.margin})`)
            .attr('style', `fill:${this.inColors[0]};`)
          // const transferRatio = this.stationFrontLineLength * barChart.transferin[inStationId] / barChart.checkin.maxTimes
          // stationInfoG.append('rect')
          //   .attr('height', transferRatio)
          //   .attr('width', encoding.height - encoding.margin * 2)
          //   .attr('transform', `translate(${x + encoding.margin}, ${y + encoding.height + encoding.margin})`)
          //   .attr('style', `fill:${this.inColors[2]};`)
          const radius = 4
          const labelx = textX - encoding.height / 2 + encoding.margin
          const labely = textY + this.stationNameMaxLength + radius
          const labelTransform = `rotate(-90, ${labelx}, ${labely})`
          const branches = this.getNewStationBranches(xKey, routeId, data, 'in')
          stationInfoG.append('circle').attr('cx', labelx).attr('cy', labely)
            .attr('r', radius).attr('fill', '#94b366').attr('fill-opacity', barChart.transferin[inStationId] / this.maxTransferNum)
            .attr('transform', labelTransform)
          const num = stationInfoG.append('text').attr('x', labelx).attr('y', labely + radius - 1)
            .text(branches.length).attr('style', 'font-size:8px;')
            .attr('fill', 'white').attr('transform', labelTransform)
          num.attr('x', labelx - num.node().getBBox().width / 2)
          const labelMatrix = this.getTransformMatrix(num.node(), 'y', reverse)
          num.attr('transform', `${labelTransform} ${labelMatrix}`)
        }

        x += encoding.width
      })
      y += encoding.height
    })
    return { inStationTexts, outStationTexts, matrixField }
  }

  // 缩写字符串
  public abbreviateString (str: string, maxLength: number) {
    if (str.length > maxLength) return str.substring(0, maxLength - 1).concat('...')
    return str
  }

  // 获取小matrix的透明度
  public getOpacity (route: Route, inN: number, outN: number) {
    if (this.searching && this.routeMatrixWhenSearching) {
      let result = 0
      if (this.routeMatrixWhenSearching[inN] && this.routeMatrixWhenSearching[inN][outN]) {
        result = this.routeMatrixWhenSearching[inN][outN] / this.getMaxFlowNum(route)
      }
      return result
    } else {
      let result = 0
      if (route && route.matrix[inN] && route.matrix[inN][outN]) {
        result = route.matrix[inN][outN] / this.getMaxFlowNum(route)
      }
      return result
    }
  }

  // 鼠标在小matrix上方时触发的函数
  public mouseOver (id: number, encoding: MatrixEncoding) {
    const matrixWave = this.matrixContainer[id]
    if (matrixWave.selected.flag) return
    const matrixG = matrixWave.matrixG
    const selected = matrixWave.selected
    const matrics = matrixWave.stationTextsObj.matrixField
    const svg = matrixG.flowG
    const stationIDs = this.indexedRoutes[matrixWave.routeId].stations
    const maxWidth = encoding.width * stationIDs.length
    const position = d3.mouse(svg.node())
    const lastRect = !matrics[selected.yKey] ? null : matrics[selected.yKey][selected.xKey]
    const edge = maxWidth + this.stationFrontLineLength + this.stationNameMaxLength
    if (position[0] >= edge || position[1] >= edge) {
      svg.attr('style', 'opacity: 1')
      this.leaveMatrix(lastRect, id)
    } else {
      const xKey = Math.floor(position[0] / encoding.width) > stationIDs.length ? stationIDs.length : Math.floor(position[0] / encoding.width)
      const yKey = Math.floor(position[1] / encoding.height) > stationIDs.length ? stationIDs.length : Math.floor(position[1] / encoding.height)
      // svg.attr('style', 'opacity: 0.2')
      const rightBranch = matrixWave.sibling.right.branch
      const leftBranch = matrixWave.sibling.left.branch
      if (leftBranch) {
        console.log('leftBranch')
        const tempId = matrixWave.sibling.left.id
        const siblingMatrix = this.matrixContainer[tempId]
        this.leaveMatrix(null, tempId)
        const stationLenght = this.indexedRoutes[siblingMatrix.routeId].stations.length
        const tempMaxWidth = encoding.width * stationLenght
        const tempXKey = stationLenght
        if (xKey === leftBranch.from_stop_num - 1 && stationIDs[xKey] === leftBranch.from_station_id) {
          const tempYKey = leftBranch.to_stop_num - 1
          this.enterMatrix(null, tempId, {xKey: tempXKey, yKey: tempYKey, length: stationLenght, maxWidth: tempMaxWidth})
        } else if (xKey === leftBranch.to_stop_num - 1 && stationIDs[xKey] === leftBranch.to_station_id) {
          const tempYKey = leftBranch.from_stop_num - 1
          this.enterMatrix(null, tempId, {xKey: tempXKey, yKey: tempYKey, length: stationLenght, maxWidth: tempMaxWidth})
        }
      }
      if (rightBranch) {
        console.log('rightBranch')
        const tempId = matrixWave.sibling.right.id
        const siblingMatrix = this.matrixContainer[tempId]
        this.leaveMatrix(null, tempId)
        const stationLenght = this.indexedRoutes[siblingMatrix.routeId].stations.length
        const tempYKey = stationLenght
        if (yKey === rightBranch.from_stop_num - 1 && stationIDs[yKey] === rightBranch.from_station_id) {
          const tempXKey = rightBranch.to_stop_num - 1
          this.enterMatrix(null, tempId, {xKey: tempXKey, yKey: tempYKey, length: stationLenght})
        } else if (yKey === rightBranch.to_stop_num - 1 && stationIDs[yKey] === rightBranch.to_station_id) {
          const tempXKey = rightBranch.from_stop_num - 1
          this.enterMatrix(null, tempId, {xKey: tempXKey, yKey: tempYKey, length: stationLenght})
        }
      }
      if (xKey === selected.xKey && yKey === selected.yKey) return
      // let rect = null
      // if (xKey < stationIDs.length && yKey < stationIDs.length) rect = matrics[yKey][xKey]
      this.leaveMatrix(lastRect, id)
      this.enterMatrix(null, id, {xKey, yKey, length: stationIDs.length})
      selected.xKey = xKey
      selected.yKey = yKey
    }
  }

  // 鼠标离开小matrix上方时触发的函数
  public mouseLeave (id: number) {
    const matrixWave = this.matrixContainer[id]
    const selected = matrixWave.selected
    if (selected.flag) return
    const matrixG = matrixWave.matrixG
    const matrics = this.matrixContainer[id].stationTextsObj.matrixField
    matrixG.flowG.attr('style', 'opacity: 1')
    const lastRect = !matrics[selected.yKey] ? null : matrics[selected.yKey][selected.xKey]
    this.leaveMatrix(lastRect, id)
  }

  private drawHightlightRect (x: number, y: number, width: number, height: number, matrixId: number, eleClass: string) {
    const matrixWave = this.matrixContainer[matrixId]
    const matrixG = matrixWave.matrixG
    matrixG.selectedG.append('rect')
      .attr('class', eleClass + ' highlight-rect')
      .attr('x', x)
      .attr('y', y)
      .attr('width', width)
      .attr('height', height)
      .attr('style', 'stroke:#d35400;stroke-width:0.5;fill:none;')
      // .on('click', () => {
      //   matrixWave.selected.flag = !matrixWave.selected.flag
      // })
  }

  // mouseOver会确定到底进入了哪个matrix，然后触发此函数
  public enterMatrix (rect: any, matrixId: number, indexes: { xKey: number, yKey: number, length: number, maxWidth?: number }, flag = false) {
    // console.log(rect, matrixId, indexes, flag)
    const encoding = this.matrixEncoding

    // hovered index
    const nStation = this.searching ? this.selectedStationIDs.length : indexes.length
    const xIndex = indexes.xKey < 0 ? nStation : indexes.xKey
    const yIndex = indexes.yKey < 0 ? nStation : indexes.yKey

    // three situation: 1) indexes.xKey === length; 2) indexes.yKey === length; 3) both
    const onlyYellow = (xIndex === nStation) && (yIndex !== nStation)
    const onlyGreen = (yIndex === nStation) && (xIndex !== nStation)
    // const bothYellowAndGreen = (!onlyGreen) && (!onlyYellow)

    // console.log(xIndex, yIndex, nStation, onlyYellow, onlyGreen)

    // encoding parameter
    const cellSize = encoding.width - encoding.margin * 2
    const cellMargin = encoding.margin
    const cellSizeIncludingMargin = encoding.width
    const space = 1

    // obtain matrixWave
    const matrixWave = this.matrixContainer[matrixId]
    if (matrixWave.selected.flag) {
      return
    }
    // console.log('matrixId', matrixId)

    // obtain matrixG and its position
    const matrixG = this.matrixContainer[matrixId].matrixG

    // obtain parameters in horizon chart
    const horizonChart = matrixWave.horizonChart
    const nBinOption = { day: 61, hour: 24, weekday: 7 }
    const nBin = nBinOption[horizonChart.current]
    const horizonChartPerBarWidth = this.getHorizonChartPerBarWidth(matrixWave.horizonChart.current)
    const horizonWidth = (horizonChartPerBarWidth + encoding.margin) * nBin + 10 // 2 is the gap between matrix and horizon

    // shared parameter
    const matrixWaveSize = horizonWidth + nStation * cellSizeIncludingMargin + this.stationFrontLineLength + cellMargin + this.stationNameMaxLength

    // obtain rect parameters along x
    const x1 = -horizonWidth
    const y1 = cellSizeIncludingMargin * yIndex - space
    const width1 = matrixWaveSize
    const height1 = cellSize + 2 * space

    // obtain rect parameters along y
    const x2 = cellSizeIncludingMargin * xIndex - space
    const y2 = -horizonWidth
    const height2 = matrixWaveSize
    const width2 = cellSize + 2 * space

    if (onlyYellow) {
      this.setHoveredStations(null, this.currentMatrixWave.stations[yIndex])
      this.drawHightlightRect(x1, y1, width1, height1, matrixId, 'highlight-along-x')
      const branch1 = matrixWave.matrixG.outRoute.select(`#station${indexes.yKey}`)
      if (branch1) branch1.attr('style', 'visibility: visible;')
    } else if (onlyGreen) {
      this.setHoveredStations(this.currentMatrixWave.stations[xIndex], this.currentMatrixWave.stations[yIndex])
      this.drawHightlightRect(x2, y2, width2, height2, matrixId, 'highlight-along-y')
      const branch2 = matrixWave.matrixG.inRoute.select(`#station${indexes.xKey}`)
      if (branch2) branch2.attr('style', 'visibility: visible;')
    } else {
      this.setHoveredStations(this.currentMatrixWave.stations[xIndex], this.currentMatrixWave.stations[yIndex])
      const branch1 = matrixWave.matrixG.outRoute.select(`#station${indexes.yKey}`)
      if (branch1) branch1.attr('style', 'visibility: visible;')
      const branch2 = matrixWave.matrixG.inRoute.select(`#station${indexes.xKey}`)
      if (branch2) branch2.attr('style', 'visibility: visible;')
      this.drawHightlightRect(x1, y1, width1, height1, matrixId, 'highlight-along-x')
      this.drawHightlightRect(x2, y2, width2, height2, matrixId, 'highlight-along-y')
    }

    // const matrixWave = this.matrixContainer[matrixId]
    // if (matrixWave.selected.flag) return
    // const maxWidth = indexes.length * encoding.width
    // const xFlag = indexes.xKey === indexes.length
    // const yFlag = indexes.yKey === indexes.length
    // const matrixG = this.matrixContainer[matrixId].matrixG
    // // matrixG.flowG.attr('style', 'opacity: 0.2')
    // const inStationText = this.matrixContainer[matrixId].stationTextsObj.inStationTexts[indexes.xKey]
    // const outStationText = this.matrixContainer[matrixId].stationTextsObj.outStationTexts[indexes.yKey]
    // const horizonChart = matrixWave.horizonChart
    // const numObj = {day: 61, hour: 24, weekday: 7}
    // if (xFlag || yFlag) {
    //   if (xFlag && yFlag) return
    //   const horizonWidth = !flag ? (this.horizonChartPerBarWidth + encoding.margin) * (numObj[horizonChart.current]) + this.horizonChartMargin : 0
    //   const selectedRectWidth = horizonWidth + maxWidth + this.stationFrontLineLength + this.stationNameMaxLength
    //   const width = xFlag ? selectedRectWidth : encoding.width - encoding.margin
    //   const height = yFlag ? selectedRectWidth : encoding.height - encoding.margin
    //   const curStation = xFlag ? outStationText : inStationText
    //   const inOrOut = xFlag ? 'out' : 'in'
    //   const key = xFlag ? indexes.yKey : indexes.xKey
    //   const rectTransform = curStation.attr('transform').split('translate(')[1].split(')')[0].split(', ')
    //   const position = {x: Number(rectTransform[0]), y: Number(rectTransform[1])}
    //   const x = xFlag ? -horizonWidth : position.x - encoding.width + encoding.margin * 2
    //   const y = xFlag ? position.y - encoding.height + encoding.margin * 2 : -horizonWidth
    //   matrixG.selectedG.append('rect')
    //     .attr('width', width)
    //     .attr('height', height)
    //     .attr('transform', `translate(${x}, ${y})`)
    //     .attr('class', 'only-row-column')
    //     .attr('style', 'stroke:#d06150;stroke-width:1;fill:transparent;')
    //   matrixG.selectedG.append(() => matrixG.flowG.select(`#${inOrOut}${key}`).clone(true).node())
    //     .on('click', () => (matrixWave.selected.flag = !matrixWave.selected.flag))
    //   if (xFlag) {
    //     matrixWave.stationTextsObj.matrixField[indexes.yKey].forEach(item => {
    //       matrixG.selectedG.append(() => item.clone().node())
    //     })
    //     const targetBranch = matrixWave.matrixG.outRoute.select(`#station${indexes.yKey}`)
    //     if (targetBranch && !flag) targetBranch.attr('style', 'visibility: visible;')
    //   } else if (yFlag) {
    //     matrixWave.stationTextsObj.matrixField.forEach(item => {
    //       item.forEach((item1, key) => {
    //         if (key === indexes.xKey) matrixG.selectedG.append(() => item1.clone().node())
    //       })
    //     })
    //     const targetBranch = matrixWave.matrixG.inRoute.select(`#station${indexes.xKey}`)
    //     if (targetBranch && !flag) targetBranch.attr('style', 'visibility: visible;')
    //   }
    //   return
    // }
    // const rectTransform = rect.attr('transform').split('translate(')[1].split(')')[0].split(', ')
    // const position = {x: Number(rectTransform[0]), y: Number(rectTransform[1])}
    // const lastStyle = rect.attr('style')
    // const arr = lastStyle.split(';')
    // rect.attr('style', `${arr[0]};${arr[1]};stroke:#d06150;stroke-width:2;`)
    // const width = encoding.width * (indexes.length - indexes.xKey) + this.stationFrontLineLength + this.stationNameMaxLength
    // const height = encoding.height * (indexes.length - indexes.yKey) + this.stationFrontLineLength + this.stationNameMaxLength
    // matrixG.selectedG.append('rect')
    //   .attr('width', width)
    //   .attr('height', encoding.height - encoding.margin)
    //   .attr('transform', `translate(${position.x - 0.5}, ${position.y - 0.5})`)
    //   .attr('class', 'selected-rect')
    // matrixG.selectedG.append('rect')
    //   .attr('width', encoding.width - encoding.margin)
    //   .attr('height', height)
    //   .attr('transform', `translate(${position.x - 0.5}, ${position.y - 0.5})`)
    //   .attr('class', 'selected-rect')
    // matrixG.selectedG.append(() => rect.clone().node())
    //   .on('click', () => (matrixWave.selected.flag = !matrixWave.selected.flag))
    // matrixG.selectedG.append(() => matrixG.flowG.select(`#in${indexes.xKey}`).clone(true).node())
    // matrixG.selectedG.append(() => matrixG.flowG.select(`#out${indexes.yKey}`).clone(true).node())
  }

  public setHoveredStations (inStationId: number | null, outStationId: number | null) {
    // console.log(inStationId, outStationId)
    // 如果是null表示没有hover
    // 例如 inStationId = null, outStationId = null 表示没有hover任何一个station
    //      inStationId = 2301, outStationId = null 表示只hover了2301号station作为in （也就是绿色）
    Exploration.setInOutStation({in: inStationId, out: outStationId})
  }

  // 每次进入下一个小matrix的时候，要先调用此函数，删除highlight
  public leaveMatrix (rect: any, id: number) {
    this.setHoveredStations(null, null)
    // console.log('leave matrix')
    const matrixWave = this.matrixContainer[id]
    const selected = matrixWave.selected
    if (selected.flag) return
    const matrixG = matrixWave.matrixG
    matrixG.flowG.attr('style', 'opacity: 1')
    // if (rect) {
    //   const lastStyle = rect.attr('style')
    //   const arr = lastStyle.split(';')
    //   rect.attr('style', `${arr[0]};${arr[1]};`)
    // }
    // const length = this.indexedRoutes[matrixWave.routeId].stations.length
    // if (selected.xKey !== selected.yKey) {
    //   if (selected.xKey === length) {
    //     const targetBranch = matrixG.outRoute.select(`#station${selected.yKey}`)
    //     if (targetBranch) targetBranch.attr('style', 'visibility: hidden;')
    //   } else if (selected.yKey === length) {
    //     const targetBranch = matrixG.inRoute.select(`#station${selected.xKey}`)
    //     if (targetBranch) targetBranch.attr('style', 'visibility: hidden;')
    //   }
    // }
    const outBranch = matrixG.outRoute.selectAll('.branch')
    if (outBranch) outBranch.attr('style', 'visibility: hidden;')
    const inBranch = matrixG.inRoute.selectAll('.branch')
    if (inBranch) inBranch.attr('style', 'visibility: hidden;')
    matrixG.selectedG.selectAll('rect').remove()
    matrixG.selectedG.selectAll('text').remove()
    matrixG.selectedG.selectAll('g').remove()
  }

  // 画in部分的horizon chart等
  public drawIn (id: number, data: number[], svg: any, encoding: MatrixEncoding, checkinSequence: CheckSequenceEachRoute, type: string) {
    // console.log('debug', 'encoding', encoding)
    const horizonChartIn: any[] = []
    let x = 0
    // svg.on('mouseover', () => this.mouseoverHorizonChart(id, 'in', type))
    //   .on('mouseleave', () => this.mouseleaveHorizonChart(id))
    const outLineG = svg.append('g')
    data.forEach((item1, index) => {
      // const tempArr: any[] = []
      const dataForType = checkinSequence[item1][type]
      const maxTimes = d3.max(dataForType, (item: DayDataFlow | HourDataFlow | WeekdayDataFlow) => item.times) || 0
      const perBarWidth = this.getHorizonChartPerBarWidth(type)
      const perBarHeight = encoding.width - encoding.margin * 2
      let barY = encoding.height - perBarWidth - encoding.margin - this.horizonChartMargin
      const wholeHorizonChartWidth = dataForType.length * (perBarWidth + encoding.margin) + this.horizonChartMargin
      outLineG.append('polyline').attr('points', `${x + encoding.margin},${encoding.height - encoding.margin - this.horizonChartMargin / 2}
        ${x + encoding.margin + perBarHeight},${encoding.height - encoding.margin - this.horizonChartMargin / 2}
        ${x + encoding.margin + perBarHeight},${encoding.height - wholeHorizonChartWidth}`)
        .attr('style', 'fill:none;stroke:#EBF2FC;stroke-width:1;')
      if (index === 0) {
        const wholeHorizonChartHeight = encoding.width * data.length
        const outlineY1 = encoding.height - wholeHorizonChartWidth - this.horizonChartMargin
        const outlineY2 = outlineY1 + this.horizonChartMargin / 2
        outLineG.append('polyline').attr('points', `${encoding.margin},${outlineY2} ${encoding.margin},${outlineY1}
          ${encoding.margin + wholeHorizonChartHeight},${outlineY1}
          ${wholeHorizonChartHeight},${outlineY2}`)
          .attr('style', 'fill:none;stroke:#EBF2FC;stroke-width:1;')
        outLineG.append('text').text('CHECK IN').attr('style', 'fill:#E9EDF1;font-size:18px;')
          .attr('transform', `translate(${encoding.margin + wholeHorizonChartHeight / 2 - 40}, ${outlineY1 - 5})`)
      }

      const inColors = this.inColors
      const tempArr = svg.selectAll('.in-horizon-' + index)
        .data(dataForType)
        .enter()
        .append('g')
        .attr('class', 'in-horizon-' + index)
        .attr('id', (item) => `${item[type]} in${item1}`)
        .each(function(item, i) {
          const localG = d3.select(this)
          if (maxTimes !== 0) {
            const item = localG.datum()
            const maxTimesPerLine = (maxTimes + 1) / 3
            const times = Math.floor(item.times / maxTimesPerLine)
            const remainderWidth = item.times % maxTimesPerLine / maxTimesPerLine * perBarHeight
            const barX = x + encoding.margin + (perBarHeight - remainderWidth)
            if (times > 0) {
              localG.append('rect')
                .attr('height', perBarWidth)
                .attr('width', perBarHeight)
                .attr('transform', `translate(${x + encoding.margin}, ${barY})`)
                .attr('style', `fill:${inColors[times - 1]};`)
            }
            if (remainderWidth > 0) {
              localG.append('rect')
                .attr('height', perBarWidth)
                .attr('width', remainderWidth)
                .attr('transform', `translate(${barX}, ${barY})`)
                .attr('style', `fill:${inColors[times]};`)
            }
          }
          barY -= (perBarWidth + encoding.margin)
        })

      svg
        .append('rect')
        .attr('class', 'horizon-background')
        .attr('height', (encoding.height - encoding.margin * 2))
        .attr('width', (encoding.width - encoding.margin * 2))
        .attr('transform', `translate(${x + encoding.margin}, ${encoding.margin})`)
        .attr('style', 'fill:transparent;')
        .on('mouseenter', () => {
          this.enterMatrix(null, id, {xKey: index, yKey: data.length, length: data.length }) // hack parameter
        })
        .on('mouseleave', () => {
          this.leaveMatrix(null, id)
        })

      // dataForType.forEach((item, key) => {
      //   const tempG = svg.append('g').attr('class', `in${item[type]}`).attr('id', `in${item1}`)
      //   // tempG
      //   //   .append('rect')
      //   //   .attr('height', perBarWidth)
      //   //   .attr('width', perBarHeight)
      //   //   .attr('transform', `translate(${x + encoding.margin}, ${barY})`)
      //   //   .attr('style', 'fill:transparent;')
      //   if (maxTimes !== 0) {
      //     const maxTimesPerLine = maxTimes / 3
      //     const times = Math.floor(item.times / maxTimesPerLine)
      //     const remainderWidth = item.times % maxTimesPerLine / maxTimesPerLine * perBarHeight
      //     const barX = x + encoding.margin + (perBarHeight - remainderWidth)
      //     if (times > 0) {
      //       tempG
      //         .append('rect')
      //         .attr('height', perBarWidth)
      //         .attr('width', perBarHeight)
      //         .attr('transform', `translate(${x + encoding.margin}, ${barY})`)
      //         .attr('style', `fill:${this.inColors[times - 1]};`)
      //     }
      //     if (remainderWidth > 0) {
      //       tempG
      //         .append('rect')
      //         .attr('height', perBarWidth)
      //         .attr('width', remainderWidth)
      //         .attr('transform', `translate(${barX}, ${barY})`)
      //         .attr('style', `fill:${this.inColors[times]};`)
      //     }
      //   }
      //   barY -= (perBarWidth + encoding.margin)
      //   tempArr.push(tempG)
      // })

      horizonChartIn.push(tempArr)
      x += encoding.width
    })
    if (this.matrixContainer[id]) this.matrixContainer[id].stationTextsObj.horizonChartIn = horizonChartIn
    return horizonChartIn
  }

  // 画out部分的horizon chart等
  public drawOut (id: number, data: number[], svg: any, encoding: MatrixEncoding, checkoutSequence: CheckSequenceEachRoute, type: string) {
    const horizonChartOut: any[] = []
    let y = 0
    // svg.on('mouseover', () => this.mouseoverHorizonChart(id, 'out', type))
    //   .on('mouseleave', () => this.mouseleaveHorizonChart(id))
    const outLineG = svg.append('g')
    data.forEach((item1, index) => {
      // const tempArr: any[] = []
      const dataForType = checkoutSequence[item1][type]
      const maxTimes = d3.max(dataForType, (item: DayDataFlow | HourDataFlow | WeekdayDataFlow) => item.times) || 0
      const perBarWidth = this.getHorizonChartPerBarWidth(type)
      const perBarHeight = encoding.height - encoding.margin * 2
      let barX = encoding.width - perBarWidth - encoding.margin - this.horizonChartMargin
      const wholeHorizonChartWidth = dataForType.length * (perBarWidth + encoding.margin) + this.horizonChartMargin
      outLineG.append('polyline').attr('points', `${encoding.width - encoding.margin - this.horizonChartMargin / 2},${y + encoding.margin}
        ${encoding.width - encoding.margin - this.horizonChartMargin / 2},${y + encoding.margin + perBarHeight}
        ${encoding.width - wholeHorizonChartWidth},${y + encoding.margin + perBarHeight}`)
        .attr('style', 'fill:none;stroke:#EBF2FC;stroke-width:1;')
      if (index === 0) {
        const wholeHorizonChartHeight = encoding.height * data.length
        const outlineY1 = encoding.width - wholeHorizonChartWidth - this.horizonChartMargin
        const outlineY2 = outlineY1 + this.horizonChartMargin / 2
        outLineG.append('polyline').attr('points', `${outlineY2},${encoding.margin} ${outlineY1},${encoding.margin}
          ${outlineY1},${encoding.margin + wholeHorizonChartHeight}
          ${outlineY2},${wholeHorizonChartHeight}`)
          .attr('style', 'fill:none;stroke:#EBF2FC;stroke-width:1;')
        outLineG.append('text').text('CHECK OUT').attr('style', 'fill:#E9EDF1;font-size:18px;')
          .attr('transform', `translate(${outlineY1 - 5}, ${encoding.margin + wholeHorizonChartHeight / 2 + 40}) rotate(-90)`)
      }

      const outColors = this.outColors
      const tempArr = svg.selectAll('.out-horizon-' + index)
        .data(dataForType)
        .enter()
        .append('g')
        .attr('class', 'out-horizon-' + index)
        .attr('id', (item) => `${item[type]} out${item1}`)
        .each(function(item, i) {
          const localG = d3.select(this)
          if (maxTimes !== 0) {
            const item = localG.datum()
            const maxTimesPerLine = (maxTimes + 1) / 3
            const times = Math.floor(item.times / maxTimesPerLine)
            const remainderWidth = item.times % maxTimesPerLine / maxTimesPerLine * perBarHeight
            const barY = y + encoding.margin + (perBarHeight - remainderWidth)
            if (times > 0) {
              localG.append('rect')
                .attr('height', perBarHeight)
                .attr('width', perBarWidth)
                .attr('transform', `translate(${barX}, ${y + encoding.margin})`)
                .attr('style', `fill:${outColors[times - 1]};`)
            }
            if (remainderWidth > 0) {
              localG.append('rect')
                .attr('height', remainderWidth)
                .attr('width', perBarWidth)
                .attr('transform', `translate(${barX}, ${barY})`)
                .attr('style', `fill:${outColors[times]};`)
            }
          }
          barX -= (perBarWidth + encoding.margin)
        })

      svg
        .append('rect')
        .attr('class', 'horizon-background')
        .attr('height', (encoding.height - encoding.margin * 2))
        .attr('width', (encoding.width - encoding.margin * 2))
        .attr('transform', `translate(${encoding.margin}, ${y + encoding.margin})`)
        .attr('style', 'fill:transparent;')
        .on('mouseenter', () => {
          this.enterMatrix(null, id, {xKey: data.length, yKey: index, length: data.length }) // hack parameter
        })
        .on('mouseleave', () => {
          this.leaveMatrix(null, id)
        })

      // dataForType.forEach(item => {
      //   const tempG = svg.append('g').attr('class', `in${item[type]}`).attr('id', `out${item1}`)
      //   tempG
      //     .append('rect')
      //     .attr('height', perBarWidth)
      //     .attr('width', perBarHeight)
      //     .attr('transform', `translate(${barX}, ${y + encoding.margin})`)
      //     .attr('style', 'fill:transparent;')
      //   const maxTimesPerLine = maxTimes / 3
      //   const times = Math.floor(item.times / maxTimesPerLine)
      //   const remainderWidth = item.times % maxTimesPerLine / maxTimesPerLine * perBarHeight
      //   const barY = y + encoding.margin + (perBarHeight - remainderWidth)
      //   if (maxTimes !== 0) {
      //     if (times > 0) {
      //       tempG
      //         .append('rect')
      //         .attr('height', perBarHeight)
      //         .attr('width', perBarWidth)
      //         .attr('transform', `translate(${barX}, ${y + encoding.margin})`)
      //         .attr('style', `fill:${this.outColors[times - 1]};`)
      //     }
      //     tempG
      //       .append('rect')
      //       .attr('height', remainderWidth)
      //       .attr('width', perBarWidth)
      //       .attr('transform', `translate(${barX}, ${barY})`)
      //       .attr('style', `fill:${this.outColors[times]};`)
      //   }
      //   barX -= (perBarWidth + encoding.margin)
      //   tempArr.push(tempG)
      // })

      horizonChartOut.push(tempArr)
      y += encoding.height
    })
    if (this.matrixContainer[id]) this.matrixContainer[id].stationTextsObj.horizonChartOut = horizonChartOut
    return horizonChartOut
  }

  // 鼠标在horizon chart上方时，触发的函数
  // public mouseoverHorizonChart (id: number, type: string, dataType: string) {
  //   const matrixWave = this.matrixContainer[id]
  //   const horizonSelected = matrixWave.horizonSelected[type]
  //   if (horizonSelected.flag) return
  //   const length = dataType === 'day' ? 61 : (dataType === 'hour' ? 24 : 7)
  //   const svg = type === 'in' ? matrixWave.matrixG.inG : matrixWave.matrixG.outG
  //   const mousePosition = d3.mouse(svg.node())
  //   const mouseX = type === 'in' ? mousePosition[1] : mousePosition[0]
  //   const mouseY = type === 'in' ? mousePosition[0] : mousePosition[1]
  //   const xKey = Math.floor((this.horizonChatrWidth - this.horizonChartMargin - mouseX) / (this.horizonChartPerBarWidth + this.matrixEncoding.margin))
  //   const yKey = Math.floor(mouseY / this.singleMatrixWidth)
  //   const horizonBarArray = type === 'in' ? matrixWave.stationTextsObj.horizonChartIn
  //     : matrixWave.stationTextsObj.horizonChartOut
  //   if (xKey >= length || xKey < 0 || yKey >= matrixWave.stations.length || (xKey === horizonSelected.xKey && yKey === horizonSelected.yKey)) return
  //   const target = horizonBarArray[yKey][xKey]
  //   this.mouseleaveHorizonChart(id)
  //   this.mouseenterHorizonChart(id, target, type)
  //   const tempX = type === 'in' ? yKey : matrixWave.stations.length
  //   const tempY = type === 'in' ? matrixWave.stations.length : yKey
  //   this.leaveMatrix(null, id)
  //   this.enterMatrix(null, id, {xKey: tempX, yKey: tempY, length: matrixWave.stations.length}, this.matrixEncoding, true)
  //   horizonSelected.xKey = xKey
  //   horizonSelected.yKey = yKey
  // }

  // mouseOver会确定到底进入了horizon chart的哪个bar，然后触发此函数
  // public mouseenterHorizonChart (id: number, target: any, type: string) {
  //   if (!target) return
  //   const matrixWave = this.matrixContainer[id]
  //   const eleId = target.attr('id')
  //   const eleClass = target.attr('class')
  //   const targetG = type === 'in' ? matrixWave.matrixG.inG : matrixWave.matrixG.outG
  //   matrixWave.matrixG.inG.select('#content').attr('style', 'opacity:0.2;')
  //   matrixWave.matrixG.outG.select('#content').attr('style', 'opacity:0.2;')
  //   targetG.select('#content').selectAll(`#${eleId}`).each(function (this, d, i) {
  //     const node = d3.select(this).clone(true)
  //     targetG.select(`#${type}`).append(() => node.node())
  //   })
  //   matrixWave.matrixG.inG.select('#content').selectAll(`.${eleClass}`).each(function (this, d, i) {
  //     const node = d3.select(this).clone(true)
  //     matrixWave.matrixG.inG.select('#in').append(() => node.node())
  //   })
  //   matrixWave.matrixG.outG.select('#content').selectAll(`.${eleClass}`).each(function (this, d, i) {
  //     const node = d3.select(this).clone(true)
  //     matrixWave.matrixG.outG.select('#out').append(() => node.node())
  //   })
  //   targetG.select(`#${type}`).append(() => target.clone(true).node())
  // }

  // 每次进入下一个horizon chart的bar的时候，要先调用此函数，删除highlight
  // public mouseleaveHorizonChart (id: number) {
  //   const matrixWave = this.matrixContainer[id]
  //   matrixWave.matrixG.inG.select('#content').attr('style', 'opacity:1;')
  //   matrixWave.matrixG.outG.select('#content').attr('style', 'opacity:1;')
  //   matrixWave.matrixG.inG.select('#in').selectAll('g').remove()
  //   matrixWave.matrixG.outG.select('#out').selectAll('g').remove()
  //   this.leaveMatrix(null, id)
  // }

  // 画hover在station上后，显示的树状branch
  public drawBranches (id: number, routeId: number, stationIDs: number[], stationTextsG: any[],
    encoding: MatrixEncoding, reverse: boolean, type: string, matrixG: any, barChart: BarchartInMatrix) {
    const svg = type === 'in' ? matrixG.inRoute : matrixG.outRoute
    let branchesSum = 0
    stationTextsG.forEach((item, stationIndex) => {
      let stationTranslate = [0, 0]
      const textPositionStr = item.attr('transform').split('translate(')[1].split(')')[0].split(', ')
      stationTranslate = [Number(textPositionStr[0]), Number(textPositionStr[1])]
      let [linex1, liney1, linex2, liney2, textx, texty, length, direction] = [0, 0, 0, 0, 0, 0, 0, 0]
      let [textAnchor, textTransform] = ['', '']
      const radius = 4
      if (type === 'in') {
        linex1 = stationTranslate[0] - encoding.width / 2 + encoding.margin
        liney1 = stationTranslate[1] + this.stationFrontLineLength + this.stationNameMaxLength + this.singleMatrixWidth - 20
        linex2 = linex1
        liney2 = liney1 + this.stationPostLineLength
        textx = linex2 - this.stationPostLineLength + 22
        texty = liney2 + radius - 1.5
        textAnchor = 'end'
        direction = reverse ? 4 : 3
      } else if (type === 'out') {
        linex1 = stationTranslate[0] + this.stationFrontLineLength + this.stationNameMaxLength + this.singleMatrixWidth - 20
        liney1 = stationTranslate[1] - encoding.height / 2 + encoding.margin
        linex2 = linex1 + this.stationPostLineLength
        liney2 = liney1
        textx = linex2 + this.stationPostLineLength - 22
        texty = liney2 + radius - 1.5
        textTransform = `rotate(-90, ${linex2}, ${liney2})`
        direction = reverse ? 1 : 2
      } else return
      // const branchDataForCurrentStation = this.getStationBranches(stationIndex, routeId, stationIDs)
      const totalTransferNum = barChart['transfer' + type][stationIDs[stationIndex]]
      const newBranchDataForCurrentStation = this.getNewStationBranches(stationIndex, routeId, stationIDs, type)
      // console.log('debug', newBranchDataForCurrentStation, totalTransferNum, detailTransfer)

      branchesSum += newBranchDataForCurrentStation.length
      const branchG = svg.append('g')
        .attr('style', 'visibility:hidden;')
        .attr('id', `station${stationIndex}`)
        .attr('class', 'branch')
      const circleColor = type === 'in' ? ['#d1ecac', '#94b366'] : ['#ffe2ae', '#cca766']

      newBranchDataForCurrentStation.forEach((branch: Branch, index) => {
        const rid = branch.route_id
        const routeG = branchG.append('g').attr('id', `route${rid}`)
        const branchText = this.indexedRoutes[rid].name
        branchG.insert('line', 'g').attr('id', `line${rid}`).attr('x1', linex1).attr('y1', liney1)
          .attr('x2', linex2).attr('y2', liney2).attr('stroke', circleColor[0])
        routeG.append('circle').attr('cx', linex2).attr('cy', liney2).attr('r', radius).attr('fill', circleColor[0])

        const sectorG = routeG.append('path').attr('d', this.returnSectorPathD({x: linex2, y: liney2}, radius, totalTransferNum, branch.records.length))
          .attr('fill', circleColor[1])
        const textN = routeG.append('text')
          .attr('x', textx)
          .attr('y', texty)
          .attr('text-anchor', textAnchor)
          .attr('class', 'branch-name')
          .attr('style', `fill:${circleColor[1]};cursor: pointer;font-size: ${this.branchTextFontSize}px; font-weight: bold;`)
          .on('click', () => this.branchRouteClick(branchG, index, id, !reverse, branch, direction, stationIndex))
          .text(branchText)
        const routeMatrix = this.getTransformMatrix(textN.node(), 'y', reverse)
        textN.attr('transform', `${textTransform} ${routeMatrix}`)
        if (type === 'in') {
          texty += 13
          liney1 += 13
          liney2 += 13
        } else if (type === 'out') {
          sectorG.attr('transform', `rotate(-90, ${linex2}, ${liney2})`)
          textx += 13
          linex1 += 13
          linex2 += 13
          textTransform = `rotate(-90, ${linex2}, ${liney2})`
        }
      })

      // branchDataForCurrentStation.forEach((dataItem, dataKey) => {
      //   const routeG = branchG.append('g').attr('id', `route${dataItem.route_id}`)
      //   const branchText = 'Route #' + this.indexedRoutes[dataItem.route_id].name
      //   branchG.append('line').attr('id', `line${dataItem.route_id}`).attr('x1', linex1).attr('y1', liney1)
      //     .attr('x2', linex2).attr('y2', liney2).attr('stroke', circleColor[0])
      //   routeG.append('circle').attr('cx', linex2).attr('cy', liney2).attr('r', radius).attr('fill', circleColor[0])

      //   const sectorG = routeG.append('path').attr('d', this.returnSectorPathD({x: linex2, y: liney2}, radius, 10, 9))
      //     .attr('fill', circleColor[1])
      //   const textN = routeG.append('text')
      //     .attr('x', textx)
      //     .attr('y', texty)
      //     .attr('text-anchor', textAnchor)
      //     .attr('class', 'branch-name')
      //     .attr('style', `fill:${circleColor[1]};cursor: pointer;font-size: ${this.branchTextFontSize}px; font-weight: bold;`)
      //     .on('click', () => this.branchRouteClick(branchG, dataKey, id, !reverse, dataItem, direction, stationIndex))
      //     .text(branchText)
      //   const routeMatrix = this.getTransformMatrix(textN.node(), 'y', reverse)
      //   textN.attr('transform', `${textTransform} ${routeMatrix}`)
      //   if (type === 'in') {
      //     texty += 13
      //     liney1 += 13
      //     liney2 += 13
      //   } else if (type === 'out') {
      //     sectorG.attr('transform', `rotate(-90, ${linex2}, ${liney2})`)
      //     textx += 13
      //     linex1 += 13
      //     linex2 += 13
      //     textTransform = `rotate(-90, ${linex2}, ${liney2})`
      //   }
      // })
    })
    if (type === 'in') this.branchesSumLeft = branchesSum
    else this.branchesSumRight = branchesSum
  }

  // 获取某个station上的分支
  // public getStationBranches (id: number, routeId: number, stations: number[]) {
  //   const currentRouteBranches = Exploration.indexedRouteBranches[routeId]
  //   return currentRouteBranches ? currentRouteBranches.branches.filter(item => {
  //     return item.from_station_id === stations[id]
  //   }) : []
  // }

  public getNewStationBranches (id: number, routeId: number, stations: number[], inOrOut: string) {
    const currentRouteBranches = Exploration.indexedRouteBranches[routeId]
    return currentRouteBranches ? currentRouteBranches['check' + inOrOut + 'Branches'].filter(item => {
      return item.from_station_id === stations[id]
    }) : []
  }

  // 树状branches中分支的点击事件处理
  public branchRouteClick (branchG: any, index: number, id: number, reverse: boolean, branch: Branch, direction: number, key: number) {
    this.addNewMartix(id, reverse, branch, direction, key)
    const target = branchG.select(`route${index}`).clone(true)
    const linesG = branchG.selectAll('line').clone(true)
  }

  // 计算扇形的path
  public returnSectorPathD (center: Point, r: number, max: number, num: number) {
    let ratio = num / max
    if (ratio === 1) ratio = 0.999999
    const largeArcFlag = ratio >= 0.5 ? 1 : 0
    const endPointX = center.x + Math.sin(2 * Math.PI * ratio) * r
    const endPointY = center.y - Math.cos(2 * Math.PI * ratio) * r
    return `M${center.x} ${center.y} L${center.x} ${center.y - r}
      A${r} ${r} 0 ${largeArcFlag} 1 ${endPointX} ${endPointY}`
  }

  // 删除树状branches
  public deleteBranchText (routeG: any) {
    routeG.selectAll('g').remove()
    routeG.selectAll('text').remove()
    routeG.selectAll('line').remove()
  }

  // 获取transform的matrix，用于翻转元素
  public getTransformMatrix (node: any, xy: string, reverse: boolean) {
    if (!reverse) return ''
    const bbox = node.getBBox()
    let par = 0
    if (xy === 'x') par = 2 * (bbox.x + bbox.width / 2)
    else if (xy === 'y') par = 2 * (bbox.y + bbox.height / 2)
    const result = `matrix(1, 0, 0, -1, 0, ${par})`
    return result
  }

  // 将传入的MatrixWave移到中心
  public async focusToNewMatrix (newMatrixWave: MatrixWave) {
    const lastSize = this.panZoomTiger.getSizes()
    const mainGSizeWidth = newMatrixWave.matrixG.mainG.node().getBoundingClientRect().width
    const outSizeWidth = newMatrixWave.matrixG.outG.node().getBoundingClientRect().width
    await this.zoomAnimation(this.initialZoom, 500,
      {x: lastSize.width / 2, y: lastSize.height / 2} as Position)
    const [x, y] = this.computePan(newMatrixWave, [lastSize.realZoom, mainGSizeWidth, outSizeWidth])
    await this.panAnimation({x, y} as Position, 500)
  }

  // 计算移动到此MatrixWave的x和y
  public computePan (newMatrixWave: MatrixWave, tempArr: number[] = []) {
    const svgSize = this.panZoomTiger.getSizes()
    let mainGSizeWidth = newMatrixWave.matrixG.mainG.node().getBoundingClientRect().width
    let outSizeWidth = newMatrixWave.matrixG.outG.node().getBoundingClientRect().width
    if (tempArr.length > 0) {
      mainGSizeWidth = tempArr[1] / tempArr[0] * svgSize.realZoom
      outSizeWidth = tempArr[2] / tempArr[0] * svgSize.realZoom
    }
    const outBox = newMatrixWave.matrixG.outG.node().getBBox()
    const sizeRatio = outSizeWidth / (outBox.width + outBox.height) * Math.sqrt(2)
    const positionX = (newMatrixWave.x - newMatrixWave.y) / Math.sqrt(2)
    const positionY = (newMatrixWave.x + newMatrixWave.y) / Math.sqrt(2)
    const x = -(positionX * sizeRatio) + svgSize.width / 2
    const y = -(positionY * sizeRatio + mainGSizeWidth / 2) + svgSize.height / 2
    return [x, y]
  }

  // zoom的动画
  public zoomAnimation (zoom: number, duration: number, position: Position) {
    const curZoom = this.panZoomTiger.getZoom()
    const distance = zoom - curZoom
    // eslint-disable-next-line @typescript-eslint/no-this-alias
    const that = this
    const startTime = Date.now()
    return new Promise(resolve => {
      requestAnimationFrame(function step() {
        const per = Math.min(1.0, (Date.now() - startTime) / duration)
        that.panZoomTiger.zoomAtPoint(curZoom + distance * per, {x: position.x, y: position.y})
        if (per < 1.0) requestAnimationFrame(step)
        else if (per === 1.0) resolve()
      })
    })
  }

  // pan的动画
  public panAnimation (position: Position, duration: number) {
    const startTime = Date.now()
    const curPosition = this.panZoomTiger.getPan()
    const distance = {x: position.x - curPosition.x, y: position.y - curPosition.y}
    // eslint-disable-next-line @typescript-eslint/no-this-alias
    const that = this
    return new Promise(resolve => {
      requestAnimationFrame(function step() {
        const per = Math.min(1.0, (Date.now() - startTime) / duration)
        that.panZoomTiger.pan({
          x: curPosition.x + per * distance.x,
          y: curPosition.y + per * distance.y
        })
        if (per < 1.0) requestAnimationFrame(step)
        else if (per === 1.0) resolve()
      })
    })
  }

  // 删除大matrix
  public deleteMatrix (matrixWave: MatrixWave, parIndex1 = -1, parMatrixWave1 = null as any) {
    if (!matrixWave) return
    const parMatrixWave = parMatrixWave1 || this.matrixContainer[parIndex1]
    const parIndex = parMatrixWave1 ? this.matrixContainer.indexOf(parMatrixWave1) : parIndex1
    if (parIndex !== -1) {
      const stationIDs = this.indexedRoutes[parMatrixWave.routeId].stations
      if (parMatrixWave.direction === 1 || parMatrixWave.direction === 2) {
        this.drawBranches(parIndex, parMatrixWave.routeId, stationIDs, parMatrixWave.stationTextsObj.outStationTexts,
          this.matrixEncoding, parMatrixWave.reverse, 'out', parMatrixWave.matrixG, parMatrixWave.barChart)
        parMatrixWave.sibling.right = {id: -1, branch: null}
      } else if (parMatrixWave.direction === 3 || parMatrixWave.direction === 4) {
        this.drawBranches(parIndex, parMatrixWave.routeId, stationIDs, parMatrixWave.stationTextsObj.inStationTexts,
          this.matrixEncoding, parMatrixWave.reverse, 'in', parMatrixWave.matrixG, parMatrixWave.barChart)
        parMatrixWave.sibling.left = {id: -1, branch: null}
      } else if (parMatrixWave.direction === 0) {
        if (matrixWave.direction === 2) {
          this.drawBranches(parIndex, parMatrixWave.routeId, stationIDs, parMatrixWave.stationTextsObj.outStationTexts,
            this.matrixEncoding, parMatrixWave.reverse, 'out', parMatrixWave.matrixG, parMatrixWave.barChart)
          parMatrixWave.sibling.right = {id: -1, branch: null}
        } else if (matrixWave.direction === 3) {
          this.drawBranches(parIndex, parMatrixWave.routeId, stationIDs, parMatrixWave.stationTextsObj.inStationTexts,
            this.matrixEncoding, parMatrixWave.reverse, 'in', parMatrixWave.matrixG, parMatrixWave.barChart)
          parMatrixWave.sibling.left = {id: -1, branch: null}
        }
      }
    }
    const index = this.matrixContainer.indexOf(matrixWave)
    matrixWave.matrixG.mainG.remove()
    matrixWave.matrixG.lineG.remove()
    this.matrixContainer.filter(item => item && item.parId === index)
      .forEach(item => {
        this.deleteMatrix(item)
        Exploration.deleteMatrixHighLightRoute(item.routeId)
      })
    delete this.matrixContainer[index]
    this.updateOverview()
  }

  // 获取从stime到etime间的时间字符串，如："2020-01-01" 到 "2020-04-01"
  public getDiffDate (stime: string, etime: string) {
    const diffdate: string[] = []
    let i = 0
    while (stime <= etime) {
      diffdate[i] = stime
      const stimTs = new Date(stime).getTime()
      const nextDate = stimTs + (24 * 60 * 60 * 1000)
      const nextDatesY = new Date(nextDate).getFullYear() + '-'
      const nextDatesM = (new Date(nextDate).getMonth() + 1 < 10) ? '0' + (new Date(nextDate).getMonth() + 1) + '-' : (new Date(nextDate).getMonth() + 1) + '-'
      const nextDatesD = (new Date(nextDate).getDate() < 10) ? '0' + new Date(nextDate).getDate() : new Date(nextDate).getDate()
      stime = nextDatesY + nextDatesM + nextDatesD;
      i++
    }
    return diffdate
  }

  // 获取小matrix中的流量的最大值，用来当小matrix透明度大小的分母
  public getMaxFlowNum (route: Route) {
    // if (!route) return 1
    // const matrix = route.matrix
    // let max = 1
    // for (const i in matrix) {
    //   for (const j in matrix[i]) {
    //     max = matrix[i][j] > max ? matrix[i][j] : max
    //   }
    // }
    // return max
    return 40
  }

  // 更新overview左半部分的视图
  public updateOverview () {
    if (this.overviewTiger) this.overviewTiger.destroy()
    d3.select('#overview').selectAll('g').remove()
    d3.select('#overview').selectAll('polygon').remove()
    d3.select('#overview').selectAll('text').remove()
    this.overviewContainer = []
    this.matrixContainer.forEach((item, key) => {
      if (item) {
        const parentID = this.overviewContainer.findIndex(item1 => item1.matrixWaveID === item.parId)
        this.drawOverviewMatrix(parentID, d3.select('#overview'), 'fill', key)
      }
    })
    this.overviewContainer.forEach((item, key) => {
      const direction = this.matrixContainer[item.matrixWaveID].direction
      if (direction !== 0 && item.childrenID.num === 0) {
        let type = 'dash-left'
        if (direction === 1 || direction === 2) type = 'dash-right'
        const dashDirection = direction === 1 ? 2 : (direction === 2 ? 1 : (direction === 3 ? 4 : 3))
        this.drawOverviewMatrix(key, d3.select('#overview'), type, -1, dashDirection)
      } else if (direction === 0 && item.childrenID.num < 2) {
        if (item.childrenID.right === -1) this.drawOverviewMatrix(key, d3.select('#overview'), 'dash-right', -1, 2)
        if (item.childrenID.left === -1) this.drawOverviewMatrix(key, d3.select('#overview'), 'dash-left', -1, 3)
      }
    })
    this.overviewTiger = svgPanZoom(this.overview, {
      zoomEnabled: false,
      panEnabled: true,
      controlIconsEnabled: false,
      fit: false,
      center: true
    })
    this.overviewTiger.zoom(0.6)
  }

  // 画overview左半部分的matrix
  public drawOverviewMatrix (parentID: number, parentG: any, type: string, key: number, direction = 0) {
    const parentOverview = this.overviewContainer[parentID]
    const size = this.overviewMatrixSize
    const target = this.matrixContainer[key]
    const targetDirection = target ? target.direction : direction
    let x = 0
    let y = 0
    const fillColor = type === 'fill' ? '#DCE7F2' : 'transparent'
    const curID = direction === 0 ? this.overviewContainer.length : -1
    if (parentOverview) {
      if (targetDirection === 1) {
        x = parentOverview.x + this.overviewMatrixSize + this.overviewMatrixMargin
        y = parentOverview.y - this.overviewMatrixSize - this.overviewMatrixMargin
        parentOverview.childrenID.right = curID
      } else if (targetDirection === 2) {
        x = parentOverview.x + this.overviewMatrixSize + this.overviewMatrixMargin
        y = parentOverview.y + this.overviewMatrixSize + this.overviewMatrixMargin
        parentOverview.childrenID.right = curID
      } else if (targetDirection === 3) {
        x = parentOverview.x - this.overviewMatrixSize - this.overviewMatrixMargin
        y = parentOverview.y + this.overviewMatrixSize + this.overviewMatrixMargin
        parentOverview.childrenID.left = curID
      } else if (targetDirection === 4) {
        x = parentOverview.x - this.overviewMatrixSize - this.overviewMatrixMargin
        y = parentOverview.y - this.overviewMatrixSize - this.overviewMatrixMargin
        parentOverview.childrenID.left = curID
      }
      if (direction === 0) parentOverview.childrenID.num++
    }
    const tempG = parentG.append('polygon').attr('fill', fillColor).attr('id', `matrix${key}`)
      .attr('points', `${x - size},${y} ${x},${y + size} ${x + size},${y} ${x},${y - size}`)
    if (type === 'fill') tempG.on('click', () => this.focusToNewMatrix(target))
    else {
      const text = String(type === 'dash-left' ? this.branchesSumLeft : this.branchesSumRight)
      const color = type === 'dash-left' ? this.inColors[2] : this.outColors[2]
      tempG.attr('style', `stroke:${color};stroke-dasharray:10,10;`)
      parentG.append('text').text(text).attr('style', `fill:${color};font-size:30px;font-weight:bold;`)
        .attr('transform', `translate(${x - 9 * text.length}, ${y + 5})`)
    }
    if (key === this.curFocusId) tempG.attr('style', 'stroke:#7ABDE6;stroke-width:3;')
    if (direction === 0) this.overviewContainer.push({parentID, matrixWaveID: key, x, y, type, childrenID: {right: -1, left: -1, num: 0}})
  }

  // 确定当前离中心最近的大matrix
  public changeFocusMatrix (newPan) {
    if (!this.panZoomTiger) return
    let curFocusId = -1
    let minDis = Number.MAX_SAFE_INTEGER
    this.matrixContainer.forEach((item, key) => {
      const tempPan = this.computePan(item)
      const tempDis = Math.pow(tempPan[0] - newPan.x, 2) + Math.pow(tempPan[1] - newPan.y, 2)
      if (minDis > tempDis) {
        curFocusId = key
        minDis = tempDis
      }
    })
    if (this.curFocusId !== curFocusId) this.curFocusId = curFocusId
  }

  @Watch('routeMatrixWhenSearching')
  private updateFlowCells () {
    if (this.searching && this.routeMatrixWhenSearching) {
      const flowMatrix = this.routeMatrixWhenSearching
      const maxFlowNum = 200 // todo hack debug
      d3.selectAll('.flow-cell')
        .each(function() {
          const idstr = d3.select(this).attr('id')
          const [inNstr, outNstr] = idstr.split('-')[1].split('_')
          let opactiy = 0
          if (flowMatrix[+inNstr] && flowMatrix[+inNstr][+outNstr]) {
            opactiy = flowMatrix[+inNstr][+outNstr] / maxFlowNum
          }
          d3.select(this).attr('fill-opacity', opactiy)
        })
    }
  }

  // 如果当前聚焦的大matrix变了，要更新overview中的focus
  @Watch('curFocusId')
  private _onCurFocusIdChange (newVal, oldVal) {
    const last = d3.select(this.overview).select(`#matrix${oldVal}`)
    const current = d3.select(this.overview).select(`#matrix${newVal}`)
    if (last) last.attr('style', '')
    if (current) current.attr('style', 'stroke:#7ABDE6;stroke-width:3;')
    Exploration.toggleMatrixHighLightRoute(this.matrixContainer[newVal].routeId)
  }

  // selectedRoute变化了，要对应更新他的stationIDS
  @Watch('selectedRoute')
  private _onSelectedRouteChange () {
    if (this.selectedRoute) {
      this.selectedStationIDs = _.clone(this.selectedRoute.stations)
    } else {
      this.selectedStationIDs = []
    }
  }

  @Watch('routeStationsWhenSearching')
  private _onRouteStationsWhenSearchingChanged () {
    console.log('_onRouteStationsWhenSearchingChanged', this.routeStationsWhenSearching)
    if (this.routeStationsWhenSearching.length > 0) {
      this.selectedStationIDs = this.routeStationsWhenSearching
    }
  }

  // selectedStationIDs变了要完全更新所有matrix
  @Watch('selectedStationIDs')
  private _onSelectedStationIDsChange () {
    if (this.selectedStationIDs) {
      this.matrixContainer.forEach((item: MatrixWave, key) => {
        // console.log('debug', item, key)
        if (!item) return
        item.matrixG.mainG.remove()
        item.matrixG.lineG.remove()
        delete this.matrixContainer[key]
      })
      const reverse = this.reverse
      const routeId = this.selectedRoute ? this.selectedRoute.id : 0
      const newMatrixG = this.appendG(this.matrixContainer.length, reverse)
      const curBranch = { // eslint-disable-next-line
        route_id: this.selectedRoute ? this.selectedRoute.id : -1
      } as Branch
      const tempObj = this.drawMatrixWave(this.matrixContainer.length, false, newMatrixG, this.selectedStationIDs,
        { x: 0, y: 0 } as MatrixWave, 0, curBranch, -1)
      const fatherMatrixWave = {
        ...tempObj,
        reverse,
        routeId,
        parId: -1,
        direction: 0,
        stations: this.selectedStationIDs,
        selected: {xKey: -1, yKey: -1, flag: false},
        sibling: {left: {id: -1, branch: null}, right: {id: -1, branch: null}},
        horizonSelected: {in: {xKey: -1, yKey: -1, flag: false}, out: {xKey: -1, yKey: -1, flag: false}}
      }
      this.matrixContainer.push(fatherMatrixWave)
      Exploration.emptyMatrixHighLightRoute()
      Exploration.addMatrixHighLightRoute(routeId)
      const panZoomTiger = svgPanZoom(this.svg, {
        zoomEnabled: true,
        controlIconsEnabled: true,
        fit: true,
        center: true,
        minZoom: 0.1,
        maxZoom: 100,
        onPan: this.changeFocusMatrix
      })
      this.initialZoom = panZoomTiger.getZoom()
      panZoomTiger.updateBBox()
      panZoomTiger.resize()
      panZoomTiger.contain()
      panZoomTiger.center()
      this.panZoomTiger = panZoomTiger
      this.curFocusId = this.matrixContainer.length - 1
    }

    this.updateConflictHint()
  }

  // matrixContainer数量变化的时候要更新overview
  @Watch('matrixContainer')
  private _onMatrixContainerChange () {
    this.updateOverview()
  }
}
</script>

<style lang="scss">
$backgroundColor: lighten(#fff, 12%);
$columnSpacing: 10px;

.matrix-container {
  position: relative;
  width: 40%;
  height: calc(100% - 40px - 20px);
  z-index: 1;
  margin: 20px 0;
  background-color: $backgroundColor;
  border: 1px solid #ccc;
  border-bottom-left-radius: 10px;
  border-top-left-radius: 10px;
  box-shadow: 0 0 20px #ccc;
  padding: 10px 15px;
  transition: transform 300ms;

  .station-name {
    cursor: pointer;
    fill: #777;
  }

  .selected-rect {
    stroke: #d06150;
    stroke-width: 1;
    fill: transparent;
    pointer-events: none;
  }

  .conflict-hint {
    stroke-width: 1;
  }

  // .highlight-rect {
  //   pointer-events: none;
  // }
}
.overview {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 100px;
  padding: 10px;
  background-color: #F0FAFF;
  display: flex;
  width: 202px;
  border-top: 1px solid #eee;
  border-right: 1px solid #eee;

  .left {
    flex: 0 0 200px;
    height: 100px;
    border: 1px solid #84C1E7;
    border-radius: 5px;
    background-color: #ffffff;
  }
  .right {
    flex: 1 1 0;
    display: flex;
    flex-direction: column;
    padding: 0 10px;

    svg {
      width: 100%;
    }
  }
}

.route-name-info {
  position: absolute;
  top: calc(100% - 150px);
  left: 10px;
  font-weight: bold;
  font-size: 20px;
  color: #92A6B9;
}
</style>
