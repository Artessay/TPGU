<template>
  <div id="route"></div>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator'

import Exploration from '@/store/modules/Exploration'

@Component
export default class Route extends Vue {
  public get selectedRoutes () {
    return Exploration.selectedRoutes
  }
}
</script>

<style lang="scss">

</style>
