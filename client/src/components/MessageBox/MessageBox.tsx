import { load, Store } from "@tauri-apps/plugin-store";
import React, { useState } from "react";
import { useApp, useMount } from "../../hooks";
import { useAtomWithSelector } from "../../utils";
import styles from "./MessageBox.module.css";
import { Actions } from "./components/Actions";
import { Textarea } from "./components/Textarea";

type MessageBoxProps = {
  store: Store;
  onSend: (message: string) => void;
  onCancel: () => void;
};
const MessageBox: React.FC<MessageBoxProps> = ({ store, onSend, onCancel }) => {
  const app = useApp();
  const [modelId, personalityId, tools, think] = useAtomWithSelector(app.generation, (state) => [
    state.modelId,
    state.personalityId,
    state.tools,
    state.think,
  ]);
  const [message, setMessage] = useState("");
  async function _onModelChange(newModelId: string, newPersonalityId: string) {
    const modelChanged = newModelId !== modelId;
    if (modelChanged) {
      const models = app.config.get().models;
      const model = models.find((model) => model.id === newModelId);
      if (!model) throw new Error(`model "${newModelId}" not found`);
      newPersonalityId = model.personalities[0].id;
    }
    await store.set("modelId", newModelId);
    await store.set("personalityId", newPersonalityId);
    app.generation.set((state) => ({
      ...state,
      modelId: newModelId,
      personalityId: newPersonalityId,
    }));
  }
  async function _onConfigChange(tools: boolean, think: boolean) {
    await store.set("tools", tools);
    await store.set("think", think);
    app.generation.set((state) => ({
      ...state,
      tools: tools,
      think: think,
    }));
  }
  async function _onSend() {
    onSend(message);
    setMessage("");
  }
  return (
    <div className={styles.root}>
      <div className={styles.container}>
        <Textarea value={message} onChange={setMessage} onSend={_onSend} />
        <Actions
          modelId={modelId}
          personalityId={personalityId}
          tools={tools}
          think={think}
          onModelChange={_onModelChange}
          onConfigChange={_onConfigChange}
          onSend={_onSend}
          onCancel={onCancel}
        />
      </div>
    </div>
  );
};
const MessageBoxWithStore: React.FC<Omit<MessageBoxProps, "store">> = (props) => {
  const [ready, setReady] = useState(false);
  const [store, setStore] = useState<Store>();
  useMount(() => {
    let canceled = false;
    load("generation-config.json").then((store) => {
      if (canceled) return;
      setStore(store);
      setReady(true);
    });
    return () => {
      canceled = true;
      setStore(undefined);
      setReady(false);
    };
  });
  if (!ready || !store) {
    return null;
  }
  return <MessageBox {...props} store={store} />;
};

export { MessageBoxWithStore as MessageBox };
