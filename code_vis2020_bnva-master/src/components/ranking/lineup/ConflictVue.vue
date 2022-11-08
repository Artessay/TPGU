<template>
  <div class="conflict"
    :style="{ flex: `0 0 ${opened ? (25 + conflict.gids.length * 30) : 25}px` }">
    <div class="title" @click="onOpenConflict"
      :style="{ backgroundColor: opened ? '#ebf2fc' : '#e9edf1' }">
      {{`Routing from Station ${conflict.origin} to Station ${conflict.dest}`}}
    </div>
    <div class="boxplot-entry-wrapper"
      :style="{ height: opened ? px(conflict.gids.length * 30) : 0 }">
      <transition-group name="boxplot-list" tag="div">
        <boxplot-entry v-for="gid in conflict.gids" :key="gid"
          :candidates="conflict.candidatesDict[gid]"
          :gid="gid"
          @onClickBoxplotEntry="handleConflict"
          @onHoverBoxplotEntry="hoverConflict"
          @onLeaveBoxplotEntry="leaveConflict" />
      </transition-group>
    </div>
  </div>
</template>

<script lang="ts">
import { Component, Vue, Prop } from 'vue-property-decorator'
import { prec, px } from '@/utils/Formatter'
import BoxplotEntry from './BoxplotEntry.vue'
import Entry from './Entry.vue'
import CandidatesList, { Candidate, ConflictInLineup } from '@/store/modules/CandidatesList'
import {Conflict} from "@/utils/RouteGroup";
import Evaluation from "@/store/modules/Evaluation";

@Component({
  components: {
    BoxplotEntry,
    Entry
  }
})
export default class ConflictVue extends Vue {
  @Prop() conflictIndex!: number
  @Prop() conflict!: ConflictInLineup

  public px = px
  public prec = prec

  get openedConflictIndex () {
    return CandidatesList.openedConflictIndex
  }

  get opened () {
    return this.openedConflictIndex === this.conflictIndex
  }

  public onOpenConflict () {
    CandidatesList.onOpenConflict(this.conflictIndex)
    // this.$emit('setOpenedIndex', this.conflictIndex)
  }

  public handleConflict (gid: number) {
    // the clicked group is stored in "this.conflict"
    // call function to handle this conflict?
    console.log('onClickBoxplotEntry', this.conflict, gid)
    Evaluation.selectGroupByGroupID(gid)
  }

  public hoverConflict (gid: number) {
    console.log('onHoverBoxplotEntry', gid)
    Evaluation.setHighLightRoutesByGroup(gid)
    Evaluation._setHighLightGroup(gid)
    Evaluation.updateConflictGeoJson()
  }

  public leaveConflict (gid: number) {
    console.log('onLeaveBoxplotEntry', gid)
    Evaluation.clearHighlightRoutes()
    Evaluation._clearHighLightGroup(gid)
    Evaluation.updateConflictGeoJson()
  }
}
</script>

<style lang="scss">
@import '../../../style/Constants.scss';

.conflict {
  position: relative;
  // flex: 0 0 $RANKING_LINEUP_ROW_HEIGHT;
  // flex-direction: row;
  border-bottom: 1px solid $GRAY2;
  transition: transform 1s, background-color 400ms, flex-basis 300ms;
  // background-color: #e9edf1;

  .title {
    position: relative;
    padding-left: 10px;
    color: #666;
    line-height: 25px;
    font-size: 15px;
    height: 25px;
    border-bottom: 1px solid #e6e6e6;
    width: 100%;
    transition: 'background-color' 200ms;
  }

  .boxplot-entry-wrapper {
    position: relative;
    width: 100%;
    overflow: hidden;
    transition: height 300ms;
  }
}

.boxplot-list-move {
  transition: transform 1s;
}
</style>
