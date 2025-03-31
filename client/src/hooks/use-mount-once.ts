import { useEffect, useRef } from "react";

function useMountOnce(effect: () => void) {
  const init = useRef(false);
  useEffect(() => {
    if (init.current) return;
    init.current = true;
    effect();
  }, []);
}

export { useMountOnce };
