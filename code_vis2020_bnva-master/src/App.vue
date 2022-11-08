<template>
  <div id="app">
    <NavBar></NavBar>
    <div id="views">
      <div id="matrix-view">
        <Matrix :reverse="false"></Matrix>
      </div>
      <div id="ranking-view">
        <Ranking></Ranking>
      </div>
      <Map></Map>
      <Panel />
    </div>
  </div>
</template>

<script lang="ts">
import 'reflect-metadata'
import { Component, Vue } from 'vue-property-decorator'

import NavBar from '@/components/NavBar.vue'
import Map from './components/explore/Map.vue'
import Matrix from '@/components/explore/Matrix.vue'
import Dataset from '@/store/modules/Dataset'
import Ranking from '@/components/ranking/Ranking.vue'
import Panel from '@/components/Panel/Panel.vue'

@Component({
  components: {
    Ranking,
    NavBar,
    Map,
    Matrix,
    Panel
  }
})
export default class App extends Vue {
  public beforeCreate () {
    Dataset.loadDataSources()
  }
}
</script>

<style lang="scss">
@font-face {
  font-family: "DIN Offc Pro";

  src: url("./assets/fonts/din.woff2") format("woff2"); /* Modern Browsers */
  font-weight: normal;
  font-style: normal;
}

@font-face {
  font-family: "DIN Offc Pro";

  src: url("./assets/fonts/din_bold.woff2") format("woff2"); /* Modern Browsers */
  font-weight: bold;
  font-style: normal;
}

body {
  margin: 0;
}
#app {
  width: 100vw;
  height: 100vh;
  font-family: 'DIN Offc Pro', Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  display: flex;
  flex-direction: column;

  #views {
    position: relative;
    flex: 1 1;
    display: flex;
    flex-direction: column;
    flex-wrap: wrap;
    width: 100%;
    overflow: hidden;

    #matrix-view {
      position: relative;
      height: 60%;
      width: 100%;
      // flex: 1 1 60%;
      // height: 60vh;
      display: flex;
      flex-direction: row-reverse;
      // align-self: end;
      // justify-content: flex-end;
    }

    #ranking-view {
      position: relative;
      height: 40%;
      width: 100%;
      // flex: 1 1 40%;
      // height: 40vh;
      display: flex;
      flex-direction: row;
    }
  }
}
</style>
