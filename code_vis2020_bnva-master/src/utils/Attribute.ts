import _ from 'lodash'
import CandidatesList, { Candidate } from '@/store/modules/CandidatesList'
import { AttributeFilter, AttributeGroup } from "@/utils/Types"
import { AttrDistribution } from "@/utils/AttrDistribution"
const INITIAL_COLUMN_WIDTH = 180

export enum AttributeType {
  CONTINUOUS,
  ORDINAL
}

export enum UpdaterType {
  DEFAULT,
  ABSOLUTE
}

export enum GetterType {
  RAW,
  DEFAULT
}

export enum NormalizerType {
  PADDED,
  DEFAULT,
  PADDED_REVERSED,
  REVERSED
}

export class Attribute {
  public type: AttributeType = AttributeType.CONTINUOUS
  public width: number = INITIAL_COLUMN_WIDTH
  public hidden = false
  public maximum = -Infinity
  public minimum = Infinity
  public sorted: null | string = null // up or dow
  public group : AttributeGroup | null = null
  // cluster: null,
  public updater: Function = this.defaultUpdater
  public normalizer: Function = this.paddedNormalizer
  public getter: Function = this.rawGetter
  public attrDistribution: AttrDistribution | null = null
  // public filter: AttributeFilter | null = null

  constructor (
    public key: string,
    public name: string,
    public unit: string,
    public unitScale: number,
    public fixNumber?: number,
    public normalizerType?: NormalizerType,
    getter?: GetterType,
    updater?: UpdaterType
  ) {
    if (getter) {
      this.changeGetter(getter)
    }
    if (updater) {
      this.changeUpdater(updater)
    }
    if (normalizerType) {
      this.changeNormalizer(normalizerType)
    }
  }

  paddedReversedNormalizer (c: Candidate) {
    if (!c.attr[this.key]) {
      return 0.02
    }
    return this.maximum === this.minimum
      ? 1
      : 0.02 + 0.98 * (this.maximum - c.attr[this.key]) / (this.maximum - this.minimum)
  }

  reversedNormalizer (c: Candidate) {
    if (!c.attr[this.key]) {
      return 0
    }
    return this.maximum === this.minimum
      ? 1
      : (this.maximum - c.attr[this.key]) / (this.maximum - this.minimum)
  }

  paddedNormalizer (c: Candidate) {
    if (!c.attr[this.key]) {
      return 0.02 // 不存在的话，返回0.1.。。 返回0.1干啥。。。
    }
    return this.maximum === this.minimum
      ? 1
      : 0.02 + 0.98 * (c.attr[this.key] - this.minimum) / (this.maximum - this.minimum)
  }

  defaultNormalizer (c: Candidate) {
    if (!c.attr[this.key]) {
      return 0
    }
    return this.maximum === this.minimum
      ? 1
      : (c.attr[this.key] - this.minimum) / (this.maximum - this.minimum)
  }

  getBoxplotValuesOfCandidates (cs: Candidate[]) {
    if (this.group) {
      return cs.map((c) => {
        const groupAttrs = (this.group as AttributeGroup).groupAttrs
        return _.reduce(groupAttrs, (sum, a) => sum + (a.width * a.normalizer(c)), 0)
      })
    } else {
      return cs.map((c) => this.width * this.normalizer(c))
    }
  }

  getValuesOfCandidates (cs: Candidate[]) {
    if (this.group) {
      return cs.map((c) => {
        const groupAttrs = (this.group as AttributeGroup).groupAttrs
        return _.reduce(groupAttrs, (sum, a) => sum + c.attr[a.key], 0)
      })
    } else {
      return cs.map((c) => c.attr[this.key])
    }
  }

  getPaddedRatioWithValue (v: number) {
    return this.maximum === this.minimum
      ? 1
      : 0.02 + 0.98 * (v - this.minimum) / (this.maximum - this.minimum)
  }

  public changeGetter (getter: GetterType) {
    switch (getter) {
      case GetterType.DEFAULT:
        this.getter = this.rawGetter
        break
      case GetterType.RAW:
        this.getter = this.rawGetter
        break
    }
  }

  public changeNormalizer(normalizer: NormalizerType) {
    console.log(normalizer)
    switch (normalizer) {
      case NormalizerType.DEFAULT:
        this.normalizer = this.defaultNormalizer
        break
      case NormalizerType.PADDED_REVERSED:
        this.normalizer = this.paddedReversedNormalizer
        break
      case NormalizerType.PADDED:
        this.normalizer = this.paddedNormalizer
        break
      case NormalizerType.REVERSED:
        this.normalizer = this.reversedNormalizer
        break
    }
  }

  rawGetter (c: Candidate) {
    return c.attr[this.key]
  }

  formatGetter (c: Candidate) {
    if (this.fixNumber) {
      return (c.attr[this.key] / this.unitScale).toFixed(this.fixNumber) + this.unit
    }
    return Math.round(c.attr[this.key] / this.unitScale) + this.unit
  }

  setFilterLeft(leftRange: number) {
    if (this.attrDistribution) {
      this.attrDistribution.setLeftRange(leftRange)
    }
  }

  setFilterRight(rightRange: number) {
    if (this.attrDistribution) {
      this.attrDistribution.setRightRange(rightRange)
    }
  }

  public changeUpdater (updater: UpdaterType) {
    switch (updater) {
      case UpdaterType.ABSOLUTE:
        this.updater = this.absoluteUpdater
        break
      case UpdaterType.DEFAULT:
        this.updater = this.defaultUpdater
        break
      default:
        break
    }
  }

  updateWeightAndWidth (newWidth: number) {
    this.width = newWidth
  }

  forceUpdaterBasedOptimizingRoute (c: Candidate) {
    if (this.key in c.attr) {
      const v = c.attr[this.key]
      if (v > this.maximum) {
        this.maximum = v
      }
      if (v < this.minimum) {
        this.minimum = v
      }
    }
  }

  defaultUpdater (c: Candidate) {
    if (c.attr[this.key] > this.maximum) {
      this.maximum = c.attr[this.key]
    }
    if (c.attr[this.key] < this.minimum) {
      this.minimum = c.attr[this.key]
    }
  }

  absoluteUpdater (c: Candidate) {
    if (c.attr[this.key] > this.maximum) {
      this.maximum = c.attr[this.key]
    }
    this.minimum = 0
  }

  initializeMinMax () {
    this.maximum = -Infinity
    this.minimum = Infinity
  }
}
