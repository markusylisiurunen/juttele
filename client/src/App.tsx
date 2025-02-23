import "./styles/globals.css";

import { CheckIcon, CopyIcon, Rows3Icon, SquarePenIcon } from "lucide-react";
import React, { useEffect, useMemo, useRef, useState } from "react";
import { API, ConfigResponse, DataResponse } from "./api";
import { AnyBlock } from "./blocks";
import { ChatHistory, MessageBox } from "./components";
import { assertNever, atom, Atom, streamCompletion, useAtomWithSelector } from "./utils";

const BASE_URL = import.meta.env.VITE_API_BASE_URL;
const API_KEY = import.meta.env.VITE_API_KEY;

//---

type AppHeaderProps = {
  title: string;
  onChatsClick: () => void;
  onCopyChatClick: () => void;
  onNewChatClick: () => void;
};
const AppHeader: React.FC<AppHeaderProps> = ({
  title,
  onChatsClick,
  onCopyChatClick,
  onNewChatClick,
}) => {
  const [copied, setCopied] = useState(false);
  useEffect(() => {
    if (!copied) return;
    const timeout = setTimeout(() => setCopied(false), 500);
    return () => clearTimeout(timeout);
  }, [copied]);
  const BLUR_SEGMENTS = 8;
  return (
    <div className="app-header">
      <div className="blur">
        {Array.from({ length: BLUR_SEGMENTS }).map((_, i) => {
          const MIN_BLUR = 0;
          const MAX_BLUR = 24;
          const blur = MIN_BLUR + (MAX_BLUR - MIN_BLUR) * (1 - i / (BLUR_SEGMENTS - 1));
          let gradCenter = (i / (BLUR_SEGMENTS - 1)) * 100;
          gradCenter *= 1 - 0.33;
          const d = 20;
          const grad1 = Math.max(0, gradCenter - 2 * d);
          const grad2 = Math.max(0, gradCenter - 1 * d);
          const grad3 = Math.min(100, gradCenter + 1 * d);
          const grad4 = Math.min(100, gradCenter + 2 * d);
          return (
            <div
              key={i}
              style={{
                backdropFilter: `blur(${blur}px)`,
                zIndex: 1,
                mask: `linear-gradient(${[
                  "to bottom",
                  `rgba(0,0,0,0) ${grad1}%`,
                  `rgba(0,0,0,1) ${grad2}%`,
                  `rgba(0,0,0,1) ${grad3}%`,
                  `rgba(0,0,0,0) ${grad4}%`,
                ].join(", ")})`,
              }}
            />
          );
        })}
        {/* make sure the blur doesn't bleed from the top */}
        <div
          style={{
            background: "linear-gradient(to bottom, var(--color-bg) 0%, transparent 20%)",
            zIndex: 99,
          }}
        />
      </div>
      <div>
        <button onClick={onChatsClick}>
          <Rows3Icon size={16} />
        </button>
        <span>{title}</span>
      </div>
      <div>
        <button
          onClick={() => {
            onCopyChatClick();
            setCopied(true);
          }}
        >
          {copied ? <CheckIcon size={16} /> : <CopyIcon size={16} />}
        </button>
        <button onClick={onNewChatClick}>
          <SquarePenIcon size={16} />
        </button>
      </div>
    </div>
  );
};

type AppProps = {
  api: API;
  configAtom: Atom<ConfigResponse>;
  dataAtom: Atom<DataResponse>;
  chatId: number;
  onGoToChats: () => void;
  onReset: () => void;
};
const App: React.FC<AppProps> = ({ configAtom, dataAtom, chatId, onGoToChats, onReset }) => {
  const scrollRef = useRef<HTMLDivElement>(null);
  const [title, setTitle] = useState("");
  const [model, setModel] = useState<{ modelId: string; personalityId: string }>();
  const [blocks, setBlocks] = useState([] as AnyBlock[]);
  useEffect(() => {
    const data = dataAtom.get();
    const chat = data.chats.find((chat) => chat.id === chatId);
    if (!chat) return;
    setTitle(chat.title);
    const blocks = [] as AnyBlock[];
    for (const item of chat.history) {
      if (item.kind === "message" && item.data.role === "user") {
        blocks.push({
          id: Date.now().toString() + "_" + blocks.length,
          type: "text",
          role: "user",
          content: item.data.content,
        });
      }
      if (item.kind === "message" && item.data.role === "assistant") {
        blocks.push({
          id: Date.now().toString() + "_" + blocks.length,
          type: "text",
          role: "assistant",
          content: item.data.content,
        });
        if (item.data.tool_calls && item.data.tool_calls.length > 0) {
          for (const t of item.data.tool_calls) {
            blocks.push({
              id: Date.now().toString() + "_" + blocks.length,
              type: "tool_call",
              name: t.function.name,
              args: t.function.arguments,
            });
          }
        }
      }
    }
    setBlocks(blocks);
  }, [chatId]);
  function onMessage(content: string) {
    const _model = model;
    if (!_model) return;
    const userId = Date.now().toString();
    let botId = userId + "_assistant";
    setBlocks((h) => [...h, { id: userId, type: "text", role: "user", content }]);
    setBlocks((h) => [...h, { id: botId, type: "text", role: "assistant", content: "" }]);
    requestAnimationFrame(() => scrollRef.current?.scrollBy({ top: 1000, behavior: "smooth" }));
    void Promise.resolve().then(async () => {
      try {
        await streamCompletion(
          BASE_URL,
          API_KEY,
          chatId,
          _model.modelId,
          _model.personalityId,
          content,
          (thinkingDelta) => {
            setBlocks((blocks) => {
              const last = blocks.at(-1);
              if (last?.type !== "thinking") {
                const id = Date.now().toString() + "_thinking";
                blocks = [...blocks, { id: id, type: "thinking", content: "" }];
              }
              return blocks.map((i, idx) => {
                if (idx !== blocks.length - 1 || i.type !== "thinking") return i;
                return { ...i, content: i.content + thinkingDelta };
              });
            });
          },
          (content) => {
            setBlocks((blocks) => {
              const last = blocks.at(-1);
              if (last?.type !== "text") {
                botId = Date.now().toString() + "_assistant";
                blocks = [...blocks, { id: botId, type: "text", role: "assistant", content: "" }];
              }
              return blocks.map((i) => {
                if (i.id === botId && i.type === "text") {
                  return { ...i, content: content };
                }
                return i;
              });
            });
          },
          (tool, args) => {
            setBlocks((blocks) => {
              const last = blocks.at(-1);
              if (last?.type === "tool_call") {
                return [...blocks.slice(0, -1), { ...last, name: tool, args: args }];
              }
              const id = Date.now().toString() + "_tool";
              return [...blocks, { id: id, type: "tool_call", name: tool, args: args }];
            });
          },
          (error) => {
            setBlocks((history) =>
              history.map((i) => {
                if (i.id === botId && i.type === "text") {
                  return { ...i, content: `Error: ${error}` };
                }
                return i;
              })
            );
          }
        );
      } catch (error) {
        setBlocks((history) =>
          history.map((i) => {
            if (i.id === botId && i.type === "text") {
              return { ...i, content: `Error: ${error}` };
            }
            return i;
          })
        );
      }
    });
  }
  function onCopyChatClick() {
    const segments = [] as string[];
    for (const block of blocks) {
      if (block.type === "text") {
        let text = "";
        text += `<!-- ${block.role.toUpperCase()} -->\n`;
        text += block.content.trim();
        segments.push(text);
      }
    }
    navigator.clipboard.writeText(segments.join("\n\n"));
  }
  function onControlModelChange(modelId: string, personalityId: string) {
    setModel({ modelId, personalityId });
  }
  return (
    <>
      <AppHeader
        title={title}
        onChatsClick={onGoToChats}
        onCopyChatClick={onCopyChatClick}
        onNewChatClick={onReset}
      />
      <div className="app-container">
        <ChatHistory scrollRef={scrollRef} blocks={blocks} />
        <MessageBox
          configAtom={configAtom}
          onMessage={onMessage}
          onControlModelChange={onControlModelChange}
        />
      </div>
    </>
  );
};

//---

type ChatListProps = {
  dataAtom: Atom<DataResponse>;
  onGoToApp: (chatId?: number) => void;
};
const ChatList: React.FC<ChatListProps> = ({ dataAtom, onGoToApp }) => {
  const chats = useAtomWithSelector(dataAtom, (data) => {
    const sortedChats = data.chats.toSorted((a, b) => {
      return new Date(b.created_at).getTime() - new Date(a.created_at).getTime();
    });
    return sortedChats.map((chat) => ({ id: chat.id, title: chat.title }));
  });
  return (
    <div className="chat-list">
      {chats.map((chat) => (
        <button onClick={() => onGoToApp(chat.id)}>{chat.title}</button>
      ))}
    </div>
  );
};

//---

const AppWrapper: React.FC = () => {
  const api = useMemo(() => new API(BASE_URL, API_KEY), []);
  const [configAtom, setConfigAtom] = useState<Atom<ConfigResponse>>();
  const [dataAtom, setDataAtom] = useState<Atom<DataResponse>>();
  const [chatId, setChatId] = useState<number>();
  const [route, setRoute] = useState<"app" | "chatList">("app");
  function navigateTo(target: typeof route) {
    setRoute(target);
    void refresh();
  }
  async function refresh() {
    const [config, data] = await Promise.all([api.getConfig(), api.getData()]);
    if (configAtom) configAtom.set(config);
    if (dataAtom) dataAtom.set(data);
  }
  const _init = useRef(false);
  async function init() {
    if (_init.current) return;
    _init.current = true;
    try {
      const [chat] = await Promise.all([
        api.rpc("create_chat", {
          title: `Chat ${new Date().toLocaleString()}`,
        }) as Promise<{ chat_id: number }>,
      ]);
      const [config, data] = await Promise.all([api.getConfig(), api.getData()]);
      setConfigAtom(atom(config));
      setDataAtom(atom(data));
      setChatId(chat.chat_id);
    } finally {
      _init.current = false;
    }
  }
  if (!configAtom || !dataAtom || !chatId) {
    void init();
    return null;
  }
  switch (route) {
    case "app":
      return (
        <App
          api={api}
          configAtom={configAtom}
          dataAtom={dataAtom}
          chatId={chatId}
          onGoToChats={() => navigateTo("chatList")}
          onReset={() => {
            setConfigAtom(undefined);
            setDataAtom(undefined);
            setChatId(undefined);
            setRoute("app");
          }}
        />
      );
    case "chatList":
      return (
        <ChatList
          dataAtom={dataAtom}
          onGoToApp={(chatId) => {
            if (chatId) setChatId(chatId);
            navigateTo("app");
          }}
        />
      );
    default:
      assertNever(route);
  }
};

export default AppWrapper;
