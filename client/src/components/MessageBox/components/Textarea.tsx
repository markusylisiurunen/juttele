import { listen, TauriEvent } from "@tauri-apps/api/event";
import React, { useRef } from "react";
import { useMount } from "../../../hooks";
import styles from "./Textarea.module.css";

type TextareaProps = {
  value: string;
  onChange: (value: string) => void;
  onSend: () => void;
};
const Textarea: React.FC<TextareaProps> = ({ value, onChange, onSend }) => {
  const ref = useRef<HTMLTextAreaElement>(null);
  useMount(() => {
    const unlisten = listen(TauriEvent.WINDOW_FOCUS, () => {
      ref.current?.focus();
    });
    return () => {
      void Promise.resolve()
        .then(() => unlisten)
        .then((unlisten) => unlisten());
    };
  });
  function onKeyDown(event: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key === "Enter" && !event.shiftKey) {
      event.preventDefault();
      onSend();
    }
  }
  return (
    <div className={styles.root}>
      <textarea
        ref={ref}
        className={styles.textarea}
        placeholder="Ask anything"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={onKeyDown}
      />
    </div>
  );
};

export { Textarea };
