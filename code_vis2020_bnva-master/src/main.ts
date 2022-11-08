import Vue from 'vue'
import App from './App.vue'
import store from './store'

import { library } from '@fortawesome/fontawesome-svg-core'
import {
  faBus,
  faProjectDiagram,
  faEye,
  faMapMarkerAlt,
  faChartArea,
  faMagic,
  faUndo,
  faRedo,
  faSlidersH,
  faTimes,
  faStop,
  faPlay,
  faPause,
  faSortAmountUpAlt,
  faSortAmountUp,
  faThumbtack,
  faGripVertical,
  faSearch,
  faCube,
  faRoute
} from '@fortawesome/free-solid-svg-icons'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'

library.add(faThumbtack)
library.add(faSortAmountUpAlt)
library.add(faSortAmountUp)
library.add(faBus)
library.add(faChartArea)
library.add(faEye)
library.add(faMapMarkerAlt)
library.add(faProjectDiagram)
library.add(faGripVertical)
library.add(faMagic)
library.add(faUndo)
library.add(faRedo)
library.add(faSlidersH)
library.add(faTimes)
library.add(faSearch)
library.add(faStop)
library.add(faPlay)
library.add(faPause)
library.add(faCube)
library.add(faRoute)
Vue.component('font-awesome-icon', FontAwesomeIcon)

Vue.config.productionTip = false

new Vue({
  store,
  render: h => h(App)
}).$mount('#app')
