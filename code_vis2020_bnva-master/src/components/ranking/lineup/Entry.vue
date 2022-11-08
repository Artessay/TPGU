<template>
  <div :class="[{
      highlight: highlightRoute && (c.routeID === highlightRoute.routeID)
    }, 'entry']"
    @click=" () => { onEntryClick(c) }"
    @mouseenter="() => onEntryEnter(c)"
    @mouseleave="() => onLeaveEnter(c)">
    <div class="name-wrap">
      <font-awesome-icon
        icon="project-diagram"
        :class="[{ 'fixed': isFixed }, 'name-icon']"></font-awesome-icon>
      <div :class="[{ 'fixed': isFixed }, 'name']">{{c.name}}</div>
    </div>
    <transition-group :name="headerResizing ? 'cell-non-animated' : 'cell'" tag="div" class="cell-wrapper">
      <div v-for="attr in attributes"
        :key="attr.key"
        :class="[{ grouped: attr.group }, 'cell']"
        :style="{
            flex: `0 0 ${px(attr.group ? attr.group.width : attr.width)}`,
            'padding-right': `${px(5 + (attr.group ? (attr.group.groupAttrs.length - 1) * 12 : 0))}`
          }">
        <div class="bar"
          :style="{
            width: `${px(attr.width * attr.normalizer(c))}`,
            overflow: attr.group ? 'hidden': 'visible'
          }">
          <div class="text" :id="attr.key">{{attr.formatGetter(c)}}</div>
        </div>
        <div class="group-bar-cont" v-if="attr.group">
          <div class="bar group"
              v-for="gattr in attr.group.children"
              :key="gattr.key"
              :style="{
              'flex-basis': px(
              gattr.normalizer(c) * gattr.width
            )
          }">
            <div class="text">
              {{gattr.formatGetter(c)}}
            </div>
          </div>
        </div>
        <div class="optimize-hint" v-if="routeOptimized && (!attr.group) && attr.rawGetter(routeOptimized)"
          :style="{ left: `${px(attr.width * attr.normalizer(routeOptimized))}` }">
        </div>
        <div class="optimize-hint" v-if="routeOptimized && attr.group && attr.group.groupedAggregate(routeOptimized)"
          :style="{ left: `${px(attr.width * attr.group.groupedAggregate(routeOptimized))}` }">
        </div>
      </div>
    </transition-group>
  </div>
</template>

<script lang="ts">
import { Component, Vue, Prop } from 'vue-property-decorator'
import CandidatesList, { Candidate } from '@/store/modules/CandidatesList'
import Exploration from '@/store/modules/Exploration'
import _ from 'lodash'
import { px } from '@/utils/Formatter'
import { AttributeGroup } from '../../../utils/Types'

@Component
export default class Entry extends Vue {
  @Prop() c!: Candidate
  @Prop() isFixed!: boolean

  public px = px

  get attributes () {
    return _.filter(CandidatesList.attributes, ['hidden', false])
  }

  get routeOptimized () {
    return CandidatesList.routeOptimized
  }

  get highlightRoute () {
    return CandidatesList.highlightCandidate
  }

  get headerResizing () {
    return CandidatesList.headerResizing
  }

  // validAttrGroupForHint (gattr: AttributeGroup) {
  //   const routeOptimized = CandidatesList.routeOptimized
  //   if (routeOptimized) {
  //     return _.every(gattr.groupAttrs, a => (a.key in routeOptimized.attr))
  //   } else {
  //     return false
  //   }
  // }

  onEntryEnter (c: Candidate) {
    Exploration.setHighlightRoutesGeoJSON({routes: [c.routeID], selected: false})
  }

  onLeaveEnter (c: Candidate) {
    Exploration.setHighlightRoutesGeoJSON({routes: [], selected: false})
  }

  onEntryClick (c: Candidate) {
    if (CandidatesList.routeOptimized) {
      console.log(CandidatesList.routeOptimized)
      return
    }
    if (CandidatesList.highlightCandidate && CandidatesList.highlightCandidate.routeID === c.routeID) {
      Exploration.toggleMatrixHighLightRoute()
      CandidatesList.toggleHighLightRoute(null)
      Exploration.toggleRoute(Exploration.indexedRoutes[c.routeID])
    } else {
      CandidatesList.toggleHighLightRoute(c)
      Exploration.toggleMatrixHighLightRoute(c.routeID)
      Exploration.toggleRoute(Exploration.indexedRoutes[c.routeID])
    }
  }
}
</script>

<style lang="scss">
@import "../../../style/Constants.scss";
.entry {
  flex: 0 0 $RANKING_LINEUP_ROW_HEIGHT;
  display: flex;
  flex-direction: row;
  border-bottom: 1px solid $GRAY2;
  transition: transform 1s, background-color 400ms;

  &.highlight {
    box-shadow: inset 0 0 3px #888;
  }

  .name-wrap {
    flex: 0 0 $RANKING_NAME_CELL_WIDTH;
    width: $RANKING_NAME_CELL_WIDTH;
    color: #666;

    .name-icon {
      position: relative;
      float: left;
      height: 100%;
      line-height: $RANKING_LINEUP_ROW_HEIGHT;
      padding: 0 5px;
      font-size: 12px;
      color: inherit;
    }

    .name {
      position: relative;
      width: 170px;
      float: left;
      line-height: $RANKING_LINEUP_ROW_HEIGHT;
      padding: 0 5px;
      font-size: 14px;
      color: inherit;
      box-sizing: border-box;
      white-space: nowrap;
      text-overflow: ellipsis;
      overflow: hidden;
    }

    .fixed {
      color: red !important;
    }
  }

  .cell-wrapper {
    display: flex;
  }

  .cell-move {
    transition: transform .3s;
  }

  .cell-leave-active {
    display: none;
  }
  .cell-enter-active, .cell-leave-active {
    // transition: flex-basis 200ms, width 200ms;
    overflow: hidden;
  }
  .cell-enter, .cell-leave-to {
    flex-basis: 0px;
    width: 0px;
    opacity: 1;
  }

  .cell {
    $cell_x_padding: 5px;
    $cell_y_padding: 5px;

    padding: $cell_x_padding $cell_y_padding;
    border-right: 1px solid transparent;
    border-left: 1px solid transparent;
    position: relative;
    // transition: flex-basis 200ms, width 200ms;

    .optimize-hint {
      position: absolute;
      width: 0px;
      border-left: 1px dashed #7abde6;
      top: 0px;
      height: 100%;
    }

    .bar {
      height: 100%;
      background-color: $RANKING_LINEUP_BAR_COLOR;
      float: left;
      position: relative;
      overflow: visible;
      // transition: all 300ms;
      // transition: width 200ms, flex-basis 200ms;
      border-radius: 3px;

      &.group {
        flex: 0 0;
        margin-left: 5px;
        overflow: hidden !important;
      }

      .text {
        position: absolute;
        height: 100%;
        width: calc(100% - 2px);
        padding-left: 2px;
        line-height: 20px;
        font-size: 12px;
        user-select: none;
        color: #666;
        // overflow: hidden;
        // text-overflow: ellipsis;
      }
    }

    .group-bar-cont {
      height: 100%;
      white-space: nowrap;
      display: flex;
      flex-direction: row;

      @keyframes squeeze {
        to {
          width: 0;
        }
      }

      .animated-spacing {
        float: left;
        height: 100%;
        animation-name: squeeze;
        animation-duration: 200ms;
        animation-fill-mode: forwards;
        transition-duration: 200ms;
      }
    }
  }

  .grouped {
    background-color: #fafbfe;
    border-right: 1px solid #e6e6e6;
    border-left: 1px solid #e6e6e6;
  }
}
</style>
