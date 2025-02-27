import { useSyncExternalStoreWithSelector } from "use-sync-external-store/shim/with-selector";

interface Atom<T> {
  get(): T;
  set(nextValue: T | ((prevValue: T) => T)): void;
  subscribe(callback: (value: T) => void): () => void;
  reset(): void;
}

function atom<T>(initialValue: T): Atom<T> {
  const callbacks = new Set<(value: T) => void>();
  let currentValue: T = initialValue;
  return {
    get() {
      return currentValue;
    },
    set(nextValue) {
      if (typeof nextValue === "function")
        currentValue = (nextValue as (prevValue: T) => T)(currentValue);
      else currentValue = nextValue;
      callbacks.forEach((callback) => callback(currentValue));
    },
    subscribe(callback) {
      callbacks.add(callback);
      return () => {
        callbacks.delete(callback);
      };
    },
    reset() {
      currentValue = initialValue;
      callbacks.forEach((callback) => callback(currentValue));
    },
  };
}

function identity<T>(value: T): T {
  return value;
}

function useAtom<T>(
  atom: Atom<T>,
  isEqual: (prevValue: T, nextValue: T) => boolean = Object.is
): T {
  return useSyncExternalStoreWithSelector(atom.subscribe, atom.get, atom.get, identity, isEqual);
}

function useAtomWithSelector<T, S>(
  atom: Atom<T>,
  selector: (value: T) => S,
  isEqual: (prevSelection: S, nextSelection: S) => boolean = Object.is
): S {
  return useSyncExternalStoreWithSelector(atom.subscribe, atom.get, atom.get, selector, isEqual);
}

export { atom, useAtom, useAtomWithSelector, type Atom };
