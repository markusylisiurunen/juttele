import { BrainIcon } from "lucide-react";
import React from "react";
import { ThinkingBlock } from "../../blocks";
import styles from "./Thinking.module.css";

type ThinkingComponentProps = {
  active: boolean;
  block: ThinkingBlock;
};
const ThinkingComponent: React.FC<ThinkingComponentProps> = ({ active, block }) => {
  return (
    <div className={styles.root} data-block="thinking" data-active={active ? "" : undefined}>
      <div className={styles.block}>
        <BrainIcon size={16} />
        <span>{block.content}</span>
      </div>
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
