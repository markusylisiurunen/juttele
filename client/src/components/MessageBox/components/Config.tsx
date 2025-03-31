import { load, Store } from "@tauri-apps/plugin-store";
import React, { useRef } from "react";
import { useApp, useMount } from "../../../hooks";
import { useAtomWithSelector } from "../../../utils";
import styles from "./Config.module.css";

type ConfigProps = Record<string, never>;
const Config: React.FC<ConfigProps> = () => {
  const store = useRef<Store>();
  const app = useApp();
  const baseFileSystemPath = useAtomWithSelector(app.settings, (state) => state.baseFileSystemPath);
  useMount(() => {
    let canceled = false;
    void load("settings.json").then((_store) => {
      if (canceled) return;
      store.current = _store;
    });
    return () => {
      canceled = true;
    };
  });
  function setBaseFileSystemPath(path: string) {
    if (store.current) store.current.set("baseFileSystemPath", path);
    app.settings.set((state) => ({ ...state, baseFileSystemPath: path }));
  }
  return (
    <div className={styles.root}>
      <h2>Config</h2>
      <hr />
      <div className={styles.fs}>
        <p>Set the base path for file system related tools.</p>
        <input
          placeholder="Code/juttele"
          value={baseFileSystemPath ?? ""}
          onChange={(event) => setBaseFileSystemPath(event.target.value)}
        />
      </div>
    </div>
  );
};

export { Config };
