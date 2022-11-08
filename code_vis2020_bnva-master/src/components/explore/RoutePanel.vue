<template>
  <div id="route-panel" v-if="selectedRoute">
    <div class="headers">
      <div id="station-header" :style="stationColumnFlexWidthStyle">
        <font-awesome-icon class="icon" icon="map-marker-alt"></font-awesome-icon>
        STATIONS
      </div>
      <div id="route-header" :style="routeColumnFlexWidthStyle">
        <font-awesome-icon class="icon" icon="project-diagram"></font-awesome-icon>
        ROUTE
      </div>
      <div id="trip-header">
        <font-awesome-icon class="icon" icon="chart-area"></font-awesome-icon>
        TRIP
      </div>
    </div>

    <div class="container">
      <div id="baseline-container">
        <div v-for="id in selectedStationIDs" :key="id" class="baseline" :style="stationRowStyle"></div>
      </div>
      <div id="station-column" class="column" :style="stationColumnFlexWidthStyle">
        <div v-for="id in selectedStationIDs" :key="id" class="row" :style="stationRowStyle">
          <span class="name">{{ indexedStations[id].name }}</span>
        </div>
      </div>
      <div id="route-column" class="column" :style="routeColumnFlexWidthStyle">
        <div id="route-decoration-container">
          <div v-for="id in selectedStationIDs" :key="id" class="decoration" :style="stationRowStyle"></div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { Component, Vue, Watch } from 'vue-property-decorator'
import _ from 'lodash'

import Exploration from '@/store/modules/Exploration'

@Component
export default class RoutePanel extends Vue {
  public selectedStationIDs: number[] = []
  public rowHeight = 40
  public stationColumnWidth = 100
  public routeColumnWidth = 150

  public get indexedStations () {
    return Exploration.indexedStations
  }

  public get indexedRoutes () {
    return Exploration.indexedRoutes
  }

  public get selectedRoute () {
    return Exploration.selectedRoutes.length > 0 ? Exploration.selectedRoutes[0] : null
  }

  public get stationColumnFlexWidthStyle () {
    return {
      flex: `0 0 ${this.stationColumnWidth}px`
    }
  }

  public get routeColumnFlexWidthStyle () {
    return {
      flex: `0 0 ${this.routeColumnWidth}px`
    }
  }

  public get stationRowStyle () {
    return {
      height: `${this.rowHeight}px`,
      'line-height': `${this.rowHeight}px`
    }
  }

  @Watch('selectedRoute')
  private _onSelectedRouteChange () {
    if (this.selectedRoute) {
      this.selectedStationIDs = _.clone(this.selectedRoute.stations)
    } else {
      this.selectedStationIDs = []
    }
  }
}
</script>

<style lang="scss">
$backgroundColor: lighten(#e6dbc7, 12%);
$columnSpacing: 10px;

#route-panel {
  flex: 0 0 40%;
  z-index: 1;
  margin: 30px;
  background-color: $backgroundColor;
  border: 1px solid #ccc;
  border-radius: 10px;
  box-shadow: 0 0 20px #ccc;
  padding: 10px 15px;

  .headers {
    display: flex;
    flex-direction: row;

    & > div {
      font-size: 10px;
      font-weight: bold;
      color: #aaa;
      margin-right: $columnSpacing;

      .icon {
        margin-right: 5px;
      }
    }
  }

  .container {
    overflow: scroll;
    margin-top: 5px;
    display: flex;
    flex-direction: row;
    position: relative;

    .column {
      margin-right: $columnSpacing;
    }

    #baseline-container {
      position: absolute;
      left: 0;
      top: 0;
      width: 100%;

      .baseline {
        position: relative;
      }

      .baseline::after {
        content: '';
        display: block;
        position: absolute;
        width: 100%;
        top: 50%;
        left: 0;
        z-index: 5;
        border-top: 1px dashed #dedede;
      }
    }

    #station-column {
      .row {
        font-size: 12px;
        color: #333;
        position: relative;

        .name {
          position: relative;
          background-color: $backgroundColor;
          padding-right: 5px;
          z-index: 10;
        }
      }
    }

    #route-column {
      position: relative;
      z-index: 5;
      #route-decoration-container {
        position: absolute;
        width: 100%;
        .decoration {
          position: relative;
          &::before {
            content: '';
            position: absolute;
            bottom: 50%;
            width: 50%;
            height: 5px;
            transform: translateY(1px);
            border-left: 1px solid #ddd;
            border-right: 1px solid #ddd;
            border-bottom: 1px solid #ddd;
          }
          &::after {
            content: '';
            position: absolute;
            left: 50%;
            bottom: 50%;
            width: 50%;
            height: 5px;
            transform: translateY(1px);
            border-right: 1px solid #ddd;
            border-bottom: 1px solid #ddd;
          }
        }
      }
    }
  }
}
</style>
