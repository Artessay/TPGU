<template>
<div class="candidate-list">
  <!-- <entry class="optimized-candidate" :isFixed="true" v-if="routeOptimized" :c="routeOptimized" /> -->
  <transition-group name="candidate-list" tag="div">
    <entry v-for="c in candidates" :isFixed="false" :key="c.routeID" :c="c" />
  </transition-group>

</div>
</template>

<script lang="ts">
import { Component, Vue, Watch, Ref } from 'vue-property-decorator'
import CandidatesList, {Candidate} from '@/store/modules/CandidatesList';
import { px } from '@/utils/Formatter'
import Entry from '@/components/ranking/lineup/Entry.vue'

@Component({
  components: {
    Entry
  }
})
export default class Entries extends Vue {
  public px = px

  get setHoverRoute () {
    return CandidatesList.setHoverRoute
  }

  get showFilter () {
    return CandidatesList.showFilter
  }

  get candidates () {
    // const candidatesToBeShown = this.showFilter ? CandidatesList.filterCandidates : CandidatesList.candidateRoutes
    return CandidatesList.filterCandidates.slice(0, 50)
  }

  // get routeOptimized () {
  //   return CandidatesList.routeOptimized
  // }
}
</script>

<style lang="scss">
@import "../../../style/Constants";

.candidate-list {
  overflow-y: scroll;
  width: 100%;
  position: relative;
  // height: calc(100% - #{$RANKING_COLUMN_HEADER_HEIGHT});
  // height: $RANKING_CONTENT_HEIGHT;
  flex: 1 1;

  // .entry {
  //   flex: 0 0 $RANKING_LINEUP_ROW_HEIGHT;
  //   display: flex;
  //   flex-direction: row;
  //   border-bottom: 1px solid $GRAY2;
  //   transition: transform 1s, background-color 400ms;

  //   &.highlight {
  //     box-shadow: inset 0 0 3px $RANKING_LINEUP_BAR_COLOR;
  //   }

  //   .name-wrap {
  //     flex: 0 0 $RANKING_NAME_CELL_WIDTH;
  //     width: $RANKING_NAME_CELL_WIDTH;

  //     .name-icon {
  //       position: relative;
  //       float: left;
  //       height: 100%;
  //       line-height: $RANKING_LINEUP_ROW_HEIGHT;
  //       padding: 0 5px;
  //       font-size: 12px;
  //       color: #666;
  //     }

  //     .name {
  //       position: relative;
  //       float: left;
  //       line-height: $RANKING_LINEUP_ROW_HEIGHT;
  //       padding: 0 5px;
  //       font-size: 14px;
  //       color: #666;
  //       box-sizing: border-box;
  //       white-space: nowrap;
  //       text-overflow: ellipsis;
  //       overflow: hidden;
  //     }
  //   }

  //   .cell-wrapper {
  //     display: flex;
  //   }

  //   .cell-move {
  //     transition: transform .3s;
  //   }

  //   .cell-leave-active {
  //     display: none;
  //   }
  //   .cell-enter-active, .cell-leave-active {
  //     // transition: flex-basis 200ms, width 200ms;
  //     overflow: hidden;
  //   }
  //   .cell-enter, .cell-leave-to {
  //     flex-basis: 0px;
  //     width: 0px;
  //     opacity: 1;
  //   }

  //   .cell {
  //     $cell_x_padding: 5px;
  //     $cell_y_padding: 5px;

  //     padding: $cell_x_padding $cell_y_padding;
  //     border-right: 1px solid transparent;
  //     border-left: 1px solid transparent;
  //     position: relative;
  //     // transition: flex-basis 200ms, width 200ms;

  //     .bar {
  //       height: 100%;
  //       background-color: $RANKING_LINEUP_BAR_COLOR;
  //       float: left;
  //       position: relative;
  //       overflow: hidden;
  //       // transition: all 300ms;
  //       // transition: width 200ms, flex-basis 200ms;
  //       border-radius: 3px;

  //       &.group {
  //         flex: 0 0;
  //         margin-left: 5px;
  //       }

  //       .text {
  //         position: absolute;
  //         height: 100%;
  //         width: calc(100% - 2px);
  //         padding-left: 2px;
  //         line-height: 20px;
  //         font-size: 12px;
  //         user-select: none;
  //         color: #4d4d4d;
  //       }
  //     }

  //     .group-bar-cont {
  //       height: 100%;
  //       white-space: nowrap;
  //       display: flex;
  //       flex-direction: row;

  //       @keyframes squeeze {
  //         to {
  //           width: 0;
  //         }
  //       }

  //       .animated-spacing {
  //         float: left;
  //         height: 100%;
  //         animation-name: squeeze;
  //         animation-duration: 200ms;
  //         animation-fill-mode: forwards;
  //         transition-duration: 200ms;
  //       }
  //     }
  //   }

  //   .grouped {
  //     background-color: #fafbfe;
  //     border-right: 1px solid #e6e6e6;
  //     border-left: 1px solid #e6e6e6;
  //   }
  // }

}
</style>
