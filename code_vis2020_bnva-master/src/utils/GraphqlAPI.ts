// Rough implementation. Untested.
function timeout(ms, promise) {
  return new Promise(function(resolve, reject) {
    setTimeout(function() {
      reject(new Error("timeout"))
    }, ms)
    promise.then(resolve, reject)
  })
}

export async function gqlQuery<T>(operationName: string, query: string, variables: any): Promise<T> {
  // @ts-ignore
  return timeout(800, fetch('query', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json'
    },
    body: JSON.stringify({
      operationName: operationName,
      query: query,
      variables: variables
    })
  })).then(response => {
    // @ts-ignore
    if (!response.ok) {
      // throw new Error(response.statusText)
      return null
    }
    // @ts-ignore
    return response.json() as Promise<T>
  })
}
