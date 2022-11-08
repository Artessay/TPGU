import gql from 'graphql-tag'

export const TRIPS_NUM_BETWEEN_STATIONS = `
query trips($s1: Int!, $s2: Int!) {
  tripsBetweenStations(s1: $s1, s2: $s2)
}
`

export const F_MATRIX = `
query fMatrix ($s: [Int!]!){
  fMatrix(s: $s)
}
`

export const CHECKIN_TRIPS_FOR_STATION = `
query checkinTrips($s: [Int!]!) {
  checkinTrips(s: $s) {
    from,
    to,
    month,
    day,
    weekday,
    hour,
  }
}
`

export const CHECKOUT_TRIPS_FOR_STATION = `
query checkoutTrips($s: [Int!]!) {
  checkoutTrips(s: $s) {
    from,
    to,
    month,
    day,
    weekday,
    hour,
  }
}
`

export const PAUSE_MCTS = `
mutation pause($username: String!) {
  pauseMCTS(username: $username) {
    done
  }
}`

export const CONTINUE_MCTS = `
mutation continue($username: String!) {
  continueMCTS(username: $username) {
    routes{
      r {
        lon
        lat
        id
      }
    }
  }
}`

export const ADD_STOP_RUNTIME = `
mutation addStop($username: String!, $stop: Int!) {
  addStopInSearch(stop: $stop, username: $username) {
    nodes {
      s {
        id
      }
    }
  }
}`

export const GET_CURRENT_ROUTES = `
query getCurrentPlanning($username: String!) {
  plannings(username: $username) {
    routes {
      r {
        id,
        lat,
        lon
      }
      criteria,
      coordinates
    }
  }
}`

export const STOP_MCTS = `
mutation stopMCTS($username: String!) {
  stopMCTS(username: $username) {
    done
  }
}`

// Mutation: get the station graph based on od and stops
export const BUILD_GRAPH = `mutation BuildGraph($origin: Int!, $dest: Int!, $stops: [Int!]!, $seed: Int!){
  createStationGraph(input: {origin: $origin, dest: $dest, stops: $stops, seed: $seed}) {
    randomSeed
    username
    graph {
      originNode {
        s{id}
      }
      destNode {
        s{id}
      }
       nodes {
          order
          s {
            id
            lat
            lon
          }
          prev {
            s {
              id
            }
          }
          next {
            s {
              id
            }
          }
        }
     }
  }
}
`;
