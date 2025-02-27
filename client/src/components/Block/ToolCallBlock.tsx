import { BracesIcon, CheckIcon, CopyIcon } from "lucide-react";
import React, { useEffect, useState } from "react";
import { ToolCallBlock } from "../../blocks";
import styles from "./ToolCallBlock.module.css";

function tryOr<T>(fn: () => T, fallback: T): T {
  try {
    return fn();
  } catch {
    return fallback;
  }
}

type ToolCallComponentProps = {
  active: boolean;
  block: ToolCallBlock;
};
const ToolCallComponent: React.FC<ToolCallComponentProps> = ({ active, block }) => {
  const [copied, setCopied] = useState(false);
  useEffect(() => {
    if (!copied) return;
    const timeout = setTimeout(() => setCopied(false), 500);
    return () => clearTimeout(timeout);
  }, [copied]);
  function onCopy() {
    const text = tryOr(() => JSON.stringify(JSON.parse(block.args), null, 2), block.args);
    navigator.clipboard.writeText(text);
    setCopied(true);
  }
  return (
    <div className={styles.root} data-block="tool-call" data-active={active ? "" : undefined}>
      <div className={styles.block}>
        <BracesIcon size={16} />
        <span>
          {block.name}({tryOr(() => JSON.stringify(JSON.parse(block.args)), block.args)})
        </span>
      </div>
      <button className={styles.copy} disabled={copied} onClick={onCopy}>
        {copied ? <CheckIcon size={16} /> : <CopyIcon size={16} />}
      </button>
    </div>
  );
};
const MemoedToolCallComponent = React.memo(ToolCallComponent, (prev, next) => {
  if (prev.block.id !== next.block.id) return false;
  if (prev.active !== next.active) return false;
  if (prev.block.name !== next.block.name) return false;
  if (prev.block.args !== next.block.args) return false;
  return true;
});

export { MemoedToolCallComponent as ToolCall };
