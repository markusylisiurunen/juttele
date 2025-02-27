import { CheckIcon, CopyIcon } from "lucide-react";
import React, { useEffect, useRef, useState } from "react";
import { codeToHtml } from "shiki";

type PreProps = React.PropsWithChildren<unknown>;
const Pre: React.FC<PreProps> = ({ children }) => {
  const ref = useRef<HTMLDivElement>(null);
  // store the copy state
  const [copied, setCopied] = useState(false);
  const copieable = useRef<string>("");
  useEffect(() => {
    if (!ref.current) return;
    const code = ref.current.querySelector("pre > code");
    const text = code?.textContent;
    if (!text) return;
    copieable.current = text;
  });
  useEffect(() => {
    if (!copied) return;
    const timeout = setTimeout(() => setCopied(false), 500);
    return () => clearTimeout(timeout);
  }, [copied]);
  // extract the language from the first code block
  const [lang, setLang] = useState<string>("plaintext");
  useEffect(() => {
    if (!ref.current) return;
    const code = ref.current.querySelector("pre > code");
    if (!code) return;
    const className = code.getAttribute("class");
    if (!className) return;
    const lang = className.split("-").at(-1);
    if (!lang) return;
    setLang(lang);
  }, [ref.current]);
  // code highlighting
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
  });
  return (
    <div ref={ref}>
      <div>
        <button
          onClick={() => {
            navigator.clipboard.writeText(copieable.current.trim() + "\n");
            setCopied(true);
          }}
        >
          {copied ? <CheckIcon size={14} /> : <CopyIcon size={14} />}
        </button>
        <span>{lang}</span>
      </div>
      <pre>{children}</pre>
    </div>
  );
};

export { Pre };
