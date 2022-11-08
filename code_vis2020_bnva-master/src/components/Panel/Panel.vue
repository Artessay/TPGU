<template>
  <div class="side-panel" :style="{
    transform: `translate(${selectedRoute ? 0 : -50}px, 0px)`
  }">
    <div class="icon-wrap">
      <font-awesome-icon
        @click="onStartOptimize()"
        :icon="optimizeBtnStatus"
        class="panel-icon" />
    </div>
    <div :class="['icon-wrap', { disable }]">
      <font-awesome-icon
        icon="search"
        @click="onSearch"
        class="panel-icon" />
    </div>
    <div :class="['icon-wrap', { disable }]">
      <font-awesome-icon
        icon="undo"
        @click="onUndo"
        class="panel-icon" />
    </div>
    <div :class="['icon-wrap', { disable }]">
      <font-awesome-icon
        icon="redo"
        @click="onRedo"
        class="panel-icon" />
    </div>
    <div :class="['icon-wrap', { disable }]">
      <font-awesome-icon
        icon="times"
        @click="onStop"
        class="panel-icon times" />
    </div>
    <div :class="['icon-wrap', { disable }]">
      <font-awesome-icon
        icon="route"
        @click="displayRoute"
        class="panel-icon" />
    </div>
    <div class="icon-wrap">
      <font-awesome-icon
        icon="sliders-h"
        class="panel-icon" />
    </div>
  </div>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator'
import CandidatesList, { Candidate } from '@/store/modules/CandidatesList'
import Dataset from '@/store/modules/Dataset'
import Manipulation from '@/store/modules/Manipulation';
import Exploration from '@/store/modules/Exploration'

@Component
export default class Panel extends Vue {
  public optimizeBtnStatus = 'magic' // 'pause', 'play'

  public onUndo () {
    Manipulation.getLastSnapshot()
  }

  public onRedo () {
    Manipulation.getNextSnapshot()
  }

  public onStop () {
    Manipulation.stopPlanning()
  }

  public displayRoute () {
    Exploration.toggleDisplayMatrixRoute()
  }

  public async onSearch () {
    console.log('onSearch')
    await CandidatesList.search(true)
    CandidatesList.clearCandidates()
    Manipulation.investigateConflicts()
  }

  public onStartOptimize () {
    if (this.optimizeBtnStatus === 'magic') {
      // eslint-disable-next-line no-undef
      CandidatesList.optimizeRoute(CandidatesList.targetRoute)
      CandidatesList.changeAttributesForManipulation()
      Manipulation.createStationGraph(CandidatesList.targetRoute)
      this.optimizeBtnStatus = 'pause'
    } else if (this.optimizeBtnStatus === 'pause') {
      Manipulation.pausePlanning()
      this.optimizeBtnStatus = 'play'
    } else if (this.optimizeBtnStatus === 'play') {
      Manipulation.continuePlanning()
      this.optimizeBtnStatus = 'pause'
    }
  }

  get selectedRoute () { // 从lineup中点击选择的路线的Route对象
    const len = Exploration.selectedRoutes.length
    return len > 0 ? Exploration.selectedRoutes[len - 1] : null
  }

  get disable () {
    if (this.optimizeBtnStatus === 'play') {
      return false
    } else {
      return true
    }
  }
}
</script>

<style lang="scss">
.side-panel {
  position: absolute;
  top: 140px;
  left: 0px;
  width: 40px;
  height: 250px;
  background-color: #fff;
  border-top-right-radius: 5px;
  border-bottom-right-radius: 5px;
  border: 1px solid #ccc;
  cursor: pointer;
  transition: transform 300ms;

  .icon-wrap {
    position: relative;
    left: 5px;
    width: 30px;
    height: 30px;
    margin-top: 5px;
    background-color: #93a6b9;
    border-radius: 2px;
    user-select: none;

    .panel-icon {
      color: #fff;
      line-height: 30px;
      font-size: 24px;
      padding: 3px;
    }

    .times {
      padding-left: 6px;
    }
  }

  .disable {
    background-color: #e4e9ed !important;
    cursor: not-allowed !important;
  }
}
</style>
