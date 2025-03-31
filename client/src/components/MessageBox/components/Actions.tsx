import { ArrowUp, ChevronDown, Lightbulb, Parentheses } from "lucide-react";
import React from "react";
import { useApp } from "../../../hooks";
import { useAtomWithSelector } from "../../../utils";
import { Button } from "../../Button/Button";
import { Select } from "../../Select/Select";
import styles from "./Actions.module.css";

type ModelPickerProps = {
  modelId: string;
  personalityId: string;
  onChange: (modelId: string, personalityId: string) => void;
};
const ModelPicker: React.FC<ModelPickerProps> = ({ modelId, personalityId, onChange }) => {
  const configAtom = useApp().config;
  const models = useAtomWithSelector(configAtom, (config) => config.models);
  const personalities = models.find((model) => model.id === modelId)?.personalities ?? [];
  if (!models.length || !personalities.length) {
    return null;
  }
  return (
    <div className={styles.modelPicker}>
      <Select
        value={modelId}
        options={models.map((model) => ({
          id: model.id,
          label: model.name.toLowerCase(),
        }))}
        iconRight={<ChevronDown size={14} strokeWidth={2.5} />}
        onChange={(modelId) => onChange(modelId, personalityId)}
      />
      <Select
        value={personalityId}
        options={personalities.map((personality) => ({
          id: personality.id,
          label: personality.name.toLowerCase(),
        }))}
        iconRight={<ChevronDown size={14} strokeWidth={2.5} />}
        onChange={(personalityId) => onChange(modelId, personalityId)}
      />
    </div>
  );
};

type GenerationConfigProps = {
  tools: boolean;
  think: boolean;
  onChange: (tools: boolean, think: boolean) => void;
};
const GenerationConfig: React.FC<GenerationConfigProps> = ({ tools, think, onChange }) => {
  return (
    <div className={styles.generationConfig}>
      <Button
        glowing={tools}
        label="Tools"
        iconLeft={<Parentheses size={13} strokeWidth={2.5} />}
        onClick={() => onChange(!tools, think)}
      />
      <Button
        glowing={think}
        label="Think"
        iconLeft={<Lightbulb size={13} strokeWidth={2.5} />}
        onClick={() => onChange(tools, !think)}
      />
    </div>
  );
};

type SendButtonProps = {
  onSend: () => void;
  onCancel: () => void;
};
const SendButton: React.FC<SendButtonProps> = ({ onSend, onCancel }) => {
  const generating = useAtomWithSelector(useApp().generation, (state) => state.generating);
  return (
    <button className={styles.sendButton} onClick={generating ? onCancel : onSend}>
      {generating ? (
        <div
          style={{
            background: "currentColor",
            borderRadius: "2px",
            height: "10px",
            width: "10px",
          }}
        />
      ) : (
        <ArrowUp size={15} strokeWidth={2.5} />
      )}
    </button>
  );
};

type ActionsProps = {
  modelId: string;
  personalityId: string;
  tools: boolean;
  think: boolean;
  onModelChange: (modelId: string, personalityId: string) => void;
  onConfigChange: (tools: boolean, think: boolean) => void;
  onSend: () => void;
  onCancel: () => void;
};
const Actions: React.FC<ActionsProps> = ({
  modelId,
  personalityId,
  tools,
  think,
  onModelChange,
  onConfigChange,
  onSend,
  onCancel,
}) => {
  return (
    <div className={styles.root}>
      <ModelPicker modelId={modelId} personalityId={personalityId} onChange={onModelChange} />
      <div className={styles.generationConfigAndSend}>
        <GenerationConfig tools={tools} think={think} onChange={onConfigChange} />
        <SendButton onSend={onSend} onCancel={onCancel} />
      </div>
    </div>
  );
};

export { Actions };
