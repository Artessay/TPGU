<template>
  <div class="boxplot-entry" @click="onClick" @mouseenter="onEnter" @mouseleave="onLeave">
    <div class="name-wrap">
      <font-awesome-icon
        icon="cube"
        class="name-icon"></font-awesome-icon>
      <div class="name">{{candidates.length}} Routes</div>
    </div>

    <transition-group :name="headerResizing ? 'cell-non-animated' : 'cell'" tag="div" class="cell-wrapper">
      <div v-for="attr in attributes"
        :key="attr.key"
        :class="[{ grouped: attr.group }, 'cell']"
        :style="{
            flex: `0 0 ${px(attr.group ? attr.group.width : attr.width)}`,
            'padding-right': `${px(5 + (attr.group ? (attr.group.groupAttrs.length - 1) * 12 : 0))}`
          }">
        <boxplot v-if="candidates.length > 1" :attr="attr"
          :values="attr.getValuesOfCandidates(candidates)"
          :boxplotValues="attr.getBoxplotValuesOfCandidates(candidates)" />
        <div v-else-if="candidates.length === 1" class="bar-container">
          <div class="bar"
            :style="{
              width: `${px(attr.width * attr.normalizer(candidates[0]))}`,
              overflow: attr.group ? 'hidden': 'visible'
            }">
            <div class="text" :id="attr.key">{{attr.formatGetter(candidates[0])}}</div>
          </div>
          <div class="group-bar-cont" v-if="attr.group">
            <div class="bar group text"
                v-for="gattr in attr.group.children"
                :key="gattr.key"
                :style="{
                'flex-basis': px(
                gattr.normalizer(candidates[0]) * gattr.width
              )
            }">
              <div class="text">
                {{gattr.formatGetter(candidates[0])}}
              </div>
            </div>
          </div>
        </div>

      </div>
    </transition-group>

  </div>
</template>

<script lang="ts">
import { Component, Vue, Prop } from 'vue-property-decorator'
import { prec, px } from '@/utils/Formatter'
import CandidatesList, { Candidate } from '@/store/modules/CandidatesList'
import _ from 'lodash'
import Boxplot from './Boxplot.vue'

@Component({
  components: {
    Boxplot
  }
})
export default class BoxplotEntry extends Vue {
  @Prop() candidates!: Candidate[]
  @Prop() gid!: number

  public px = px

  get attributes () {
    return _.filter(CandidatesList.attributes, ['hidden', false])
  }

  get headerResizing () {
    return CandidatesList.headerResizing
  }

  public onClick () {
    this.$emit('onClickBoxplotEntry', this.gid)
  }

  public onEnter () {
    this.$emit('onHoverBoxplotEntry', this.gid)
  }

  public onLeave () {
    this.$emit('onLeaveBoxplotEntry', this.gid)
  }
}
</script>

<style lang="scss" scoped>
@import '../../../style/Constants.scss';
.boxplot-entry {
  position: relative;
  height: 30px;
  width: 100%;
  display: flex;

  .name-wrap {
    flex: 0 0 $RANKING_NAME_CELL_WIDTH;
    width: $RANKING_NAME_CELL_WIDTH;
    color: #666;
    border-right: 1px solid #eee;

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
      font-weight: 600;
    }
  }

  .cell-wrapper {
    display: flex;
  }

  .cell {
    $cell_x_padding: 5px;
    $cell_y_padding: 5px;

    padding: $cell_x_padding $cell_y_padding;
    border-right: 1px solid transparent;
    border-left: 1px solid transparent;
    position: relative;
    // transition: flex-basis 200ms, width 200ms;

    .bar-container {
      position: relative;
      height: 100%;
      width: 100%;

      .bar {
        height: 100%;
        background-color: $RANKING_LINEUP_BAR_COLOR;
        float: left;
        position: relative;
        overflow: hidden;
        // transition: all 300ms;
        // transition: width 200ms, flex-basis 200ms;
        border-radius: 3px;

        &.group {
          flex: 0 0;
          margin-left: 5px;
          overflow: hidden;
        }

        .text {
          position: absolute;
          height: 100%;
          width: calc(100% - 2px);
          padding-left: 2px;
          line-height: 20px;
          font-size: 12px;
          user-select: none;
          color: #4d4d4d;
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
  }

  .grouped {
    background-color: #fafbfe;
    border-right: 1px solid #e6e6e6;
    border-left: 1px solid #e6e6e6;
  }

}
</style>
