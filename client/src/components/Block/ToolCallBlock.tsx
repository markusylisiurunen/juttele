import { BracesIcon, CopyIcon } from "lucide-react";
import React from "react";
import { ToolCallBlock } from "../../blocks";
import { useBlock } from "../../hooks";
import { tryOr } from "../../utils";
import styles from "./ToolCallBlock.module.css";

function formatFunc(name: string, args: string): [string, string] {
  args = tryOr(() => JSON.stringify(JSON.parse(args)), args);
  if (args === "{}") args = "";
  return [name, args];
}

type ToolCallComponentProps = {
  block: ToolCallBlock;
};
const ToolCallComponent: React.FC<ToolCallComponentProps> = ({ block }) => {
  const { isActive } = useBlock();
  const [name, args] = formatFunc(block.name, block.args);
  const label = `${name}(${args})`;
  function onCopy() {
    navigator.clipboard.writeText(args);
  }
  return (
    <div className={styles.root} data-block="tool-call" data-active={isActive ? "" : undefined}>
      <div className={styles.block}>
        <BracesIcon size={15} />
        <span>{label}</span>
      </div>
      <button className={styles.copy} onClick={onCopy}>
        <CopyIcon size={15} />
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
