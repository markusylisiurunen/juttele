export interface Resolvable<T> {
  promise: Promise<T>;
  readonly resolved: boolean;
  resolve(value: T): void;
  reject(reason: any): void;
}

export function resolvable<T>(): Resolvable<T> {
  let _resolved = false;
  let resolve: (value: T) => void;
  let reject: (reason: any) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return {
    promise,
    get resolved() {
      return _resolved;
    },
    resolve: (...args) => {
      if (!_resolved) {
        _resolved = true;
        resolve!(...args);
      }
    },
    reject: (...args) => {
      if (!_resolved) {
        _resolved = true;
        reject!(...args);
      }
    },
  };
}
