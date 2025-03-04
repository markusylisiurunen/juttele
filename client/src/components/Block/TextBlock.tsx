import { CopyIcon, RotateCwIcon } from "lucide-react";
import React from "react";
import Markdown from "react-markdown";
import rehypeKatex from "rehype-katex";
import remarkGfm from "remark-gfm";
import remarkMath from "remark-math";
import { TextBlock } from "../../blocks";
import { Pre } from "./Markdown/Pre";
import { Table } from "./Markdown/Table";
import styles from "./TextBlock.module.css";

function preprocess(content: string) {
  content = content.replace(/\\\[(.*?)\\\]/gs, (_, eq) => `$$${eq}$$`);
  content = content.replace(/\\\((.*?)\\\)/gs, (_, eq) => `$${eq}$`);
  return content;
}

type TextComponentProps = {
  block: TextBlock;
};
const TextComponent: React.FC<TextComponentProps> = ({ block }) => {
  function onCopy() {
    navigator.clipboard.writeText(block.content.trim() + "\n");
  }
  return (
    <div className={styles.root} data-block="text" data-role={block.role}>
      <div className={styles.content} style={{ opacity: block.role === "user" ? 0.5 : undefined }}>
        <Markdown
          remarkPlugins={[remarkGfm, remarkMath]}
          rehypePlugins={[rehypeKatex]}
          components={{ pre: Pre, table: Table }}
        >
          {preprocess(block.content)}
        </Markdown>
      </div>
      {block.role === "assistant" ? (
        <div className={styles.actions}>
          <button onClick={onCopy}>
            <CopyIcon size={13} />
          </button>
          <button>
            <RotateCwIcon size={13} />
          </button>
          <span>
            {new Date().toLocaleDateString()} {new Date().toLocaleTimeString()}
          </span>
        </div>
      ) : null}
    </div>
  );
};
const MemoedTextComponent = React.memo(TextComponent, (prev, next) => {
  if (prev.block.id !== next.block.id) return false;
  if (prev.block.role !== next.block.role) return false;
  if (prev.block.content !== next.block.content) return false;
  return true;
});

export { MemoedTextComponent as Text };
