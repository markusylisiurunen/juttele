import { load } from "@tauri-apps/plugin-store";
import { createContext, useMemo, useState } from "react";
import { API, ConfigResponse, DataResponse } from "../api";
import { useMountOnce } from "../hooks";
import { atom, Atom } from "../utils";

const BASE_URL = import.meta.env.VITE_API_BASE_URL;
const API_KEY = import.meta.env.VITE_API_KEY;

type Settings = {
  baseFileSystemPath: string | null;
};

type GenerationConfig = {
  generating: boolean;
  modelId: string;
  personalityId: string;
  tools: boolean;
  think: boolean;
};

type AppContextValue = {
  api: API;
  settings: Atom<Settings>;
  generation: Atom<GenerationConfig>;
  config: Atom<ConfigResponse>;
  data: Atom<DataResponse>;
};
const appContext = createContext<AppContextValue>({} as AppContextValue);

type AppProviderProps = React.PropsWithChildren;
const AppProvider: React.FC<AppProviderProps> = ({ children }) => {
  const api = useMemo(() => new API(BASE_URL, API_KEY), []);
  const [settingsAtom, setSettingsAtom] = useState<Atom<Settings>>();
  const [generationAtom, setGenerationAtom] = useState<Atom<GenerationConfig>>();
  const [configAtom, setConfigAtom] = useState<Atom<ConfigResponse>>();
  const [dataAtom, setDataAtom] = useState<Atom<DataResponse>>();
  async function init() {
    const [config, data] = await Promise.all([api.getConfig(), api.getData()]);
    setConfigAtom(atom(config));
    setDataAtom(atom(data));
    const settingsStore = await load("settings.json");
    const baseFileSystemPath = await settingsStore.get<string>("baseFileSystemPath");
    setSettingsAtom(
      atom({
        baseFileSystemPath: baseFileSystemPath ?? null,
      })
    );
    const configStore = await load("generation-config.json");
    const modelId = await configStore.get<string>("modelId");
    const personalityId = await configStore.get<string>("personalityId");
    const tools = await configStore.get<boolean>("tools");
    const think = await configStore.get<boolean>("think");
    setGenerationAtom(
      atom({
        generating: false as boolean,
        modelId: modelId ?? config.models[0].id,
        personalityId: personalityId ?? config.models[0].personalities[0].id,
        tools: tools ?? false,
        think: think ?? false,
      })
    );
  }
  useMountOnce(() => {
    void init();
  });
  if (!settingsAtom || !generationAtom || !configAtom || !dataAtom) {
    return null;
  }
  return (
    <appContext.Provider
      value={{
        api: api,
        settings: settingsAtom,
        generation: generationAtom,
        config: configAtom,
        data: dataAtom,
      }}
    >
      {children}
    </appContext.Provider>
  );
};

export { appContext, AppProvider };
