import { useContext } from "react";
import { blockContext } from "../contexts";

function useBlock() {
  return useContext(blockContext);
}

export { useBlock };
