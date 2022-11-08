<template>
<div id="ranking">
  <Lineup id="lineup"></Lineup>
</div>
</template>

<script lang="ts">
import { Component, Vue, Watch, Ref } from 'vue-property-decorator'
import Lineup from '@/components/ranking/lineup/Lineup.vue'
import ProjectionView from '@/components/ranking/Projection.vue'
import CandidatesList from '@/store/modules/CandidatesList'

@Component({
  components: {
    Projection: ProjectionView,
    Lineup
  }
})
export default class Ranking extends Vue {
  public toggleFilter : Function = CandidatesList.toggleRankingFilter

  public get showFilter () {
    return CandidatesList.showFilter
  }

  public mounted (): void {
    console.log('')
  }
}
</script>

<style lang="scss">
@import "../../style/Constants.scss";
$backgroundColor: lighten(#e6dbc7, 12%);
$columnSpacing: 10px;
#ranking {
  position: relative;
  height: 100%;
  flex-direction: row;
  align-self: flex-end;
  flex: 0 0 100%;
  display: flex;
  z-index: 1;
  // margin: 10px;
  background-color: $backgroundColor;
  // border: 1px solid #ccc;
  // border-radius: 5px;
  box-shadow: 0 0 10px #ccc;
  // padding: 0 10px 0px 0;

  // .filter-cont {
  //   height: $RANKING_FILTER_BAR_HEIGHT;
  //   background-color: $GRAY1;
  //   border-top: 1px solid $GRAY2;
  //   position: relative;

  //   .filters {
  //     position: relative;
  //     margin-left: $RANKING_NAME_CELL_WIDTH;
  //     border-left: 1px solid $GRAY2;
  //     padding: 0 10px;
  //     font-size: 14px;
  //     color: #777;

  //     span {
  //       margin-right: 5px;
  //       line-height: $RANKING_FILTER_BAR_HEIGHT - 2px;
  //       transform: translateY(-1px);
  //     }

  //     .filter-btn {
  //       position: relative;
  //       display: inline-block;
  //       background-color: white;
  //       height: 23px;
  //       line-height: 22px;
  //       padding: 0 25px 0 10px;
  //       border: 1px solid $GRAY2;

  //       .remove {
  //         position: absolute;
  //         top: 4px;
  //         right: 6px;
  //         width: 15px;
  //         height: 15px;
  //         line-height: 15px;
  //         text-align: center;
  //         cursor: pointer;

  //         transition: color 200ms;
  //         color: $GRAY4;

  //         &:hover {
  //           color: $RANKING_COLUMN_DELETE_BUTTON_ACTIVE_COLOR;
  //         }
  //       }
  //     }

  //     .add-btn {
  //       position: absolute;
  //       top: 7px;
  //       right: 7px;
  //       display: block;
  //       height: 24px;
  //       line-height: 24px;
  //       padding: 0 10px;
  //       background-color: $GRAY0;
  //       border: 1px solid $GRAY3;
  //       border-radius: 5px;
  //       cursor: pointer;
  //       transition: background-color 200ms, color 200ms;

  //       &:hover {
  //         background-color: $GRAY1;
  //       }

  //       &.active {
  //         background-color: $GRAY4;
  //         color: white;
  //       }
  //     }
  //   }

  //   .left-btn {
  //     display: block;
  //     position: absolute;
  //     top: #{($RANKING_FILTER_BAR_HEIGHT - 24) / 2 - 1};
  //     width: 22px;
  //     height: 22px;
  //     line-height: 22px;
  //     border: 1px solid $GRAY4;
  //     border-radius: 3px;
  //     text-align: center;
  //     font-size: 13px;
  //     background-color: $GRAY0;
  //     transition: background-color 200ms, color 200ms;
  //     cursor: pointer;

  //     &:hover {
  //       background-color: $GRAY1;
  //     }

  //     &.active {
  //       background-color: $GRAY4;
  //       color: white;
  //     }
  //   }

  //   .column-btn {
  //     // top: #{($RANKING_FILTER_BAR_HEIGHT - 24) / 2 + 1};
  //     left: 7px;

  //     %triangle {
  //       content: '';
  //       position: absolute;
  //       left: 17px;
  //       bottom: -10px;
  //       border-right: 8px solid transparent;
  //       border-left: 8px solid transparent;
  //       border-top: 10px solid #ddd;
  //       transform: translateX(-4px);
  //     }

  //     .menu{
  //       font-family: 'Lato', Arial, Helvetica, sans-serif;
  //       text-shadow: none;
  //       position: absolute;
  //       left: -10px;
  //       bottom: calc(100% + 14px);
  //       width: 120px;

  //       &::before {
  //         @extend %triangle;
  //       }

  //       &::after {
  //         @extend %triangle;
  //         border-top: 9px solid #fff;
  //         transform: translate(-4px, -2px)
  //       }

  //       ul {
  //         list-style-type: none;
  //         padding: 0;
  //         margin: 0;
  //         background-color: #fff;
  //         border: 1px solid #ddd;
  //         border-radius: 5px;

  //         li {
  //           color: $GRAY5;
  //           text-align: left;
  //           padding: 0 10px;
  //           height: 28px;
  //           line-height: 28px;
  //           border-bottom: 1px solid #eee;
  //           cursor: pointer;

  //           &:last-child {
  //             border-bottom: none;
  //           }

  //           &.active {
  //             background-color: #f7f7f7;
  //           }
  //           &:hover {
  //             background-color: #eee;
  //           }
  //         }
  //       }
  //     }

  //     &.disabled {
  //       .menu {
  //         display: none;
  //       }
  //       color: $GRAY3;
  //       cursor: not-allowed;
  //       &:hover {
  //         background-color: white;
  //       }
  //     }
  //   }

  //   .text-btn {
  //     left: 35px;
  //   }

  //   .candidate-count {
  //     display: block;
  //     position: absolute;
  //     top: 0;
  //     left: 65px;
  //     width: 75px;
  //     line-height: 35px;
  //     color: #666;
  //     text-align: right;
  //   }
  // }
}
#lineup {
  width: 100%;
}
</style>
