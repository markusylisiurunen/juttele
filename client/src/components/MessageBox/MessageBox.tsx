import { listen } from "@tauri-apps/api/event";
import React, { useEffect, useRef, useState } from "react";
import styles from "./MessageBox.module.css";

type MessageBoxProps = {
  onMessage: (message: string) => void;
  onControlReset: () => void;
  onControlModelChange: (modelId: string, personalityId: string) => void;
};
const MessageBox: React.FC<MessageBoxProps> = ({
  onMessage,
  onControlReset,
  onControlModelChange,
}) => {
  type Model = { id: string; name: string; personalities: Personality[] };
  type Personality = { id: string; name: string };
  const [models, setModels] = useState<Model[]>([]);
  const [personalities, setPersonalities] = useState<Personality[]>([]);
  const [selectedModel, setSelectedModel] = useState<Model>();
  const [selectedPersonality, setSelectedPersonality] = useState<Personality>();
  useEffect(() => {
    if (!selectedModel || !selectedPersonality) return;
    onControlModelChange(selectedModel.id, selectedPersonality.id);
  }, [selectedModel, selectedPersonality]);
  useEffect(() => {
    void Promise.resolve().then(async () => {
      const resp = await fetch(`${import.meta.env.VITE_API_BASE_URL}/models`, {
        headers: { Authorization: `Bearer ${import.meta.env.VITE_API_KEY}` },
      });
      const { models } = (await resp.json()) as {
        models: {
          id: string;
          name: string;
          personalities: {
            id: string;
            name: string;
          }[];
        }[];
      };
      setModels(models);
      const _selectedModel = models[0];
      setSelectedModel(_selectedModel);
      setPersonalities(_selectedModel.personalities);
      const _selectedPersonality = _selectedModel.personalities[0];
      setSelectedPersonality(_selectedPersonality);
    });
  }, []);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  useEffect(() => {
    const unlistenPromise = listen("tauri://focus", () => {
      textareaRef.current?.focus();
    });
    return () => {
      unlistenPromise.then((unlisten) => unlisten());
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
  function onResetClick() {
    onControlReset();
  }
  function onModelChangeClick() {
    const nextModelIdx = models.findIndex((i) => i.id === selectedModel?.id);
    const nextModel = models[(nextModelIdx + 1) % models.length];
    if (!nextModel) {
      setSelectedModel(undefined);
      setSelectedPersonality(undefined);
      return;
    }
    setSelectedModel(nextModel);
    setPersonalities(nextModel.personalities);
    setSelectedPersonality(nextModel.personalities[0]);
  }
  function onPersonalityChangeClick() {
    const nextPersonalityIdx = personalities.findIndex((i) => i.id === selectedPersonality?.id);
    const nextPersonality = personalities[(nextPersonalityIdx + 1) % personalities.length];
    setSelectedPersonality(nextPersonality);
  }
  return (
    <div className={styles.root}>
      <div className={styles.main}>
        <textarea ref={textareaRef} rows={1} placeholder="Ask anything" onKeyDown={onKeyDown} />
      </div>
      <div className={styles.footer}>
        <button onClick={onResetClick}>Reset</button>
        <button onClick={onModelChangeClick}>{selectedModel?.name}</button>
        <button onClick={onPersonalityChangeClick}>{selectedPersonality?.name}</button>
      </div>
    </div>
  );
};

export { MessageBox };
