<template>
  <div id="map-tooltip" :style="tooltipPositionStyle">
    <span class="hint">{{ pinned ? 'Select a route...' : 'Click to pin' }}</span>
    <div v-for="feat in features" :key="feat.properties.name" class="route" @click="toggleRoute(feat.properties.id)">
      <div class="left">
        <font-awesome-icon class="icon" icon="project-diagram"></font-awesome-icon>
        <span>{{ feat.properties.name }}</span>
      </div>
      <div class="right">
        <font-awesome-icon v-if="indexedRoutes[feat.properties.id].states.selected" class="icon" icon="eye"></font-awesome-icon>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { Component, Vue, Prop } from 'vue-property-decorator'
import mapboxgl from 'mapbox-gl'
import Exploration from '../../store/modules/Exploration'

@Component
export default class MapTooltip extends Vue {
  @Prop() pinned!: boolean
  @Prop() routeFeatures!: mapboxgl.MapboxGeoJSONFeature[]
  @Prop() mousePosition!: { x: number, y: number } | null

  public cachedRouteFeatures!: mapboxgl.MapboxGeoJSONFeature[]
  public cachedMousePosition!: { x: number, y: number } | null

  public get features () {
    if (this.pinned) {
      return this.cachedRouteFeatures
    } else {
      this.cachedRouteFeatures = this.routeFeatures
      return this.routeFeatures
    }
  }

  public get position () {
    if (this.pinned) {
      return this.cachedMousePosition
    } else {
      this.cachedMousePosition = this.mousePosition
      return this.mousePosition
    }
  }

  public get indexedRoutes () {
    return Exploration.indexedRoutes
  }

  public get visible () {
    return this.pinned || (this.position != null && this.features.length !== 0)
  }

  public get tooltipPositionStyle () {
    return {
      position: 'fixed',
      left: this.visible ? `${this.position!.x + 10}px` : 0,
      top: this.visible ? `${this.position!.y + 10}px` : 0,
      display: this.visible ? 'block' : 'none',
      'pointer-events': this.pinned ? 'all' : 'none',
      opacity: this.pinned ? 0.9 : 0.8
    }
  }

  public toggleRoute (routeId: number) {
    const route = this.indexedRoutes[routeId]
    Exploration.toggleRoute(route)
  }
}
</script>

<style lang="scss">
#map-tooltip {
  width: 150px;
  background-color: darken(#c1ccd7, 30%);
  border-radius: 5px;
  padding-bottom: 5px;
  transition: opacity 200ms;

  .hint {
    color: #ccc;
    margin: 0 10px;
    font-size: 10px;
    font-style: italic;
  }

  .route {
    height: 25px;
    line-height: 25px;
    padding: 0 10px;
    color: white;
    font-size: 12px;
    cursor: pointer;
    display: flex;
    flex-direction: row;
    background-color: darken(#c1ccd7, 30%);
    transition: background-color 200ms;

    .left {
      flex: 1 1 0;
      min-width: 0;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;

      .icon {
        font-size: 10px;
        margin-right: 5px;
      }
    }

    .right {
      flex: 0 0 auto;

      .icon {
        font-size: 10px;
        margin-right: 5px;
      }
    }

    &:hover {
      background-color: darken(#c1ccd7, 20%);
    }
  }
}
</style>
