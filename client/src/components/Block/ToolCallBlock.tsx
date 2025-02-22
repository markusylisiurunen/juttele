import { BracesIcon, CheckIcon, CopyIcon } from "lucide-react";
import React, { useEffect, useState } from "react";
import { ToolCallBlock } from "../../blocks";
import styles from "./ToolCallBlock.module.css";

type ToolCallComponentProps = {
  block: ToolCallBlock;
};
const ToolCallComponent: React.FC<ToolCallComponentProps> = ({ block }) => {
  const [copied, setCopied] = useState(false);
  useEffect(() => {
    if (!copied) return;
    const timeout = setTimeout(() => setCopied(false), 500);
    return () => clearTimeout(timeout);
  }, [copied]);
  function onCopy() {
    navigator.clipboard.writeText(JSON.stringify(block.args, null, 2));
    setCopied(true);
  }
  return (
    <div className={styles.root} data-block="tool-call">
      <div className={styles.block}>
        <BracesIcon size={16} />
        <span>
          {block.name}({JSON.stringify(block.args)})
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
  if (prev.block.name !== next.block.name) return false;
  if (prev.block.args !== next.block.args) return false;
  return true;
});

export { MemoedToolCallComponent as ToolCall };
