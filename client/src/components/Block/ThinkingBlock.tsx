import { BrainIcon, CheckIcon, CopyIcon } from "lucide-react";
import React, { useEffect, useMemo, useState } from "react";
import { ThinkingBlock } from "../../blocks";
import styles from "./Thinking.module.css";

type ThinkingComponentProps = {
  active: boolean;
  block: ThinkingBlock;
};
const ThinkingComponent: React.FC<ThinkingComponentProps> = ({ active, block }) => {
  const beginAt = useMemo(() => Date.now(), []);
  const thoughtForSeconds = `${Math.round((Date.now() - beginAt) / 1000)} s`;
  const [copied, setCopied] = useState(false);
  useEffect(() => {
    if (!copied) return;
    const timeout = setTimeout(() => setCopied(false), 500);
    return () => clearTimeout(timeout);
  }, [copied]);
  function onCopy() {
    navigator.clipboard.writeText(block.content);
    setCopied(true);
  }
  return (
    <div className={styles.root} data-block="thinking" data-active={active ? "" : undefined}>
      <div className={styles.container}>
        <div className={styles.block}>
          <BrainIcon size={16} />
          <span>
            {active ? `Thinking for ${thoughtForSeconds}...` : `Thought for ${thoughtForSeconds}`}
          </span>
        </div>
        <button className={styles.copy} disabled={copied} onClick={onCopy}>
          {copied ? <CheckIcon size={16} /> : <CopyIcon size={16} />}
        </button>
      </div>
      {active ? (
        <div className={styles.preview}>
          {block.content.split("\n").map((line, i) => (
            <p key={i}>{line}</p>
          ))}
        </div>
      ) : null}
    </div>
  );
};
const MemoedThinkingComponent = React.memo(ThinkingComponent, (prev, next) => {
  if (prev.block.id !== next.block.id) return false;
  if (prev.active !== next.active) return false;
  if (prev.block.content !== next.block.content) return false;
  return true;
});

export { MemoedThinkingComponent as Thinking };
