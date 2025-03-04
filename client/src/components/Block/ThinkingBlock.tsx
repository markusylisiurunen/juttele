import { BrainIcon, CopyIcon } from "lucide-react";
import React, { useEffect, useRef, useState } from "react";
import { ThinkingBlock } from "../../blocks";
import { useBlock } from "../../hooks";
import styles from "./Thinking.module.css";

function useSeconds() {
  const { isActive } = useBlock();
  const start = useRef(Date.now());
  const [duration, setDuration] = useState(0);
  useEffect(() => {
    if (!isActive) return;
    let _duration = 0;
    const interval = setInterval(() => {
      const next = Date.now() - start.current;
      if (Math.floor(next / 1000) !== Math.floor(_duration / 1000)) {
        _duration = next;
        setDuration(next);
      }
    }, 100);
    return () => {
      clearInterval(interval);
    };
  }, [isActive]);
  return Math.floor(duration / 1000);
}

type ThinkingComponentProps = {
  block: ThinkingBlock;
};
const ThinkingComponent: React.FC<ThinkingComponentProps> = ({ block }) => {
  const cot = block.content.trim();
  const { isActive } = useBlock();
  const seconds = useSeconds();
  function onCopy() {
    navigator.clipboard.writeText(cot);
  }
  const label = isActive ? `Thinking (${seconds}s)...` : "Thinking done";
  return (
    <div className={styles.root} data-block="thinking" data-active={isActive ? "" : undefined}>
      <div className={styles.container}>
        <div className={styles.block}>
          <BrainIcon size={15} />
          <span>{label}</span>
        </div>
        <button className={styles.copy} onClick={onCopy}>
          <CopyIcon size={15} />
        </button>
      </div>
      {isActive && cot !== "" ? (
        <div className={styles.preview}>
          {cot.split("\n").map((line, i) => (
            <p key={i}>{line}</p>
          ))}
        </div>
      ) : null}
    </div>
  );
};
const MemoedThinkingComponent = React.memo(ThinkingComponent, (prev, next) => {
  if (prev.block.id !== next.block.id) return false;
  if (prev.block.content !== next.block.content) return false;
  return true;
});

export { MemoedThinkingComponent as Thinking };
