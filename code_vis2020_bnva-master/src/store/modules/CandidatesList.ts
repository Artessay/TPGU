import store from '@/store'
import Vue from 'vue'
import { Action, getModule, Module, Mutation, VuexModule } from "vuex-module-decorators"
import { Attribute, NormalizerType } from "@/utils/Attribute"
import _ from 'lodash'
import Exploration, {Route} from "@/store/modules/Exploration"
import { AttributeFilter, AttributeGroup } from "@/utils/Types";
import { AttrDistribution } from '@/utils/AttrDistribution'
import {RouteList} from "@/utils/MCTSTypes";
import Evaluation from "@/store/modules/Evaluation";
import {Conflict} from "@/utils/RouteGroup";
import {ServiceCost} from "@/utils/Criterion";
import {TargetRoute} from "@/utils/targetRoute";

export interface ConflictInLineup {
  conflictID: number
  origin: number
  dest: number
  RouteGroupAlternatives: {[id: number]: number[]}
  candidatesDict: {[id: number]: Candidate[]}
  opened: boolean
  gids: number[]
}

export interface Candidate {
  aggregate: number
  routeID: number
  name: string
  attr: {
    [key: string]: number
  }
}

@Module({ name: 'CandidateStore', dynamic: true, namespaced: true, store })
class CandidatesList extends VuexModule {
  private _attributes: Attribute[] = [
    new Attribute('dist', 'ROUTE LENGTH', 'km', 1000),
    new Attribute('stationsNum', 'NUM OF STATIONS', '', 1),
    new Attribute('flow', 'PASSENGER VOL', '', 1),
    new Attribute('load', 'AVERAGE LOAD', '', 1, 3),
    new Attribute('directness', 'DIRECTNESS', '', 1, 3, NormalizerType.PADDED_REVERSED),
    new Attribute('serviceCost', 'SERVICE COST', 'k', 1000, 2, NormalizerType.PADDED_REVERSED)
  ]

  private _candidateRoutes: Candidate[] = [] // mock
  private _filterCandidates: Candidate[] = []
  private _removedCandidateRoutes: number[] = []
  private _removedAttributes: Attribute[] = []
  private _highlightRoute: Candidate | null = null
  private _sortingAttribute: Attribute | null = null
  private _showFilter = false
  private _filters: { [key: string]: { name: string, fn: Function} } = {}
  private _routeOptimized: Candidate | null = null
  private _headerResizing = false
  private _hoverRoute = -1
  private _searching = false
  private _conflicts: Candidate[][][] = []
  private _conflictsInLineup: ConflictInLineup[] = []
  private _openedConflictIndex = -1
  private _openedConflictID = -2
  private _targetRoute : TargetRoute | null = null
  private _changeTargetWatcher = 0

  public get changeTargetWatcher () {
    return this._changeTargetWatcher
  }

  public get finished () {
    return (
      this._conflictsInLineup.length === 1 &&
      this._conflictsInLineup[0].gids.length === 1 &&
      this._conflictsInLineup[0].candidatesDict[this._conflictsInLineup[0].gids[0]].length === 1
    )
  }

  @Mutation
  public incChangeTargetWatcher () {
    this._changeTargetWatcher++
  }

  public get targetRoute () {
    return this._targetRoute
  }

  public get headerResizing () {
    return this._headerResizing
  }

  public get hoverRoute () {
    if (this._hoverRoute >= 0) {
      return this._hoverRoute
    }
  }

  @Mutation
  public setHoverRoute(rid: number) {
    this._hoverRoute = rid
  }

  public get filterCandidates () {
    return this._filterCandidates
  }

  public get candidateRoutes () {
    return this._candidateRoutes
  }

  public get removedCandidateRoutes () {
    return this._removedCandidateRoutes
  }

  public get attributes () {
    return this._attributes
  }

  public get highlightCandidate () {
    return this._highlightRoute
  }

  public get showFilter () {
    return this._showFilter
  }

  public get routeOptimized () {
    return this._routeOptimized
  }

  public get conflicts () {
    return this._conflicts
  }

  public get conflictsInLineup () {
    return this._conflictsInLineup
  }

  public get searching () {
    return this._searching
  }

  public get openedConflictIndex () {
    return this._openedConflictIndex
  }

  public get openedConflictID () {
    return this._openedConflictID
  }

  @Action({ commit: '_setConflictsToLineup' })
  public setConflictsToLineup (conflicts: Conflict[]) {
    return conflicts
  }

  @Mutation
  private _setConflictsToLineup (conflicts: Conflict[]) {
    this._conflictsInLineup = conflicts.map((conflict) => {
      const candidatesDict: {[id: number]: Candidate[]} = {}
      const gids = Object.keys(conflict.RouteGroupAlternatives).map((idstr) => +idstr)
      gids.forEach((gid) => {
        // @ts-ignore
        candidatesDict[gid] = Evaluation.groupCollection.candidatesByGroupID(gid)
        console.log('candidatesDict[gid]', candidatesDict[gid])
      })
      return {
        ...conflict,
        candidatesDict,
        gids,
        opened: false
      }
    })
    console.log(this._conflictsInLineup)
  }

  @Action({ commit: '_optimizeRoute' })
  public optimizeRoute (targets: number[]) {
    this.context.dispatch('clearCandidates')
    return targets
  }

  @Mutation
  private _optimizeRoute (targets: TargetRoute) {
    this._routeOptimized = this._highlightRoute
    // hack: adjust the criterion\
    if (this._routeOptimized && targets && targets.selectedStations) {
      const matrix = Exploration.indexedRoutes[this._routeOptimized.routeID].matrix
      let flow = 0
      const stops = targets.targetStops()
      _.each(matrix, (value, i) => {
        _.each(value, (v, j) => {
          // @ts-ignore
          if (stops.has(parseInt(j)) && stops.has(parseInt(i))) {
            flow += v
          }
        })
      })
      this._routeOptimized.attr.flow = flow
      this._routeOptimized.attr.time = this._routeOptimized.attr.time * (targets.selectedStations.size / targets.route.stations.length)
      this._routeOptimized.attr.serviceCost = this._routeOptimized.attr.serviceCost * (targets.selectedStations.size / targets.route.stations.length)
      console.log('route optimize； ', flow)
    }
  }

  @Action({ commit: '_addCandidates' })
  public onOpenConflict (index: number) {
    let openedConflictIndex = -1
    let cList: Candidate[] = []
    if (this._openedConflictIndex !== index && index !== -1) {
      const openedConflictInLineup: ConflictInLineup = this._conflictsInLineup[index]
      // cList = _.flatten(conflict)
      cList = _.reduce(openedConflictInLineup.candidatesDict, (result, value, key) => {
        return result.concat(value)
      }, [] as Candidate[])

      openedConflictIndex = index
    }
    this.context.commit('_clearCandidates')
    // this.context.commit('_addCandidates', cList)
    this.context.commit('_onOpenConflict', openedConflictIndex)

    return cList
  }

  @Mutation
  private _onOpenConflict (index: number) {
    console.log(index)
    this._openedConflictIndex = index
    if (index >= 0) {
      this._openedConflictID = this._conflictsInLineup[index].conflictID
    } else {
      this._openedConflictID = -2
    }
    Evaluation.updateConflictGeoJson()
  }

  @Action({ commit: '_search' })
  public search (flag: boolean) {
    return flag
  }

  @Mutation
  private _search (flag: boolean) {
    this._searching = flag

    // hack conflicts
    // const conflict1 = [this._filterCandidates.slice(0, 3), this._filterCandidates.slice(3, 12), this._filterCandidates.slice(12, 13), this._filterCandidates.slice(14, 16), this._filterCandidates.slice(16, 19), this._filterCandidates.slice(19, 23)]
    // const conflict2 = [this._filterCandidates.slice(13, 19), this._filterCandidates.slice(19, 22)]
    // this._conflicts = [conflict1, conflict2]
  }

  @Action({ commit: '_changeHeaderResizing' })
  public changeHeaderResizing () {
    return undefined
  }

  @Mutation
  private _changeHeaderResizing () {
    this._headerResizing = !this._headerResizing
  }

  @Action({ commit: '_reorderHeaders' })
  public reorderHeaders (attr: Attribute) {
    return attr
  }

  @Mutation
  private _reorderHeaders(attr: Attribute) {
    const pos = this._attributes.indexOf(attr)
    const newAttributes = [attr]
    _.each(this._attributes, (a, i) => {
      if (i !== pos) {
        newAttributes.push(a)
      }
    })
    this._attributes = newAttributes
  }

  @Action({ commit: '_sortConflict' })
  public sortConflict (attr: Attribute) {
    return attr
  }

  @Mutation
  private _sortConflict(attr: Attribute) {
    const openedConflictIndex = this._openedConflictIndex
    const conflictInLineup: ConflictInLineup = this._conflictsInLineup[openedConflictIndex]
    const originGids = conflictInLineup.gids

    console.log(attr)
    console.log(originGids)
    // handle current attr
    this._sortingAttribute = attr
    if (!attr.sorted) {
      attr.sorted = 'down'
    } else if (attr.sorted === 'down') {
      attr.sorted = 'up'
    } else {
      attr.sorted = 'down'
    }

    // handle other attrs
    let keys = [attr]
    if (attr.group) {
      keys = attr.group.groupAttrs
      _.each(keys, k => { k.sorted = attr.sorted })
    } else {
      _.each(this._attributes, a => {
        if (a.key !== attr.key) {
          a.sorted = null
        }
      })
    }

    let newGids: number[] = []
    if (attr.sorted === 'up') {
      newGids = _.sortBy(originGids, gid => {
        const cs = conflictInLineup.candidatesDict[gid]
        return _.mean(attr.getBoxplotValuesOfCandidates(cs))
      })
      // conflict = _.sortBy(conflict, cs => _.mean(attr.getBoxplotValuesOfCandidates(cs)))
    } else {
      newGids = _.sortBy(originGids, gid => {
        const cs = conflictInLineup.candidatesDict[gid]
        return -_.mean(attr.getBoxplotValuesOfCandidates(cs))
      })
      // conflict = _.sortBy(conflict, cs => -_.mean(attr.getBoxplotValuesOfCandidates(cs)))
    }
    console.log(newGids)
    conflictInLineup.gids = newGids
    // Vue.set(this._conflicts, openedConflictIndex, conflict)
  }

  @Action({ commit: '_sortCandidates' })
  public sortCandidates (attr: Attribute) {
    return attr
  }

  @Mutation
  private _sortCandidates(attr: Attribute) {
    console.log(attr)
    this._sortingAttribute = attr
    if (!attr.sorted) {
      attr.sorted = 'down'
    } else if (attr.sorted === 'down') {
      attr.sorted = 'up'
    } else {
      attr.sorted = 'down'
    }

    let keys = [attr]
    if (attr.group) {
      keys = attr.group.groupAttrs
      _.each(keys, k => { k.sorted = attr.sorted })
    } else {
      _.each(this._attributes, a => {
        if (a.key !== attr.key) {
          a.sorted = null
        }
      })
    }

    _.each(this._candidateRoutes, c => {
      c.aggregate = 0
      _.each(keys, k => {
        c.aggregate += k.normalizer(c) * k.width
      })
    })

    if (attr.sorted === 'up') {
      this._candidateRoutes = _.sortBy(this._candidateRoutes, c => c.aggregate)
      this._filterCandidates = _.sortBy(this._filterCandidates, c => c.aggregate)
    } else {
      this._candidateRoutes = _.sortBy(this._candidateRoutes, c => -c.aggregate)
      this._filterCandidates = _.sortBy(this._filterCandidates, c => -c.aggregate)
    }
  }

  @Mutation
  public changeAttributesForManipulation () {
    this._attributes = [
      new Attribute('time', 'SERVICE TIME', '', 1, 0, NormalizerType.PADDED_REVERSED),
      new Attribute('flow', 'PASSENGER FLOW', '', 1),
      new Attribute('directness', 'DIRECTNESS', '', 1, 3, NormalizerType.PADDED_REVERSED),
      new Attribute('constructCost', 'CONSTRUCTION COST', '', 1, 0, NormalizerType.PADDED_REVERSED),
      new Attribute('serviceCost', 'SERVICE COST', 'k', 1000, 2, NormalizerType.PADDED_REVERSED)
    ]
  }

  @Action({ commit: '_addCandidates' })
  public addPossibleSolutions(list: RouteList | null) {
    if (list) {
      const cList: Candidate[] = []
      _.each(list.routes, route => {
        if (route.criteria.length >= 3) {
          const c = {
            routeID: route.id,
            name: '#Route ' + route.id.toString(),
            aggregate: 0,
            attr: {
              time: route.criteria[0],
              flow: route.criteria[1],
              directness: route.criteria[2],
              constructCost: route.constructCost,
              serviceCost: route.serviceCost
            }
          }
          cList.push(c)
        }
      })

      return cList
    }
  }

  @Action({ commit: '_addCandidates' })
  public addCandidates ({indexedRoutes, newCandidates}: { indexedRoutes: { [id: number]: Route }, newCandidates: number[] }) {
    const cList : Candidate[] = []
    const serviceCost = new ServiceCost()
    _.each(
      newCandidates,
      r => {
        const c = {
          routeID: r,
          name: indexedRoutes[r].name,
          aggregate: 0,
          attr: {
            dist: indexedRoutes[r].length,
            stationsNum: indexedRoutes[r].stations.length,
            flow: indexedRoutes[r].checkin + indexedRoutes[r].checkout,
            load: Math.log(indexedRoutes[r].load + 1),
            directness: indexedRoutes[r].directness,
            serviceCost: serviceCost.value(indexedRoutes[r].time),
            time: indexedRoutes[r].time
          }
        }
        cList.push(c)
        // _.each(
        //   this._attributes,
        //   (attr, key) => attr.updater(c)
        // )
      }
    )

    return cList
  }

  @Action({ commit: '_clearCandidates' })
  public clearCandidates () {
    // console.log('[Lineup: Action]: clearCandidates')
    return undefined
  }

  @Action({ commit: '_toggleHighLightRoute' })
  public toggleHighLightRoute(c : Candidate | null) {
    return c
  }

  @Action( { commit: '_ToggleAttributeGrouping' })
  public ToggleAttributeGrouping(attr: Attribute) {
    return attr
  }

  // @Action({ commit: '_resizeAttribute '})
  // public resizeAttribute ({ attr, delta }: { attr: Attribute, delta: number }) {
  //   return {attr: attr, delta: delta}
  // }

  @Mutation
  private _addCandidates (newCandidates: Candidate[]) {
    // console.log('this._candidateRoutes before add', this._candidateRoutes)
    this._candidateRoutes = _.concat(this._candidateRoutes, newCandidates)
    this._showFilter = true
    this._filterCandidates = this._candidateRoutes
    // console.log('[CandidateList]: add new Candidates', this._candidateRoutes)
    if (this._routeOptimized && (!this._searching)) {
      // console.log(this._routeOptimized)
    }

    // initialize attributes min and max
    _.each(
      this._attributes,
      (attr, key) => attr.initializeMinMax()
    )
    this._candidateRoutes.forEach((c) => {
      _.each(
        this._attributes,
        (attr, key) => attr.updater(c))
    })

    // hack for routeOptimized
    // console.log(this._searching, this.searching)
    if (this._routeOptimized && (!this._searching)) {
      _.each(
        this._attributes,
        (attr, key) => attr.forceUpdaterBasedOptimizingRoute(this._routeOptimized as Candidate)
      )
    }

    _.each(this._attributes, attr => {
      // console.log('initialize attrDistribution')
      attr.attrDistribution = new AttrDistribution(attr)
      attr.attrDistribution.initialize(this._candidateRoutes)
      // console.log(attr)
    })
  }

  @Mutation
  private _deleteCandidate (candidate: number) {
    this._removedCandidateRoutes.push(candidate)
  }

  @Mutation
  private _clearCandidates() {
    // console.log('[Lineup: mutation]: clearCandidates')
    this._candidateRoutes = []
    this._filterCandidates = []
    this._attributes.forEach((attr) => {
      attr.attrDistribution = new AttrDistribution(attr)
    })
  }

  // @Mutation
  // private _resizeAttribute({ attr, delta }: { attr: Attribute, delta: number}) {
  //   if (attr.width + delta < 40) {
  //     return
  //   }
  //   attr.width += delta
  // }

  @Mutation
  private _toggleHighLightRoute(c : Candidate | null) {
    this._highlightRoute = c
    if (c) {
      this._targetRoute = new TargetRoute(Exploration.indexedRoutes[c.routeID], Exploration.indexedStations)
    } else {
      this._targetRoute = null
    }
  }

  @Mutation
  private _ToggleAttributeGrouping(attr: Attribute) {
    console.log('toggle group:')
    console.log(attr)
    if (attr.group) {
      attr.group.removeAttribute(attr)
    } else {
      attr.group = new AttributeGroup([attr])
      const pos = this._attributes.indexOf(attr)
      if (pos !== 0 && this._attributes[pos - 1].group) {
        attr.group.prependGroup(this._attributes[pos - 1].group)
      }
      if (pos !== this._attributes.length - 1 && this._attributes[pos + 1].group) {
        attr.group.appendGround(this._attributes[pos + 1].group)
      }
    }
  }

  @Mutation
  public toggleRankingFilter () {
    if (this._showFilter) {
      this._showFilter = false
    } else {
      this._showFilter = true
      this._filterCandidates = this._candidateRoutes

      _.each(this._attributes, attr => {
        attr.attrDistribution = new AttrDistribution(attr)
        attr.attrDistribution.initialize(this._candidateRoutes)
      })
    }
  }

  @Action({ commit: '_setAttributeFilterRange' })
  public moveAttributeFilterRange({left, attr, delta}: {left: boolean, attr: Attribute, delta: number}) {
    // @ts-ignore
    return {
      attr: attr,
      range: [
        attr.attrDistribution.range[0] + (left ? delta : 0),
        attr.attrDistribution.range[1] + (left ? 0 : delta)
      ]
    }
  }

  @Action({ commit: '_setAttributeFilterRange' })
  public setAttributeFilterRange({attr, range} : {attr: Attribute, range: number[]}) {
    return {attr, range}
  }

  @Mutation
  private _setAttributeFilterRange ({attr, range} : {attr: Attribute, range: number[]}) {
    if (attr.attrDistribution) {
      attr.attrDistribution.range = range
    }
  }

  @Mutation
  public applyAttributeFilter () {
    // @ts-ignore
    const allFn = _(this._attributes)
      .filter(attr => !attr.attrDistribution.pristine)
      .map(attr => attr.attrDistribution.testFunc)
      .value()
    const testFn = (c: Candidate) => _.every(allFn, fn => fn(c))

    // @ts-
    // _.each(
    //   this._attributes,
    //   attr => attr.attrDistribution.updateShadow(testFn)
    // )
    this._filterCandidates = _.filter(
      this._candidateRoutes,
      testFn
    )
    if (!Evaluation.start) {
      Exploration.setHighlightRoutesGeoJSON({routes: _.map(this._filterCandidates, c => c.routeID), selected: true})
    }
  }

  @Action({ commit: '_applyAttributeFilterWhenSearching' })
  public applyAttributeFilterWhenSearching() {
    return undefined
  }

  @Mutation
  private _applyAttributeFilterWhenSearching () {
    console.log('_applyAttributeFilterWhenSearching')
    const openedConflict = this._conflictsInLineup[this._openedConflictIndex]
    const allFn = _(this._attributes)
      .filter(attr => !attr.attrDistribution.pristine)
      .map(attr => attr.attrDistribution.intersectFunc)
      .value()
    const testFn = (cs: Candidate[]) => _.every(allFn, fn => fn(cs))

    console.log(openedConflict.candidatesDict)

    const allGids = Object.keys(openedConflict.candidatesDict).map(gid => +gid)
    const newGids = _.filter(
      allGids,
      gid => testFn(openedConflict.candidatesDict[gid])
    )
    openedConflict.gids = newGids
    console.log(allGids, newGids)
  }
}

export default getModule(CandidatesList)
