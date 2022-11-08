export interface Station {
  id: number
  lon: number
  lat: number
}

export interface StationNode {
  s: Station
  order: number
  prev: [{ s: {id: number} }]
  next: [{ s: {id: number} }]
}

export interface StationGraph {
  nodes: StationNode[]
  originNode: StationNode
  destNode: StationNode
}

export interface MonterCarloTreeNode {
  sNodes: StationNode[]
}

export interface MonterCarloTree {
  nodes: MonterCarloTreeNode[]
}

export interface SRoute {
  id: number
  r: Station[]
  // HARD-CODE: [ service time, passenger flow, directness]
  criteria: number[]
  weight: number[]
  coordinates: number[][]
  serviceCost: number
  constructCost: number
}

export interface RouteList {
  indexedRoutes: {[id: number]: SRoute}
  routes: SRoute[]
}

export interface Trip {
  from: number
  to: number
  day: number
  hour: number
  month: number
  weekday: number
}
