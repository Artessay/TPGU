export function linspace (minimum : number, maximum : number, count : number) {
  const step = (maximum - minimum) / count
  const result: [number, number][] = []
  for (let p = minimum, i = 0; i < count; i++) {
    result.push([
      p,
      p + step
    ])
    p += step
  }
  console.log(result)
  return result
}

export function randomString () {
  return '_' + Math.random().toString(36).slice(2)
}
