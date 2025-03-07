import { createContext, useMemo, useState } from "react";
import { API, ConfigResponse, DataResponse } from "../api";
import { useMount } from "../hooks";
import { atom, Atom } from "../utils";

const BASE_URL = import.meta.env.VITE_API_BASE_URL;
const API_KEY = import.meta.env.VITE_API_KEY;

type AppContextValue = {
  api: API;
  config: Atom<ConfigResponse>;
  data: Atom<DataResponse>;
};
const appContext = createContext<AppContextValue>({} as AppContextValue);

type AppProviderProps = React.PropsWithChildren;
const AppProvider: React.FC<AppProviderProps> = ({ children }) => {
  const api = useMemo(() => new API(BASE_URL, API_KEY), []);
  const [configAtom, setConfigAtom] = useState<Atom<ConfigResponse>>();
  const [dataAtom, setDataAtom] = useState<Atom<DataResponse>>();
  async function init() {
    const [config, data] = await Promise.all([api.getConfig(), api.getData()]);
    setConfigAtom(atom(config));
    setDataAtom(atom(data));
  }
  useMount(() => {
    void init();
  });
  if (!configAtom || !dataAtom) {
    return null;
  }
  return (
    <appContext.Provider value={{ api: api, config: configAtom, data: dataAtom }}>
      {children}
    </appContext.Provider>
  );
};

export { appContext, AppProvider };
