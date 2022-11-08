<template>
  <div class="conflicts-view">
    <ConflictVue v-for="(conflict, i) in conflicts" :key="i"
      :conflict="conflict"
      :conflictIndex="i" />
  </div>
</template>

<script lang="ts">
import { Component, Vue, Watch, Ref, Prop } from 'vue-property-decorator'
import ConflictVue from '@/components/ranking/lineup/ConflictVue.vue'
import CandidatesList, { Candidate } from '@/store/modules/CandidatesList'

@Component({
  components: {
    ConflictVue
  }
})
export default class Ranking extends Vue {
  // get conflict1 () {
  //   return [CandidatesList.filterCandidates.slice(0, 3), CandidatesList.filterCandidates.slice(3, 12), CandidatesList.filterCandidates.slice(12, 13), CandidatesList.filterCandidates.slice(12, 14), CandidatesList.filterCandidates.slice(12, 15), CandidatesList.filterCandidates.slice(12, 16)]
  // }

  // get conflict2 () {
  //   return [CandidatesList.filterCandidates.slice(13, 19), CandidatesList.filterCandidates.slice(19, 22)]
  // }

  get conflicts () {
    return CandidatesList.conflictsInLineup
  }

  @Watch('conflicts')
  public onConflictsChanged () {
    console.log(this.conflicts)
    console.log('onConflictsChanged')
    CandidatesList.onOpenConflict(-1)
    if (this.conflicts.length > 0) {
      CandidatesList.onOpenConflict(0)
    }
  }

  // setOpenedIndex (v: number) {
  //   console.log(v)
  //   if (v === this.openedIndex) {
  //     this.openedIndex = -1
  //   } else {
  //     this.openedIndex = v
  //   }
  // }
}
</script>

<style lang="scss">
@import '../../../style/Constants.scss';

.conflicts-view {
  overflow-y: scroll;
  width: 100%;
  position: relative;
  // height: calc(100% - #{$RANKING_COLUMN_HEADER_HEIGHT});
  height: $RANKING_CONTENT_HEIGHT;
  display: flex;
  flex-direction: column;
}
</style>
