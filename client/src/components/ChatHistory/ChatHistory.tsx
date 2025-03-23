import React from "react";
import { AnyBlock } from "../../blocks";
import { BlockProvider } from "../../contexts";
import { useApp } from "../../hooks";
import { useAtomWithSelector } from "../../utils";
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
      return 1.5 * GAP_MD;
    }
    if (prev.type === "thinking") {
      if (next.type === "thinking") return GAP_SM;
      return GAP_MD;
    }
    if (prev.type === "tool") {
      if (next.type === "tool") return GAP_SM;
      return GAP_MD;
    }
    if (prev.type === "text") {
      return GAP_MD;
    }
    if (prev.type === "error" || next.type === "error") {
      return GAP_MD;
    }
    return 0;
  }
  return <div style={{ height: `${getHeight()}em` }} />;
};

type ChatHistoryProps = {
  chatId: number;
  scrollRef: React.RefObject<HTMLDivElement>;
};
const ChatHistory: React.FC<ChatHistoryProps> = ({ chatId, scrollRef }) => {
  const streaming = useAtomWithSelector(useApp().generation, (state) => state.generating);
  let blocks = useAtomWithSelector(useApp().data, (data) => {
    const chat = data.chats.find((chat) => chat.id === chatId);
    if (!chat) return [];
    return chat.blocks;
  });
  blocks = blocks.filter((i) => {
    if (i.type === "text" && i.content.trim() === "") return false;
    return true;
  });
  return (
    <div className={styles.root}>
      <div className={styles.history} ref={scrollRef}>
        <div className={styles.blocks}>
          {blocks.map((b, idx) => {
            const p = idx === 0 ? null : blocks[idx - 1] ?? null;
            const isActive = streaming && idx === blocks.length - 1;
            let child: React.ReactNode = null;
            switch (b.type) {
              case "thinking":
                child = <Block.Thinking block={b} />;
                break;
              case "text":
                child = <Block.Text block={b} />;
                break;
              case "tool":
                child = <Block.Tool block={b} />;
                break;
              case "error":
                child = <Block.Error block={b} />;
                break;
              default:
                return [];
            }
            return (
              <React.Fragment key={b.id}>
                <Divider prev={p} next={b} />
                <BlockProvider isActive={isActive}>{child}</BlockProvider>
              </React.Fragment>
            );
          })}
        </div>
      </div>
    </div>
  );
};

export { ChatHistory };
