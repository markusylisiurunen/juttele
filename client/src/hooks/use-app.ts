import { useContext } from "react";
import { appContext } from "../contexts";

function useApp() {
  return useContext(appContext);
}

export { useApp };
