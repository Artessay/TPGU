<template>
  <div></div>
</template>

<script lang="ts">
import {Component, Prop, Vue, Watch} from 'vue-property-decorator';
import {DeferredPromise} from "../../utils";
import Evaluation from "@/store/modules/Evaluation";
import checkImage from '../../assets/check.png'
import yellowQuestionImage from '../../assets/yellowquestion.png'
import grayQuestionImage from '../../assets/grayquestion.png'
import mapboxgl from 'mapbox-gl'

@Component
export default class ConflictGraph extends Vue {
  @Prop()
  map!: DeferredPromise<mapboxgl.Map>

  public get conflictGraphLinkGeoJSON () {
    return Evaluation.conflictGraphGeoJSON.line
  }

  public get conflictGraphNodeGeoJSON () {
    return Evaluation.conflictGraphGeoJSON.point
  }

  public get conflictResolvedNum () {
    return Evaluation.conflictResolvedNum
  }

  public get geoJsonUpdateWatcher () {
    return Evaluation.geoJsonUpdateWatcher
  }

  public get highlightRoutesWatcher () {
    return Evaluation.highlightRoutesWatcher
  }

  public async mounted () {
    const map = await this.map.get()
    // const quenstionUrl = 'question.png'
    // const imageResp = await fetch(quenstionUrl)

    // console.log(imageResp)
    // const imageBlob = await imageResp.blob()
    // const imageBitmap = await createImageBitmap(imageBlob)
    // const canvas = document.createElement("canvas");
    // canvas.width = 100
    // canvas.height = 100
    // const ctx = canvas.getContext("2d") as CanvasRenderingContext2D
    // ctx.drawImage(imageBitmap, 0, 0);
    // const image = ctx.getImageData(0, 0, 100, 100)
    // map.addImage('questionImage', image)
    map.loadImage(checkImage, (error, image) => {
      if (error) throw error;
      map.addImage('check', image)
    })
    map.loadImage(yellowQuestionImage, (error, image) => {
      if (error) throw error;
      map.addImage('yellowquestion', image)
    })
    map.loadImage(grayQuestionImage, (error, image) => {
      if (error) throw error;
      map.addImage('grayquestion', image)
    })
  }

  @Watch('highlightRoutesWatcher', {immediate: true})
  public async drawHighlightRoutes () {
    console.log('[ConflictGraph] Redrawing highlight routes...')
    const map = await this.map.get()

    const oldSource = map.getSource('eval-highlight-routes')
    if (oldSource) {
      map.removeLayer('eval-highlight-routes')
      map.removeSource('eval-highlight-routes')
    }

    if (Evaluation.highlightRoutesGeoJSON) {
      map.addSource('eval-highlight-routes', {
        type: 'geojson',
        data: Evaluation.highlightRoutesGeoJSON
      })
      map.addLayer({
        id: 'eval-highlight-routes',
        type: 'line',
        source: 'eval-highlight-routes',
        paint: {
          'line-color': '#2c7fb8',
          'line-opacity': 0.5,
          'line-width': 2
        }
      })
      if (map.getLayer('matrixRoute')) {
        map.moveLayer('eval-highlight-routes', 'matrixRoute')
      }
    }
  }

  @Watch('geoJsonUpdateWatcher')
  public onGeoJsonUpdated () {
    this.drawConflictGraph()
  }

  @Watch('conflictResolvedNum', {immediate: true, deep: true})
  public async drawConflictGraph () {
    if (!this.conflictGraphLinkGeoJSON || !this.conflictGraphNodeGeoJSON) {
      return
    }
    console.log('[ConflictGraph] Redrawing conflict graph...')
    console.log(this.conflictGraphNodeGeoJSON, this.conflictGraphLinkGeoJSON)
    const map = await this.map.get()
    const oldLinkSource = map.getSource('conflictGraphLink')
    const oldNodeSource = map.getSource('conflictGraphNode')
    if (oldLinkSource && oldNodeSource) {
      oldLinkSource.setData(this.conflictGraphLinkGeoJSON)
      oldNodeSource.setData(this.conflictGraphNodeGeoJSON)
      return
    }

    if (this.conflictGraphLinkGeoJSON && this.conflictGraphNodeGeoJSON) {
      console.log(this.conflictGraphNodeGeoJSON)
      // @ts-ignore
      map.addSource('conflictGraphLink', {
        type: 'geojson',
        data: this.conflictGraphLinkGeoJSON
      })
      // @ts-ignore
      map.addSource('conflictGraphNode', {
        type: 'geojson',
        data: this.conflictGraphNodeGeoJSON
      })
      map.addLayer({
        id: 'conflictGraphLink',
        type: 'line',
        source: 'conflictGraphLink',
        paint: {
          'line-color': ['get', 'color'],
          'line-opacity': 0.7,
          'line-width': 3
        }
      })
      map.addLayer({
        id: 'conflictGraphNode',
        type: 'symbol',
        source: 'conflictGraphNode',
        layout: {
          'icon-image': ['get', 'marker'],
          'icon-size': 0.15
        }
      });
      map.on('mouseenter', 'conflictGraphNode', (e) => {
        if (e.features && e.features[0].id) {
          Evaluation.setHighLightRoutesByStation(parseInt(e.features[0].id.toString()))
        }
      })
      map.on('mouseleave', 'conflictGraphNode', () => {
        Evaluation.clearHighlightRoutes()
      })
      map.on('click', 'conflictGraphNode', (e) => {
        if (e.features && e.features[0].properties && e.features[0].properties.groupID) {
          console.log(e.features)
          Evaluation.selectGroup(e.features[0].properties.id)
        }
      })
    }
  }
}
</script>

<style scoped>

</style>
