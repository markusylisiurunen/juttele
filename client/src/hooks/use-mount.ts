import { useEffect, useRef } from "react";

function useMount(effect: React.EffectCallback) {
  const mounted = useRef(false);
  useEffect(() => {
    if (mounted.current) return;
    mounted.current = true;
    return effect();
  }, []);
}

export { useMount };
