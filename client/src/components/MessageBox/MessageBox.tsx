import { listen } from "@tauri-apps/api/event";
import { load, Store } from "@tauri-apps/plugin-store";
import React, { useEffect, useRef, useState } from "react";
import { ConfigResponse } from "../../api";
import { useApp, useMount } from "../../hooks";
import { useAtomWithSelector } from "../../utils";
import { Button } from "../Button/Button";
import styles from "./MessageBox.module.css";

type MessageBoxProps = {
  store: Store;
  defaultModel?: string;
  defaultPersonality?: string;
  streaming?: boolean;
  onMessage: (message: string, opts?: { tools?: boolean }) => void;
  onControlModelChange: (modelId: string, personalityId: string) => void;
};
const MessageBox: React.FC<MessageBoxProps> = ({
  store,
  defaultModel,
  defaultPersonality,
  streaming,
  onMessage,
  onControlModelChange,
}) => {
  const app = useApp();
  type Model = ConfigResponse["models"][number];
  type Personality = Model["personalities"][number];
  const [model, setModel] = useState<Model>();
  const [personality, setPersonality] = useState<Personality>();
  const [tools, setTools] = useState(false);
  const options = useAtomWithSelector(app.config, (config) => config.models);
  useEffect(() => {
    if (!model || !personality) return;
    onControlModelChange(model.id, personality.id);
    void Promise.resolve().then(async () => {
      await store.set("defaultModel", model.id);
      await store.set("defaultPersonality", personality.id);
    });
  }, [model, personality]);
  useMount(() => {
    if (model && personality) return;
    if (defaultModel && defaultPersonality) {
      const _model = options.find((i) => i.id === defaultModel);
      if (_model) {
        const _personality = _model.personalities.find((i) => i.id === defaultPersonality);
        if (_personality) {
          setModel(_model);
          setPersonality(_personality);
          return;
        }
      }
    }
    const _selectedModel = options[0];
    const _selectedPersonality = _selectedModel.personalities[0];
    setModel(_selectedModel);
    setPersonality(_selectedPersonality);
  });
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  useMount(() => {
    const unlistenPromise = listen("tauri://focus", () => {
      textareaRef.current?.focus();
    });
    return () => {
      void unlistenPromise.then((unlisten) => unlisten());
    };
  });
  function onKeyDown(event: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key === "Enter" && !event.shiftKey) {
      event.preventDefault();
      const target = event.target as HTMLTextAreaElement;
      onMessage(target.value, { tools: tools });
      target.value = "";
    }
  }
  function onModelChangeClick() {
    const nextModelIdx = options.findIndex((i) => i.id === model?.id);
    const nextModel = options[(nextModelIdx + 1) % options.length];
    if (!nextModel) {
      setModel(undefined);
      setPersonality(undefined);
      return;
    }
    setModel(nextModel);
    setPersonality(nextModel.personalities[0]);
  }
  function onPersonalityChangeClick() {
    const personalities = model?.personalities || [];
    const nextPersonalityIdx = personalities.findIndex((i) => i.id === personality?.id);
    const nextPersonality = personalities[(nextPersonalityIdx + 1) % personalities.length];
    setPersonality(nextPersonality);
  }
  return (
    <div className={styles.root}>
      <div className={styles.main}>
        <textarea ref={textareaRef} rows={1} placeholder="Ask anything" onKeyDown={onKeyDown} />
      </div>
      <div className={styles.footer}>
        <div>
          <Button glowing={tools} icon="wrench" label="Tools" onClick={() => setTools(!tools)} />
          <Button label={model?.name ?? ""} onClick={onModelChangeClick} />
          <Button label={personality?.name ?? ""} onClick={onPersonalityChangeClick} />
        </div>
        <div>{streaming ? <span className={styles.responding}>Responding...</span> : null}</div>
      </div>
    </div>
  );
};

type _MessageBoxProps = Omit<MessageBoxProps, "store" | "defaultModel" | "defaultPersonality">;
const _MessageBox: React.FC<_MessageBoxProps> = (props) => {
  const [ready, setReady] = useState(false);
  const [extras, setExtras] = useState<{
    store: Store;
    defaultModel?: string;
    defaultPersonality?: string;
  }>();
  useMount(() => {
    const store = load("message-box.json");
    store
      .then(async (store) => {
        let defaultModel = await store.get<string>("defaultModel");
        let defaultPersonality = await store.get<string>("defaultPersonality");
        setExtras({ store, defaultModel, defaultPersonality });
      })
      .finally(() => setReady(true));
  });
  if (!ready || !extras) return null;
  return (
    <MessageBox
      {...props}
      store={extras.store}
      defaultModel={extras.defaultModel}
      defaultPersonality={extras.defaultPersonality}
    />
  );
};

export { _MessageBox as MessageBox };
