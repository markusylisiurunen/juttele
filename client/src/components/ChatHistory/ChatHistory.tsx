import React from "react";
import { AnyBlock } from "../../blocks";
import { Block } from "../Block/Block";
import styles from "./ChatHistory.module.css";

type ChatHistoryProps = {
  scrollRef: React.RefObject<HTMLDivElement>;
  blocks: AnyBlock[];
};
const ChatHistory: React.FC<ChatHistoryProps> = ({ scrollRef, blocks }) => {
  return (
    <div className={styles.root}>
      <div className={styles.history} ref={scrollRef}>
        {blocks.map((i, idx) => {
          switch (i.type) {
            case "text":
              return <Block.Text key={i.id} block={i} />;
            case "thinking":
              return <Block.Thinking key={i.id} active={idx === blocks.length - 1} block={i} />;
            case "tool_call":
              return <Block.ToolCall key={i.id} active={idx === blocks.length - 1} block={i} />;
            default:
              return null;
          }
        })}
      </div>
    </div>
  );
};

export { ChatHistory };
