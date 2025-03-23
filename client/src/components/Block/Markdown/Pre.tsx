import React, { useEffect, useRef, useState } from "react";
import { codeToHtml } from "shiki";
import { useBlock, useMount } from "../../../hooks";

function language(el: HTMLPreElement) {
  let lang = "plaintext";
  const code = el.querySelector("code");
  if (code) {
    const className = code.getAttribute("class");
    if (className) {
      lang = className.split("-").at(-1) || "plaintext";
    }
  }
  return lang;
}

function code(el: HTMLPreElement) {
  const code = el.querySelector("code");
  if (!code) return "";
  return code.textContent ?? "";
}

async function highlight(el: HTMLPreElement) {
  const lang = language(el);
  const code = el.querySelector("code");
  if (!code) return;
  const html = await codeToHtml(code.textContent ?? "", {
    lang: lang,
    theme: "github-dark-dimmed",
  });
  const div = document.createElement("div");
  div.innerHTML = html;
  el.replaceWith(div.firstChild!);
}

type PreProps = React.PropsWithChildren<React.ComponentPropsWithoutRef<"pre">>;
const Pre: React.FC<PreProps> = ({ children }) => {
  const { isActive } = useBlock();
  const ref = useRef<HTMLPreElement>(null);
  const copieable = useRef("");
  useMount(() => {
    if (isActive || !ref.current) return;
    copieable.current = code(ref.current);
  });
  if (isActive && ref.current) {
    copieable.current = code(ref.current);
  }
  const [lang, setLang] = useState("plaintext");
  useEffect(() => {
    if (!ref.current) return;
    setLang(language(ref.current));
  }, [ref.current]);
  useEffect(() => {
    if (!ref.current || isActive) return;
    highlight(ref.current);
  }, [ref.current, isActive]);
  function onCopy() {
    navigator.clipboard.writeText(copieable.current.trim() + "\n");
  }
  return (
    <div data-el="pre">
      <div>
        <span>{lang}</span>
        <button onClick={onCopy}>copy</button>
      </div>
      <pre ref={ref}>{children}</pre>
    </div>
  );
};

export { Pre };
