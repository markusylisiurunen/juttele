import { useEffect } from "react";

function useMount(effect: React.EffectCallback) {
  useEffect(() => effect(), []);
}

export { useMount };
