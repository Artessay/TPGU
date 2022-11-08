<template>
  <div class="lineup-wrapper">
    <ColumnHeaders></ColumnHeaders>
    <Entries v-if="!isSearching"></Entries>
    <Conflicts v-else />
  </div>
</template>

<script lang="ts">
import { Component, Vue, Watch, Ref } from 'vue-property-decorator'
import ColumnHeaders from '@/components/ranking/lineup/ColumnHeaders.vue'
import Entries from '@/components/ranking/lineup/Entries.vue'
import Conflicts from '@/components/ranking/lineup/Conflicts.vue'
import CandidatesList, { Candidate } from '@/store/modules/CandidatesList'

@Component({
  components: {
    ColumnHeaders,
    Entries,
    Conflicts
  }
})
export default class Lineup extends Vue {
  get isSearching () {
    return CandidatesList.searching
  }
}
</script>

<style lang="scss">
@import "../../../style/Constants.scss";
.lineup-wrapper {
  height: 100%;
  position: relative;
  width: 100%;
  background-color: #fff;
  display: flex;
  flex-direction: column;
}
</style>
