import CandidatesList, { Candidate } from "@/store/modules/CandidatesList";
import { Attribute, AttributeType, NormalizerType } from "@/utils/Attribute";
import { linspace } from "@/utils/Helper";
import { csn } from "@/utils/Formatter";
import _ from 'lodash'
import * as d3 from 'd3'

export class AttrDistribution {
  private _range = [0, 1]
  public maxValue = -Infinity
  public minValue = Infinity
  public maxDistributionValue = 0
  public distribution : number[] = []
  public path = ''

  constructor(public attr : Attribute) {
    this.attr = attr
  }

  public initialize (candidates: Candidate[]) {
    this._computeMinMaxValue(candidates)
    this._getDistribution(candidates, 20)
    this.maxDistributionValue = Math.max(...this.distribution)
    this._getPath()
  }

  public setLeftRange (leftRange: number) {
    this._range = [leftRange, this._range[1]]
    // if (this._range[0] < 0) {
    //   this._range = [0, this._range[1]]
    // }
    // if (this._range[1] > 1) {
    //   this._range = [this._range[0], 1]
    // }
  }

  public setRightRange (rightRange: number) {
    this._range = [this._range[0], rightRange]
    // if (this._range[0] < 0) {
    //   this._range = [0, this._range[1]]
    // }
    // if (this._range[1] > 1) {
    //   this._range = [this._range[0], 1]
    // }
  }

  private _getPath () {
    const svgWidth = 178
    const svgHeight = 70

    const xScale = d3.scaleLinear()
      .domain([0, this.distribution.length - 1])
      .range([0, svgWidth])
    const yScale = d3.scaleLinear()
      .domain([0, this.maxDistributionValue])
      .range([svgHeight, 0])

    const hackDistribution = this.distribution.map((v, i) => [v, i])
    hackDistribution.unshift([0, 0])
    hackDistribution.push([0, this.distribution.length - 1])

    this.path = d3.line()
      .curve(d3.curveBasis)
      // @ts-ignore
      .x(d => xScale(d[1]))
      // @ts-ignore
      .y(d => yScale(d[0]))(hackDistribution) as string
  }

  private _getDistribution (candidates: Candidate[], nBin: number) {
    const normalizerType = this.attr.normalizerType
    let minValue = this.minValue
    let maxValue = this.maxValue
    if (normalizerType === NormalizerType.PADDED_REVERSED) {
      minValue = this.maxValue
      maxValue = this.minValue
    }

    const distribution = new Array(nBin).fill(0)
    const interval = (maxValue - minValue) / nBin
    _.each(candidates, c => {
      let bin = Math.floor((c.attr[this.attr.key] - minValue) / interval)
      bin = bin >= nBin ? (bin - 1) : bin

      distribution[bin]++
    })
    this.distribution = distribution
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

    // hack for routeOptimized
    if (CandidatesList.routeOptimized && (!CandidatesList.searching) && (this.attr.key in CandidatesList.routeOptimized.attr)) {
      const v = this.attr.rawGetter(CandidatesList.routeOptimized)
      if (v > this.maxValue) {
        this.maxValue = v
      }
      if (v < this.minValue) {
        this.minValue = v
      }
    }

    if (this.continues) {
      this.maxValue += 1e-5
    } else {
      this.minValue -= 0.5
      this.maxValue += 0.5
    }
  }

  public valueAt (p: number) {
    const normalizerType = this.attr.normalizerType
    let minValue = this.minValue
    let maxValue = this.maxValue
    if (normalizerType === NormalizerType.PADDED_REVERSED) {
      minValue = this.maxValue
      maxValue = this.minValue
    }
    return p * (maxValue - minValue) + minValue
  }

  // updateShadow (filter: Function) {
  //   _.each(this.options, opt => opt.updateShadow(filter))
  // }

  get continues () {
    return this.attr.type === AttributeType.CONTINUOUS
  }

  get validRange () {
    let r0 = this.range[0]
    let r1 = this.range[1]
    if (r0 > r1) {
      if (r1 < 0) {
        r1 = 0
      }
      if (r0 > 1) {
        r0 = 1
      }
      return [r1, r0]
    } else {
      if (r0 < 0) {
        r0 = 0
      }
      if (r1 > 1) {
        r1 = 1
      }
      return [r0, r1]
    }
  }

  // get discretizedValueRange () {
  //   if (this.continues) {
  //     return linspace(this.minValue, this.maxValue + 1e-5, 12)
  //   } else {
  //     const range = _.range(Math.ceil(this.minValue), this.maxValue)
  //     return _.map(range, r => [r, r + 1e-5])
  //   }
  // }

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
    return this.validRange[0] === 0 && this.validRange[1] === 1
  }

  get valueRange () {
    return [this.valueAt(this.validRange[0]), this.valueAt(this.validRange[1])]
  }

  set valueRange (vr) { // ? 没看懂 可能有bug
    this.range = [
      (vr[0] - this.minValue) / (this.maxValue - this.minValue),
      (vr[1] - this.minValue) / (this.maxValue - this.minValue)
    ]
  }

  get testFunc () {
    const valueRange = this.valueRange
    const normalizerType = this.attr.normalizerType
    let minValue = valueRange[0]
    let maxValue = valueRange[1]
    if (normalizerType === NormalizerType.PADDED_REVERSED) {
      minValue = valueRange[1]
      maxValue = valueRange[0]
    }
    return (c: Candidate) => c.attr[this.attr.key] >= minValue && c.attr[this.attr.key] < maxValue
  }

  get intersectFunc () {
    const normalizerType = this.attr.normalizerType
    let vr0 = this.valueRange[0]
    let vr1 = this.valueRange[1]
    if (normalizerType === NormalizerType.PADDED_REVERSED) {
      vr0 = this.valueRange[1]
      vr1 = this.valueRange[0]
    }

    return (cs: Candidate[]) => {
      const values = cs.map((c) => c.attr[this.attr.key])
      const maxValueInCs = Math.max(...values)
      const minValueInCs = Math.min(...values)
      console.log(minValueInCs, maxValueInCs, vr0, vr1)
      return vr0 < maxValueInCs && vr1 > minValueInCs
    }
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
