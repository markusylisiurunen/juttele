import React, { useEffect, useRef, useState } from "react";
import { ThinkingBlock } from "../../blocks";
import { useBlock } from "../../hooks";
import styles from "./ThinkingBlock.module.css";

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
  const [expanded, setExpanded] = useState(isActive);
  const seconds = useSeconds();
  useEffect(() => {
    if (isActive) return;
    setExpanded(false);
  }, [isActive]);
  function onExpandOrCollapse() {
    setExpanded(!expanded);
  }
  function onCopy() {
    navigator.clipboard.writeText(cot);
  }
  const label = isActive ? `thinking (${seconds}s)...` : "thinking done";
  return (
    <div className={styles.root} data-block="thinking">
      <div className={styles.header}>
        <span>{label}</span>
        <div className={styles.actions}>
          <button onClick={onExpandOrCollapse}>{expanded ? "collapse" : "expand"}</button>
          <button onClick={onCopy}>copy</button>
        </div>
      </div>
      <div className={styles.content} data-expanded={expanded ? "" : undefined}>
        {cot.split("\n").map((line, idx) => (
          <p key={idx}>{line}</p>
        ))}
      </div>
    </div>
  );
};
const MemoedThinkingComponent = React.memo(ThinkingComponent, (prev, next) => {
  if (prev.block.id !== next.block.id) return false;
  if (prev.block.hash !== next.block.hash) return false;
  return true;
});

export { MemoedThinkingComponent as Thinking };
