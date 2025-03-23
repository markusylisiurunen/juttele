import React, { useEffect, useState } from "react";
import { ToolBlock } from "../../blocks";
import { useBlock } from "../../hooks";
import { tryOr } from "../../utils";
import styles from "./ToolBlock.module.css";

function formatFunc(name: string, args: string): [string, string] {
  args = tryOr(() => JSON.stringify(JSON.parse(args)), args);
  if (args === "{}") args = "";
  return [name, args];
}

type ToolComponentProps = {
  block: ToolBlock;
};
const ToolComponent: React.FC<ToolComponentProps> = ({ block }) => {
  const { isActive } = useBlock();
  const [name, args] = formatFunc(block.name, block.args);
  const [expanded, setExpanded] = useState(isActive);
  useEffect(() => {
    if (isActive) return;
    setExpanded(false);
  }, [isActive]);
  function onExpandOrCollapse() {
    setExpanded(!expanded);
  }
  function onCopy() {
    navigator.clipboard.writeText(args);
  }
  const label = `${name}()`;
  return (
    <div className={styles.root} data-block="tool-call">
      <div className={styles.header}>
        <span>{label}</span>
        <div className={styles.actions}>
          <button onClick={onExpandOrCollapse}>{expanded ? "collapse" : "expand"}</button>
          <button onClick={onCopy}>copy</button>
        </div>
      </div>
      <div className={styles.content} data-expanded={expanded ? "" : undefined}>
        {Object.entries(tryOr(() => JSON.parse(block.args), {})).map(([key, value], idx) => {
          return (
            <div key={idx} className={styles.arg}>
              <span>{key}</span>
              <span>{`${value}`}</span>
            </div>
          );
        })}
      </div>
    </div>
  );
};
const MemoedToolComponent = React.memo(ToolComponent, (prev, next) => {
  if (prev.block.id !== next.block.id) return false;
  if (prev.block.name !== next.block.name) return false;
  if (prev.block.args !== next.block.args) return false;
  return true;
});

export { MemoedToolComponent as Tool };
