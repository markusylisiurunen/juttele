import { micromark } from "micromark";
import React, { useEffect, useRef, useState } from "react";
import { codeToHtml } from "shiki";
import styles from "./ChatHistory.module.css";

type ThinkingProps = {
  thinking: string;
};
const _Thinking: React.FC<ThinkingProps> = ({ thinking }) => {
  const [open, setOpen] = useState(false);
  return (
    <div className={styles.thinking}>
      <div>
        <span style={{ opacity: open ? 1 : 0.5 }}>Thinking process</span>
        <button onClick={() => setOpen(!open)}>{open ? "Hide" : "Show"}</button>
      </div>
      {open ? <pre>{thinking}</pre> : null}
    </div>
  );
};
const Thinking = React.memo(_Thinking);

type MessageProps = {
  role: string;
  thinking?: string;
  content: string;
};
const _Message: React.FC<MessageProps> = ({ role, thinking, content }) => {
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
    }, 200);
    return () => clearTimeout(timeout);
  }, [thinking, content]);
  useEffect(() => {
    const container = ref.current;
    if (!container) return;
    const observer = new MutationObserver((mutations) => {
      for (const mutation of mutations) {
        for (const node of mutation.addedNodes) {
          if (!(node instanceof HTMLElement && node.tagName === "PRE")) continue;
          const code = node.querySelector("code");
          if (!code) continue;
          // add copy buttons (top and bottom)
          for (const pos of [{ top: 8 }]) {
            const btn = document.createElement("button");
            btn.textContent = "Copy";
            btn.style.top = pos.top ? `${pos.top}px` : "auto";
            btn.classList.add("copy");
            btn.addEventListener("click", () => {
              const text = code.textContent;
              if (!text) return;
              navigator.clipboard.writeText(text);
              btn.textContent = "Copied!";
              setTimeout(() => {
                btn.textContent = "Copy";
              }, 2000);
            });
            node.appendChild(btn);
          }
        }
      }
    });
    observer.observe(container, { childList: true, subtree: true });
    return () => observer.disconnect();
  }, [ref]);
  return (
    <>
      {thinking ? <Thinking thinking={thinking} /> : null}
      <div
        ref={ref}
        className={styles.message}
        style={{ opacity: role === "user" ? 0.67 : 1 }}
        dangerouslySetInnerHTML={{ __html: micromark(content) }}
      />
    </>
  );
};
const Message = React.memo(_Message);

type ChatHistoryProps = {
  history: {
    id: string;
    role: string;
    thinking?: string;
    content: string;
  }[];
};
const ChatHistory: React.FC<ChatHistoryProps> = ({ history }) => {
  const scrollViewRef = useRef<HTMLDivElement>(null);
  // useEffect(() => {
  //   if (!scrollViewRef.current) return;
  //   scrollViewRef.current.scrollBy({
  //     top: scrollViewRef.current.scrollHeight,
  //     behavior: "smooth",
  //   });
  // }, [history.map((i) => i.id + i.content.trim()).join("_")]);
  return (
    <div className={styles.root}>
      <div className={styles.history} ref={scrollViewRef}>
        {history.map((item) => (
          <Message key={item.id} role={item.role} thinking={item.thinking} content={item.content} />
        ))}
      </div>
    </div>
  );
};

export { ChatHistory };
