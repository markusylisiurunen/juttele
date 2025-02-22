import { listen } from "@tauri-apps/api/event";
import { WrenchIcon } from "lucide-react";
import React, { useEffect, useRef, useState } from "react";
import { ConfigResponse } from "../../api";
import { Atom, useAtomWithSelector } from "../../utils";
import styles from "./MessageBox.module.css";

type MessageBoxProps = {
  configAtom: Atom<ConfigResponse>;
  onMessage: (message: string) => void;
  onControlModelChange: (modelId: string, personalityId: string) => void;
};
const MessageBox: React.FC<MessageBoxProps> = ({ configAtom, onMessage, onControlModelChange }) => {
  type Model = ConfigResponse["models"][number];
  type Personality = Model["personalities"][number];
  const [model, setModel] = useState<Model>();
  const [personality, setPersonality] = useState<Personality>();
  const [tools, setTools] = useState(false);
  const options = useAtomWithSelector(configAtom, (config) => config.models);
  useEffect(() => {
    if (!model || !personality) return;
    onControlModelChange(model.id, personality.id);
  }, [model, personality]);
  useEffect(() => {
    if (model && personality) return;
    const _selectedModel = options[0];
    const _selectedPersonality = _selectedModel.personalities[0];
    setModel(_selectedModel);
    setPersonality(_selectedPersonality);
  }, []);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  useEffect(() => {
    const unlistenPromise = listen("tauri://focus", () => {
      textareaRef.current?.focus();
    });
    return () => {
      void unlistenPromise.then((unlisten) => unlisten());
    };
  }, []);
  function onKeyDown(event: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key === "Enter" && !event.shiftKey) {
      event.preventDefault();
      const target = event.target as HTMLTextAreaElement;
      onMessage(target.value);
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
          <button data-active={tools ? "" : undefined} onClick={() => setTools((t) => !t)}>
            <WrenchIcon size={14} />
            <span>Tools</span>
          </button>
          <button onClick={onModelChangeClick}>{model?.name}</button>
          <button onClick={onPersonalityChangeClick}>{personality?.name}</button>
        </div>
        <div></div>
      </div>
    </div>
  );
};

export { MessageBox };
