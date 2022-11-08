import _ from 'lodash'
import {SRoute, Trip} from "./MCTSTypes";
import CandidatesList, {Candidate} from "@/store/modules/CandidatesList";
import {Station} from "@/store/modules/Exploration";
import {GeoJSON} from "geojson";
import index from "@/store";
import {gqlQuery} from "@/utils/GraphqlAPI";
import {CHECKIN_TRIPS_FOR_STATION, CHECKOUT_TRIPS_FOR_STATION, F_MATRIX} from "@/query";

const MAX_DIFFERENCE_IN_GROUP = 3

function stdDev(scores : number[]) {
  const mean = _.mean(scores)
  let v1 = 0.0
  let v2 = 0.0
  scores.forEach(s => {
    v1 += (s - mean) * (s - mean)
    v2 += (s - mean)
  })
  v2 = v2 * v2 / scores.length
  let variance = (v1 - v2) / (scores.length - 1)
  if (variance < 0) {
    variance = 0
  }
  return Math.sqrt(variance)
}

export interface Conflict {
  conflictID: number
  origin: number
  dest: number
  RouteGroupAlternatives: { [id: number]: number[]}
}

export class RouteGroup {
  public stationVector: number[] = []
  // negative means no conflict
  public conflictVector: number[] = []
  public highlight = true

  public constructor(
    public routes: SRoute[],
    public candidates: Candidate[],
    stationNum: number,
    public id: number,
    public stationIndex: {[order: number]: number},
    public orderStations: number[]) {
    this.stationVector = new Array<number>(stationNum)
    this.conflictVector = new Array<number>(stationNum)
    for (let i = 0; i < stationNum; i++) {
      this.stationVector[i] = 0
      this.conflictVector[i] = -1
    }
  }

  public merge(group: RouteGroup, stationNum: number, id: number) {
    // Create the new group
    const newGroup = new RouteGroup(
      _.concat(this.routes, group.routes),
      _.concat(this.candidates, group.candidates),
      stationNum,
      id,
      this.stationIndex,
      this.orderStations)
    // compute the station vector for the merged group
    _.each(this.stationVector, (value, i) => {
      const add = value + group.stationVector[i]
      if (add === 2) {
        newGroup.stationVector[i] = 1
      }
    })
    return newGroup
  }

  public setStationVector(vector: number[]) {
    this.stationVector = vector
  }

  // difference between 2 route group
  public difference(group: RouteGroup) {
    let diff = 0
    _.each(this.stationVector, (value, i) => {
      const add = value + group.stationVector[i]
      if (add === 1) {
        diff += 1
      }
    })
    return diff
  }

  // Is this group include another group
  public include(group: RouteGroup | null) {
    if (!group) {
      return true
    }
    let flag = true
    _.each(group.stationVector, (value, i) => {
      const diff = this.stationVector[i] - value
      if (diff < 0) {
        flag = false
        return flag
      }
    })
    return flag
  }

  public share(group: RouteGroup) {
    let share = 0
    _.each(this.stationVector, (value, i) => {
      const add = value + group.stationVector[i]
      if (add === 2) {
        share += 1
      }
    })
    return share
  }

  public shareWithVector(vector: number[]) {
    if (vector.length !== this.stationVector.length) {
      return null
    }
    const result = new Array<number>(this.stationVector.length)
    _.each(this.stationVector, (value, i) => {
      if (value + vector[i] === 2) {
        result[i] = 1
      } else {
        result[i] = 0
      }
    })
    return result
  }

  public unionWithVector(vector: number[]) {
    const result = new Array<number>(this.stationVector.length)
    _.each(this.stationVector, (value, i) => {
      if (value + vector[i] > 0) {
        result[i] = 1
      } else {
        result[i] = 0
      }
    })
    return result
  }

  public conflictStations(cPoints: [number, number]) {
    const stations : number[] = []
    for (let i = cPoints[0] + 1; i < cPoints[1]; i++) {
      if (this.stationVector[i] === 1) {
        stations.push(this.orderStations[i])
      }
    }
    return stations
  }

  public tagConflictVector(conflicts: Conflict[]) {
    conflicts.forEach(conflict => {
      const origin = this.stationIndex[conflict.origin]
      const dest = this.stationIndex[conflict.dest]
      for (let i = origin + 1; i < dest; i++) {
        this.conflictVector[i] = conflict.conflictID
      }
    })
  }

  public stationGeoJSON (indexedStations: {[id: number]: Station}) {
    const features: GeoJSON.Feature<GeoJSON.Point>[] = []
    _.each(this.stationVector, (val, i) => {
      if (val === 1 && this.conflictVector[i] >= 0) {
        // @ts-ignore
        features.push(this.pointFeature(i, indexedStations, true, this.conflictVector[i]))
      } else if (val === 1) {
        // @ts-ignore
        features.push(this.pointFeature(i, indexedStations, false, -1))
      }
    })
    return features
  }

  public polylineGeoJSON (indexedStations: {[id: number]: Station}) {
    const features: GeoJSON.Feature<GeoJSON.LineString>[] = []
    let lastIndex = -1
    let isConflict = false
    _.each(this.stationVector, (val, i) => {
      if (val === 1 && lastIndex < 0) {
        lastIndex = i
      } else if (val === 1 && this.conflictVector[i] >= 0) {
        // conflict station
        // @ts-ignore
        features.push(this.lineFeature(lastIndex, i, indexedStations, true, this.conflictVector[i]))
        isConflict = true
        lastIndex = i
      } else if (val === 1 && this.conflictVector[i] < 0 && isConflict) {
        isConflict = false
        // @ts-ignore
        features.push(this.lineFeature(lastIndex, i, indexedStations, true, this.conflictVector[lastIndex]))
        lastIndex = i
      } else if (val === 1 && this.conflictVector[i] < 0 && !isConflict) {
        // @ts-ignore
        features.push(this.lineFeature(lastIndex, i, indexedStations, false, -1))
        lastIndex = i
      } else if (val === 0 && this.conflictVector[i] >= 0) {
        isConflict = true
      }
    })
    return features
  }

  private lineFeature(i : number, j : number, indexedStations: {[id: number]: Station}, conflict: boolean, conflictID: number) {
    const station1 = indexedStations[this.orderStations[i]]
    const station2 = indexedStations[this.orderStations[j]]
    const openedConflictID = CandidatesList.openedConflictID
    let color = ''
    if (!conflict) {
      color = '#7585d7'
    } else {
      if (conflictID === openedConflictID) {
        color = '#ffd078'
      } else {
        color = '#ccc'
      }
    }
    return {
      id: 0,
      type: 'Feature',
      geometry: {
        type: 'LineString',
        coordinates: [
          station1.coordinates,
          station2.coordinates
        ]
      },
      properties: {
        conflict,
        conflictID,
        color
      }
    }
  }

  private pointFeature(i: number, indexedStations: {[id: number]: Station}, conflict: boolean, conflictID: number) {
    const station = indexedStations[this.orderStations[i]]
    const openedConflictID = CandidatesList.openedConflictID
    let marker = ''
    if (!conflict) {
      marker = 'check'
    } else {
      if (conflictID === openedConflictID && this.highlight) {
        marker = 'yellowquestion'
      } else {
        marker = 'grayquestion'
      }
    }
    return {
      id: station.id,
      type: 'Feature',
      geometry: {
        type: 'Point',
        coordinates: station.coordinates
      },
      properties: {
        id: station.id,
        marker,
        groupID: this.id,
        conflict,
        conflictID
      }
    }
  }
}

export class RouteGroupCollection {
  public stationNum = 0
  // beta
  public minimumGroupNum = 3
  // all groups
  public routeGroups: RouteGroup[] = []
  public indexRouteGroups: { [id: number]: RouteGroup } = {}
  public stationIndex: {[id: number]: number} = {}
  public conflicts: Conflict[] = []
  private idx = 0
  public currentRoute: number[] = []

  public linkGeoJSONFeatures () {
    let features: GeoJSON.Feature<GeoJSON.LineString>[] = []
    this.routeGroups.forEach(group => {
      features = _.concat(features, group.polylineGeoJSON(this.indexedStations))
    })
    return features
  }

  public pointGeoJSONFeatures () {
    let features: GeoJSON.Feature<GeoJSON.Point>[] = []
    let highlightGroup : RouteGroup | null = null
    let noHighlight = true
    this.routeGroups.forEach(group => {
      if (!group.highlight) {
        noHighlight = false
      } else {
        highlightGroup = group
      }
      features = _.concat(features, group.stationGeoJSON(this.indexedStations))
    })
    if (!noHighlight && highlightGroup) {
      // @ts-ignore
      features = _.concat(features, highlightGroup.stationGeoJSON(this.indexedStations))
    }
    return features
  }

  public candidatesByGroupID (gid: number) {
    return this.indexRouteGroups[gid].candidates
  }

  public constructor(
    public alternatives: SRoute[],
    public orderedStations: number[],
    public indexedStations: {[id: number]: Station}
  ) {
    // orderedStations: stations in order
    // stationIndex: map station ID to order index
    for (let i = 0; i < orderedStations.length; i++) {
      this.stationIndex[orderedStations[i]] = i
    }
    this.constructGroups()
  }

  public static constructCandidateWrap (route: SRoute) {
    return {
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
  }

  private constructGroups () {
    this.stationNum = this.orderedStations.length
    // construct route-station vectors
    const groups : RouteGroup[] = []
    _.each(this.alternatives, (route) => {
      const group = new RouteGroup(
        [route],
        [RouteGroupCollection.constructCandidateWrap(route)],
        this.stationNum,
        this.idx,
        this.stationIndex,
        this.orderedStations
      )
      this.idx++
      const vector = new Array<number>(this.stationNum)
      for (let i = 0; i < this.stationNum; i++) {
        vector[i] = 0
      }
      _.each(route.r, station => {
        vector[this.stationIndex[station.id]] = 1
        group.setStationVector(vector)
      })
      groups.push(group)
    })
    // merge groups from bottom-up
    this.routeGroups = this.mergeGroups(groups)
    console.log(this.routeGroups)
    this.routeGroups.forEach(group => {
      this.indexRouteGroups[group.id] = group
    })
  }

  private mergeGroups (groups: RouteGroup[]) {
    // Stop condition 1: the number of groups less than BETA
    if (groups.length <= this.minimumGroupNum) {
      console.log('return because groups length <= ' + this.minimumGroupNum)
      return groups
    }
    const groupNum = groups.length
    // Find a pair of groups to merge
    let maxShare = 0
    let maxSharePairSet : Set<[number, number]> = new Set()
    for (let i = 0; i < groupNum; i++) {
      const group1 = groups[i]
      for (let j = i + 1; j < groupNum; j++) {
        const group2 = groups[j]
        const share = group1.share(group2)
        if (share > maxShare) {
          maxShare = share
          maxSharePairSet = new Set<[number, number]>([[i, j]])
        } else if (share === maxShare) {
          maxSharePairSet.add([i, j])
        }
      }
    }
    const candidateGroups : RouteGroup[] = []
    if (maxSharePairSet.size > 0) {
      maxSharePairSet.forEach(pair => {
        const newGroup = groups[pair[0]].merge(groups[pair[1]], this.stationNum, this.idx)
        for (let i = 0; i < groupNum; i++) {
          if (!groups[i].include(newGroup)) {
            candidateGroups.push(newGroup)
            this.idx++
            break
          }
        }
      })
    }
    if (candidateGroups.length === 0) {
      return groups
    }
    // compare merged groups by criterion
    let minDev = Infinity
    let minDevGroup : RouteGroup | null = null
    _.maxBy(candidateGroups, group => {
      let new_group : RouteGroup | null = null
      groups.forEach(other => {
        if (other.include(group)) {
          if (!new_group) {
            new_group = other
            new_group.id = group.id
          } else {
            new_group = new_group.merge(other, this.stationNum, group.id)
          }
        }
      })
      if (new_group) {
        group = new_group
      }
      const scores : number[] = []
      group.routes.forEach(r => {
        const score = _.reduce(r.criteria, (sum, val, i) => {
          return sum + val * r.weight[i]
        }, 0)
        scores.push(score)
      })
      const dev = stdDev(scores)
      if (dev < minDev) {
        minDev = dev
        minDevGroup = group
      }
    })
    // filter existing groups
    // @ts-ignore
    const newGroups : RouteGroup[] = [minDevGroup]
    groups.forEach(group => {
      if (!group.include(minDevGroup)) {
        newGroups.push(group)
      } else {
        console.log('give up')
      }
    })

    return this.mergeGroups(newGroups)
  }

  public searchConflicts() {
    if (this.routeGroups.length > 1) {
      // Find the intersection vector
      let vector = this.routeGroups[0].stationVector
      let union = this.routeGroups[0].stationVector
      for (let i = 1; i < this.routeGroups.length; i++) {
        const v = this.routeGroups[i].shareWithVector(vector)
        union = this.routeGroups[i].unionWithVector(union)
        if (v) {
          vector = v
        } else {
          console.log('[Route Group] intersection failed...')
          return []
        }
      }
      console.log("group intersection vector: " + vector)
      console.log("group union vector: " + union)
      // Find conflicts
      let last = -1
      let lastVal = 0
      let conflictID = 0
      let currentConflict : Conflict | null = null
      this.currentRoute = []
      const conflicts : Conflict[] = []
      _.each(vector, (val, i) => {
        if (i !== 0 && val === 1 && lastVal === 0) {
          this.currentRoute.push(this.orderedStations[i])
          // the end of the conflict
          if (currentConflict && currentConflict.origin >= 0) {
            currentConflict.dest = this.orderedStations[i]
            conflicts.push(currentConflict)
            currentConflict = null
          } else {
            console.error("Error detecting the conflict tail")
          }
        } else if (i !== 0 && val === 0 && lastVal === 1) {
          // the start of the conflicts
          currentConflict = {
            conflictID: conflictID,
            origin: this.orderedStations[last],
            dest: -1,
            RouteGroupAlternatives: {}
          }
          conflictID++
        } else if (i !== 0 && val === 1 && lastVal === 1) {
          this.currentRoute.push(this.orderedStations[i])
        } else if ( i === 0 ) {
          // the origin station
          this.currentRoute.push(this.orderedStations[i])
        }
        if (union[i] > 0) {
          last = i
          lastVal = val
        }
      })
      this.conflicts = this.alternativesBetweenConflicts(conflicts)
      this.routeGroups.forEach(group => { group.tagConflictVector(this.conflicts) })
      return this.conflicts
    } else {
      // NO CONFLICTS
      if (this.routeGroups.length === 1) {
        this.currentRoute = []
        this.routeGroups[0].routes[0].r.forEach(s => {
          this.currentRoute.push(s.id)
        })
      }
      const gs: { [id: number]: number[] } = {}
      gs[this.routeGroups[0].id] = []
      this.indexRouteGroups[this.routeGroups[0].id] = this.routeGroups[0]
      return [{
        conflictID: 0,
        origin: this.currentRoute[0],
        dest: this.currentRoute[this.currentRoute.length - 1],
        RouteGroupAlternatives: gs
      }]
    }
  }

  public alternativesBetweenConflicts(conflicts: Conflict[]) {
    if (conflicts.length > 0) {
      conflicts.forEach(conflict => {
        _.each(this.routeGroups, group => {
          conflict.RouteGroupAlternatives[group.id] = group.conflictStations([this.stationIndex[conflict.origin], this.stationIndex[conflict.dest]])
        })
      })
      return conflicts
    } else {
      return []
    }
  }

  public async computeCurrentMatrix() {
    const matrix : {[id: number]: {[id: number]: number}} = {}
    const {data: { fMatrix }} = await gqlQuery<{data: {fMatrix: number[][]}}>('fMatrix', F_MATRIX, {s: this.currentRoute})
    for (let i = 0; i < this.currentRoute.length; i++) {
      const s1 = this.currentRoute[i]
      matrix[s1] = {}
      for (let j = 0; j < this.currentRoute.length; j++) {
        const s2 = this.currentRoute[j]
        matrix[s1][s2] = fMatrix[i][j]
      }
    }
    return matrix
  }

  public async computeCurrentRouteTrips() {
    const {data: { checkinTrips } } = await gqlQuery<{data: {checkinTrips: Trip[][]}}>('checkinTrips', CHECKIN_TRIPS_FOR_STATION, {s: this.currentRoute})
    const {data: { checkoutTrips } } = await gqlQuery<{data: {checkoutTrips: Trip[][]}}>('checkoutTrips', CHECKOUT_TRIPS_FOR_STATION, {s: this.currentRoute})
    const checkin : {[id: number]: Trip[]} = {}
    const checkout : {[id: number]: Trip[]} = {}
    for (let i = 0; i < this.currentRoute.length; i++) {
      checkin[this.currentRoute[i]] = checkinTrips[i]
      checkout[this.currentRoute[i]] = checkoutTrips[i]
    }
    return {checkin: checkin, checkout: checkout}
  }
}
