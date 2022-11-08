type ResolveFunc<T> = (value?: T | Promise<T> | undefined) => void
// eslint-disable-next-line
type RejectFunc = (reason?: any) => void
// eslint-disable-next-line
type ExecutorFunc<T> = (resolve: ResolveFunc<T>, reject: RejectFunc) => any

export class DeferredPromise<T> {
  public resolve!: ResolveFunc<T>
  public reject!: RejectFunc

  private _promise: Promise<T>

  constructor () {
    this._promise = new Promise((resolve, reject) => {
      this.resolve = resolve
      this.reject = reject
    })
  }

  public async get () {
    return this._promise
  }
}
