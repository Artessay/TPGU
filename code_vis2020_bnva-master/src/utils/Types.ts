import { Attribute, AttributeType } from "@/utils/Attribute";
import _ from 'lodash'
import { Candidate } from "@/store/modules/CandidatesList";
import { linspace } from "@/utils/Helper";
import { csn } from "@/utils/Formatter";
import {Branch, RouteBranches, Transfer} from "@/store/modules/Exploration";

export interface BarchartInMatrix {
  checkin: {
    data: {[station: number]: number}
    maxTimes: number
  }
  checkout: {
    data: {[station: number]: number}
    maxTimes: number
  }
  transferin: {[sid: number]: number}
  transferout: {[sid: number]: number}
  transferoutDetail: {[sid: number]: {
    [rid: number]: number[]
  }}
  transferinDetail: {
    [sid: number]: {
      [rid: number]: number[]
    }
  }
}

export interface RouteTransfers {
  checkinCount: {[sid: number]: number}
  checkoutCount: {[sid: number]: number}
  checkinDetail: {
    [sid: number]: {
      [rid: number]: number[]
    }
  }
  checkoutDetail: {
    [sid: number]: {
      [rid: number]: number[]
    }
  }
}

export function RouteTransfersToRouteBranches(
  routesTransfers: {[id: number]: RouteTransfers},
  indexedTransfers: {[id: number]: Transfer}) {
  const indexedRouteBranches: { [id: number]: RouteBranches} = {}
  _.each(routesTransfers, (transfer, rid) => {
    const branches : RouteBranches = {
      route_id: parseInt(rid),
      checkinBranches: [],
      checkoutBranches: [],
      branches: []
    }
    // checkin
    _.each(transfer.checkinDetail, (checkin, sid) => {
      if (Object.keys(checkin).length > 0) {
        _.each(checkin, (recordIDs, routeID) => {
          const branch : Branch = {
            route_id: parseInt(routeID),
            from_stop_num: indexedTransfers[recordIDs[0]].to_stop_num,
            to_stop_num: indexedTransfers[recordIDs[0]].from_stop_num,
            from_station_id: indexedTransfers[recordIDs[0]].to_station_id,
            to_station_id: indexedTransfers[recordIDs[0]].from_station_id,
            records: recordIDs
          }
          branches.checkinBranches.push(branch)
        })
      }
    })
    _.each(transfer.checkoutDetail, (checkout, sid) => {
      if (Object.keys(checkout).length > 0) {
        _.each(checkout, (recordIDs, routeID) => {
          const branch : Branch = {
            route_id: parseInt(routeID),
            from_stop_num: indexedTransfers[recordIDs[0]].from_stop_num,
            to_stop_num: indexedTransfers[recordIDs[0]].to_stop_num,
            from_station_id: indexedTransfers[recordIDs[0]].from_station_id,
            to_station_id: indexedTransfers[recordIDs[0]].to_station_id,
            records: recordIDs
          }
          branches.checkoutBranches.push(branch)
        })
      }
    })
    indexedRouteBranches[rid] = branches
  })
  return indexedRouteBranches
}

export enum Stage {
  EXPLORATION,
  MANIPULATION,
  EVALUATION,
  HIGHLIGHT
}

export interface ProjectionPoint {
  id: number
  checkin: number
  checkout: number
  px: number
  py: number
}

export class AttributeGroup {
  private _width = 0
  public _groupAttrs : Attribute[] = []

  constructor (attrs: Attribute[]) {
    this._groupAttrs = attrs
    this.takeOwnerShip(attrs)
  }

  takeOwnerShip (attrs : Attribute[]) {
    _.each(
      attrs,
      attr => {
        attr.group = this
        this._width += attr.width
      }
    )
  }

  updateWeightAndWidth (newWidth: number) {
    this._width = newWidth
  }

  prependGroup(group: AttributeGroup | null) {
    this.groupAttrs[0].hidden = true
    // @ts-ignore
    this._groupAttrs = _.concat(group.groupAttrs, this.groupAttrs)
    // @ts-ignore
    this.takeOwnerShip(group.groupAttrs)
  }

  appendGround (group: AttributeGroup | null) {
    // @ts-ignore
    group.groupAttrs[0].hidden = true
    // @ts-ignore
    this._groupAttrs = _.concat(this.groupAttrs, group.groupAttrs)
    // @ts-ignore
    this.takeOwnerShip(group.groupAttrs)
  }

  removeAttribute (attr : Attribute) {
    const pos = this.groupAttrs.indexOf(attr)
    console.log(pos)

    if (pos !== 0) {
      this._groupAttrs[pos - 1].group = new AttributeGroup(this.groupAttrs.slice(0, pos))
    }
    if (pos !== this.groupAttrs.length - 1) {
      this.groupAttrs[pos].hidden = false
      this.groupAttrs[pos + 1].hidden = false
      this._groupAttrs[pos + 1].group = new AttributeGroup(this.groupAttrs.slice(pos + 1))
    }

    console.log(this.groupAttrs)

    attr.hidden = false
    attr.group = null
  }

  groupedAggregate (c: Candidate) {
    let percentage = 0
    const flag = _.every(this.groupAttrs, a => (a.key in c.attr))
    if (!flag) {
      return null
    }
    _.each(this.groupAttrs, a => {
      if (a.key in c.attr) {
        percentage += a.normalizer(c)
      }
    })
    return percentage
  }

  get width () {
    return this._width
  }

  get groupAttrs () {
    return this._groupAttrs
  }

  get children () {
    return this._groupAttrs.slice(1)
  }
}

export class AttributeFilterOption {
  public styles = {
    borderLeft: 0,
    borderRight: 0,
    left: 0,
    width: 0,
    height: 0,
    shadow: 1
  }

  constructor (public attr: Attribute,
               public range: number[],
               public candidates: Candidate[]) {
    const test = (c : Candidate) => c.attr[this.attr.key] > this.range[0] && c.attr[this.attr.key] < this.range[1]
    this.candidates = _.filter(this.candidates, test)
  }

  public setStyle (width: number, step: number, i: number, max: number) {
    this.styles.borderLeft = step * i
    this.styles.borderRight = step * (i + 1)
    this.styles.left = step * i + (step - width) / 2 + 0.008
    this.styles.width = width - 0.016
    this.styles.height = this.candidates.length / max
  }

  public updateShadow (filter: Function) {
    this.styles.shadow = _.filter(this.candidates, filter).length / this.candidates.length
  }
}

export class AttributeFilter {
  private _range = [0, 1]
  public maxValue = -Infinity
  public minValue = Infinity
  public maxDistributionValue = 0
  public options : AttributeFilterOption[] = []

  constructor(public attr : Attribute) {
    this.attr = attr
  }

  public initialize (candidates: Candidate[]) {
    this._computeMinMaxValue(candidates)
    this.options = _.map(this.discretizedValueRange, r => new AttributeFilterOption(this.attr, r, candidates))
    // @ts-ignore
    this.maxDistributionValue = _.maxBy(this.options, o => o.candidates.length).candidates.length
    const step = 1 / this.options.length
    const width = Math.min(step, 0.15)
    _.each(this.options, (opt, i) => {
      opt.setStyle(width, step, i, this.maxDistributionValue)
    })
  }

  private _computeMinMaxValue (candidates: Candidate[]) {
    if (candidates.length === 0) {
      this.minValue = 0
      this.maxValue = 0
      return
    }
    // @ts-ignore
    this.maxValue = _.maxBy(candidates, c => c.attr[this.attr.key]).attr[this.attr.key]
    // @ts-ignore
    this.minValue = _.minBy(candidates, c => c.attr[this.attr.key]).attr[this.attr.key]
    if (this.continues) {
      this.maxValue += 1e-5
    } else {
      this.minValue -= 0.5
      this.maxValue += 0.5
    }
  }

  public valueAt (p: number) {
    return p * (this.maxValue - this.minValue) + this.minValue
  }

  updateShadow (filter: Function) {
    _.each(this.options, opt => opt.updateShadow(filter))
  }

  get continues () {
    return this.attr.type === AttributeType.CONTINUOUS
  }

  get discretizedValueRange () {
    if (this.continues) {
      return linspace(this.minValue, this.maxValue + 1e-5, 12)
    } else {
      const range = _.range(Math.ceil(this.minValue), this.maxValue)
      return _.map(range, r => [r, r + 1e-5])
    }
  }

  get range () {
    return this._range
  }

  set range(r: number[]) {
    if (r[0] >= r[1]) {
      return
    }
    r[0] = Math.max(r[0], 0)
    r[1] = Math.min(r[1], 1)
    this._range = r
  }

  get pristine () {
    return this.range[0] === 0 && this.range[1] === 1
  }

  get valueRange () {
    return [this.valueAt(this.range[0]), this.valueAt(this.range[1])]
  }

  set valueRange (vr) {
    this.range = [
      (vr[0] - this.minValue) / (this.maxValue - this.minValue),
      (vr[1] - this.minValue) / (this.maxValue - this.minValue)
    ]
  }

  get testFunc () {
    const range = this.valueRange
    return (c: Candidate) => c.attr[this.attr.key] >= range[0] && c.attr[this.attr.key] < range[1]
  }

  get descriptor () {
    const range = this.valueRange
    return {
      name: this.continues
        ? `${this.attr.name}: ${csn(range[0])} ~ ${csn(range[1])}`
        : `${this.attr.name}: ${
          _.range(Math.ceil(range[0]), range[1]).join(', ')
        }`,
      fn: (c: Candidate) => c.attr[this.attr.key] >= range[0] && c.attr[this.attr.key] < range[1]
    }
  }
}
