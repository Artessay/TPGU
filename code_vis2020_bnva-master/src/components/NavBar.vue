<template>
  <div id="navbar">
    <div class="left">
      <font-awesome-icon class="icon" icon="bus"></font-awesome-icon>
      <span class="title">Bus Network Visual Analytics System</span>
    </div>
    <div class="right">
      <span class="dataset">
        <span class="hint">DATASET:</span>
        {{ selectedDatasetName || 'None' }}
      </span>
    </div>
  </div>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator'

import Dataset from '@/store/modules/Dataset'

@Component
export default class NavBar extends Vue {
  public get selectedDatasetName () {
    return Dataset.selectedDataSource?.name
  }
}
</script>

<style lang="scss">
$navbarHeight: 40px;
$navbarColor: darken(#c1ccd7, 15%);//darken(#e6dbc7, 15%);
#navbar {
  flex: 0 0 $navbarHeight;
  background-color: $navbarColor;
  // border-bottom: 1px solid darken($navbarColor, 10%);
  display: flex;
  flex-direction: row;
  line-height: $navbarHeight;

  .left {
    flex: 1 1;
    font-size: 18px;
    color: white;

    .icon {
      margin-left: 15px;
    }
    .title {
      margin-left: 10px;
      font-weight: 700;
    }
  }

  .right {
    flex: 0 0 auto;
    background-color: darken($navbarColor, 5%);
    border-left: 1px solid darken($navbarColor, 7%);

    .dataset {
      margin: 0px 20px;
      font-size: 14px;
      font-weight: bold;
      color: white;

      .hint {
        color: darken($navbarColor, 20%);
        margin-right: 5px;
        font-weight: normal;
      }
    }
  }
}
</style>
