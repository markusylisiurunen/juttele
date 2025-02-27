import "./styles/globals.css";

import { CheckIcon, CopyIcon, RefreshCcwIcon, Rows3Icon, SquarePenIcon } from "lucide-react";
import React, { useEffect, useMemo, useRef, useState } from "react";
import { API, ConfigResponse, DataResponse } from "./api";
import { AnyBlock } from "./blocks";
import { ChatHistory, MessageBox } from "./components";
import { makeListFilesTool, makeReadFileTool, makeWriteFileTool } from "./tools";
import { assertNever, atom, Atom, streamCompletion, useAtomWithSelector } from "./utils";

const BASE_URL = import.meta.env.VITE_API_BASE_URL;
const API_KEY = import.meta.env.VITE_API_KEY;
const FS_BASE_DIR = import.meta.env.VITE_FS_BASE_DIR;

//---

type AppHeaderProps = {
  chatTitle: string;
  onChatsClick: () => void;
  onRenameChatClick: () => void;
  onCopyChatClick: () => void;
  onNewChatClick: () => void;
};
const AppHeader: React.FC<AppHeaderProps> = ({
  chatTitle,
  onChatsClick,
  onRenameChatClick,
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
            background: "linear-gradient(to bottom, var(--color-bg) 0%, transparent 33%)",
            zIndex: 99,
          }}
        />
      </div>
      <div style={{ overflow: "hidden" }}>
        <button onClick={onChatsClick}>
          <Rows3Icon size={16} />
        </button>
        <span
          style={{
            flexShrink: 1,
            maxWidth: "50vw",
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
          }}
        >
          {chatTitle}
        </span>
        <button className="rename" onClick={onRenameChatClick}>
          <RefreshCcwIcon size={16} />
        </button>
      </div>
      <div style={{ flexShrink: 0 }}>
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
const App: React.FC<AppProps> = ({ api, configAtom, dataAtom, chatId, onGoToChats, onReset }) => {
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
      if (item.kind === "reasoning") {
        blocks.push({
          id: Date.now().toString() + "_" + blocks.length,
          type: "thinking",
          content: item.data.content,
        });
      }
    }
    setBlocks(blocks);
  }, [chatId]);
  function onMessage(content: string, opts: { tools: boolean }) {
    void Promise.resolve().then(async () => {
      if (!model) return;
      // append the user message
      setBlocks((blocks) => [
        ...blocks,
        { id: Date.now().toString(), type: "text", role: "user", content },
      ]);
      requestAnimationFrame(() => {
        scrollRef.current?.scrollTo({ top: 1_000_000, behavior: "smooth" });
      });
      // stream the completion
      try {
        await streamCompletion(
          BASE_URL,
          API_KEY,
          chatId,
          model.modelId,
          model.personalityId,
          opts.tools,
          content,
          [
            makeListFilesTool(FS_BASE_DIR),
            makeReadFileTool(FS_BASE_DIR),
            makeWriteFileTool(FS_BASE_DIR),
          ],
          (msg) => {
            if (msg.method === "block") {
              setBlocks((blocks) => {
                const idx = blocks.findIndex((i) => i.id === msg.params.id);
                if (idx === -1) return [...blocks, msg.params];
                blocks[idx] = msg.params;
                return [...blocks];
              });
            }
          }
        );
      } catch (error) {
        console.error(error);
      }
    });
  }
  function onRenameChatClick() {
    void Promise.resolve().then(async () => {
      if (!model) return;
      await api.rpc("rename_chat", { id: chatId, model_id: model.modelId });
      const data = await api.getData();
      dataAtom.set(data);
      setTitle(data.chats.find((chat) => chat.id === chatId)?.title ?? "");
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
        chatTitle={title}
        onChatsClick={onGoToChats}
        onRenameChatClick={onRenameChatClick}
        onCopyChatClick={onCopyChatClick}
        onNewChatClick={onReset}
      />
      <div className="app-container">
        <ChatHistory scrollRef={scrollRef} blocks={blocks} />
        <MessageBox
          configAtom={configAtom}
          onMessage={(content, opts) => onMessage(content, { tools: opts?.tools ?? false })}
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
    return sortedChats.map((chat) => ({
      id: chat.id,
      title: chat.title,
      message: chat.history.find((i) => i.kind === "message")?.data.content ?? null,
    }));
  });
  return (
    <div className="chat-list">
      {chats.map((chat) => (
        <button onClick={() => onGoToApp(chat.id)}>
          <span>{chat.title}</span>
          <span>{chat.message ?? "No messages available"}</span>
        </button>
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
  const devModeIndicator = BASE_URL.includes("localhost") ? (
    <div
      style={{
        background: "yellow",
        height: "2px",
        left: "0px",
        position: "fixed",
        right: "0px",
        top: "0px",
        zIndex: 99999,
      }}
    />
  ) : null;
  switch (route) {
    case "app":
      return (
        <>
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
          {devModeIndicator}
        </>
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
