import React, { useEffect, useRef } from "react";
import Markdown from "react-markdown";
import rehypeKatex from "rehype-katex";
import remarkGfm from "remark-gfm";
import remarkMath from "remark-math";
import { codeToHtml } from "shiki";
import { TextBlock } from "../../blocks";
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
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const timeout = setTimeout(() => {
      if (!ref.current) return;
      const pres = ref.current.querySelectorAll("& > pre");
      for (const pre of Array.from(pres)) {
        if (pre.classList.contains("shiki")) continue;
        const code = pre.querySelector("& > code");
        if (!code) continue;
        const lang =
          code
            .getAttribute("class")
            ?.split(" ")
            .find((i) => i.startsWith("language-"))
            ?.slice(9) ?? "plaintext";
        codeToHtml(code.textContent || "", {
          lang: lang,
          theme: "github-dark-dimmed",
        }).then((html) => {
          const div = document.createElement("div");
          div.innerHTML = html;
          pre.replaceWith(div.firstChild!);
        });
      }
    }, 500);
    return () => clearTimeout(timeout);
  }, [block.content]);
  return (
    <div className={styles.root} data-block="text" data-role={block.role}>
      <div
        ref={ref}
        className={styles.content}
        style={{ opacity: block.role === "user" ? 0.5 : undefined }}
      >
        <Markdown
          remarkPlugins={[remarkGfm, remarkMath]}
          rehypePlugins={[rehypeKatex]}
          components={{
            table: ({ children }) => (
              <div>
                <table>{children}</table>
              </div>
            ),
          }}
        >
          {preprocessLaTeX(block.content)}
        </Markdown>
      </div>
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
