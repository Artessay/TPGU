// decimal to percentage
export const prec = (d: number) => `${d * 100}%`
export const prec2 = (d: number) => (d * 100).toFixed(2) + '%'

// decimal to pixel
export const px = (d: number) => `${d}px`

// format time components
export const tc = (t: number) => ('0' + Math.floor(t)).slice(-2)

// format time string from second
export const ts = (s: number) => `${tc(s / 3600)}:${tc(s / 60 % 60)}`

// format comma splitted number
export const csn = (d: number) => {
  if (d > 10000) {
    return (d / 1000).toFixed(2).replace(/\B(?=(\d{3})+(?!\d))/g, ',') + 'k'
  }
  return d.toFixed(2).replace(/\B(?=(\d{3})+(?!\d))/g, ',')
}

// K
// export const toK = (d: number) => ``
