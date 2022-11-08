<template>
  <div class="boxplot">
    <div class="box" v-if="l > 3" :style="boxStyle" :title="values.toString()" />
    <div class="whiskers" :style="whiskersStyle">
      <div class="whiskers-link"></div>
    </div>
    <div class="median" v-if="l > 4 || l === 3" :style="medianStyle" />
    <!-- {{`${q1},${q3}`}} -->
  </div>
</template>

<script lang="ts">
import { Component, Vue, Prop } from 'vue-property-decorator'
import { prec, px } from '@/utils/Formatter'
import { Candidate } from '@/store/modules/CandidatesList'
import { Attribute } from '@/utils/Attribute'

@Component
export default class Boxplot extends Vue {
  @Prop() attr!: Attribute
  @Prop() values!: number[]
  @Prop() boxplotValues!: number[]

  get l () {
    return this.boxplotValues.length
  }

  get sortedValues () {
    return this.boxplotValues.sort((a, b) => a - b)
  }

  get medianStyle () {
    const i = Math.floor(this.l / 2)
    const median = this.sortedValues[i]
    return {
      left: px(median),
      'background-color': this.l > 3 ? '#bad2e4' : '#ccdeeb'
    }
  }

  get q1 () {
    const i = Math.floor(this.l / 4)
    return this.sortedValues[i]
  }

  get q3 () { // q3 > q1
    let i = Math.ceil(this.l * 3 / 4)
    i = i >= this.l ? (i - 1) : i
    return this.sortedValues[i]
  }

  get boxStyle () {
    return {
      left: px(this.q1),
      width: px(this.q3 - this.q1)
    }
  }

  get whiskersStyle () {
    return {
      left: px(this.sortedValues[0]),
      width: px(this.sortedValues[this.l - 1] - this.sortedValues[0])
    }
  }

  // get l () {
  //   return this.values.length
  // }

  // get sortedValues () {
  //   return this.values.sort((a, b) => a - b)
  // }

  // get q1 () {
  //   const i = Math.floor(this.l / 4)
  //   return this.sortedValues[i]
  // }

  // get q3 () { // q3 > q1
  //   let i = Math.ceil(this.l * 3 / 4)
  //   i = i >= this.l ? (i - 1) : i
  //   return this.sortedValues[i]
  // }

  // get iqr () {
  //   return this.q1 - this.q3
  // }

  // get width () {
  //   return this.attr.group ? this.attr.group.width : this.attr.width
  // }

  // get boxStyle () {
  //   const leftRatio = this.attr.getPaddedRatioWithValue(this.q1)
  //   const left = this.width * leftRatio

  //   const rightRatio = this.attr.getPaddedRatioWithValue(this.q3)
  //   const right = this.width * rightRatio

  //   return {
  //     left: px(left),
  //     width: px(right - left)
  //   }
  // }

  // get whiskersStyle () {
  //   const leftRatio = this.attr.getPaddedRatioWithValue(this.sortedValues[0])
  //   const left = this.width * leftRatio

  //   const rightRatio = this.attr.getPaddedRatioWithValue(this.sortedValues[this.l - 1])
  //   const right = this.width * rightRatio

  //   return {
  //     left: px(left),
  //     width: px(right - left)
  //   }
  // }
}
</script>

<style lang="scss">
@import '../../../style/Constants.scss';
.boxplot {
  position: relative;
  height: 100%;
  width: 100%;

  .box {
    position: absolute;
    background-color: #ccdeeb;
    height: 100%;
    border-radius: 4px;
    min-width: 1px !important;
  }

  .median {
    position: absolute;
    height: 100%;
    background-color: #a7c6dd;
    width: 1px;
  }

  .whiskers {
    position: absolute;
    border-left: 1px solid #ccdeeb;
    border-right: 1px solid #ccdeeb;
    height: 100%;

    .whiskers-link {
      position: relative;
      height: 1px;
      width: 100%;
      top: calc(50% - 0.5px);
      background-color: #ccdeeb;
    }
  }
}
</style>
