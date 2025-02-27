import { CheckIcon, CopyIcon, RotateCwIcon } from "lucide-react";
import React, { useEffect, useState } from "react";
import Markdown from "react-markdown";
import rehypeKatex from "rehype-katex";
import remarkGfm from "remark-gfm";
import remarkMath from "remark-math";
import { TextBlock } from "../../blocks";
import { Pre, Table } from "../Markdown/Markdown";
import styles from "./TextBlock.module.css";

const preprocessLaTeX = (content: string) => {
  content = content.replace(/\\\[(.*?)\\\]/gs, (_, eq) => `$$${eq}$$`);
  content = content.replace(/\\\((.*?)\\\)/gs, (_, eq) => `$${eq}$`);
  return content;
};

type TextComponentProps = {
  block: TextBlock;
};
const TextComponent: React.FC<TextComponentProps> = ({ block }) => {
  const [copied, setCopied] = useState(false);
  useEffect(() => {
    if (!copied) return;
    const timeout = setTimeout(() => setCopied(false), 500);
    return () => clearTimeout(timeout);
  }, [copied]);
  return (
    <div className={styles.root} data-block="text" data-role={block.role}>
      <div className={styles.content} style={{ opacity: block.role === "user" ? 0.5 : undefined }}>
        <Markdown
          remarkPlugins={[remarkGfm, remarkMath]}
          rehypePlugins={[rehypeKatex]}
          components={{ pre: Pre, table: Table }}
        >
          {preprocessLaTeX(block.content)}
        </Markdown>
      </div>
      {block.role === "assistant" ? (
        <div className={styles.actions}>
          <button
            onClick={() => {
              navigator.clipboard.writeText(block.content.trim() + "\n");
              setCopied(true);
            }}
          >
            {copied ? <CheckIcon size={13} /> : <CopyIcon size={13} />}
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
