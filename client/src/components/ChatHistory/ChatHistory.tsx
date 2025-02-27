import React from "react";
import { AnyBlock } from "../../blocks";
import { Block } from "../Block/Block";
import styles from "./ChatHistory.module.css";

type DividerProps = {
  prev: AnyBlock | null;
  next: AnyBlock | null;
};
const Divider: React.FC<DividerProps> = ({ prev, next }) => {
  const GAP_SM = 0.5;
  const GAP_MD = 1;
  function getHeight() {
    if (!prev || !next) return 0;
    if (
      (prev.type === "text" && prev.role === "user") ||
      (next.type === "text" && next.role === "user")
    ) {
      return 1.75 * GAP_MD;
    }
    if (prev.type === "thinking") {
      if (next.type === "thinking") return GAP_SM;
      return GAP_MD;
    }
    if (prev.type === "tool_call") {
      if (next.type === "tool_call") return GAP_SM;
      return GAP_MD;
    }
    if (prev.type === "text") {
      return GAP_MD;
    }
    return 0;
  }
  return <div style={{ height: `${getHeight()}em` }} />;
};

type ChatHistoryProps = {
  scrollRef: React.RefObject<HTMLDivElement>;
  blocks: AnyBlock[];
};
const ChatHistory: React.FC<ChatHistoryProps> = ({ scrollRef, blocks }) => {
  return (
    <div className={styles.root}>
      <div className={styles.history} ref={scrollRef}>
        {blocks
          .filter((i) => {
            if (i.type === "text" && i.content.trim() === "") return false;
            return true;
          })
          .flatMap((i, idx) => {
            const prev = idx === 0 ? null : blocks[idx - 1] ?? null;
            switch (i.type) {
              case "text":
                if (i.content.trim() === "") return [];
                return [
                  <Divider key={i.id + "_divider"} prev={prev} next={i} />,
                  <Block.Text key={i.id} block={i} />,
                ];
              case "thinking":
                return [
                  <Divider key={i.id + "_divider"} prev={prev} next={i} />,
                  <Block.Thinking key={i.id} active={idx === blocks.length - 1} block={i} />,
                ];
              case "tool_call":
                return [
                  <Divider key={i.id + "_divider"} prev={prev} next={i} />,
                  <Block.ToolCall key={i.id} active={idx === blocks.length - 1} block={i} />,
                ];
              default:
                return null;
            }
          })}
      </div>
    </div>
  );
};

export { ChatHistory };
