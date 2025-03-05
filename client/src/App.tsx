import "./styles/globals.css";

import React, { useEffect, useMemo, useRef, useState } from "react";
import { API, ConfigResponse, DataResponse } from "./api";
import { AnyBlock } from "./blocks";
import { ChatHistory, Header, MessageBox } from "./components";
import { makeListFilesTool, makeReadFileTool, makeWriteFileTool } from "./tools";
import { assertNever, atom, Atom, streamCompletion, useAtomWithSelector } from "./utils";

const BASE_URL = import.meta.env.VITE_API_BASE_URL;
const API_KEY = import.meta.env.VITE_API_KEY;
const FS_BASE_DIR = import.meta.env.VITE_FS_BASE_DIR;

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
  const [model, setModel] = useState<{ modelId: string; personalityId: string }>();
  const [blocks, setBlocks] = useState([] as AnyBlock[]);
  const [streaming, setStreaming] = useState(false);
  useEffect(() => {
    const data = dataAtom.get();
    const chat = data.chats.find((chat) => chat.id === chatId);
    if (!chat) return;
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
        setStreaming(true);
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
        const message = error instanceof Error ? error.message : "Something went wrong.";
        setBlocks((blocks) => [
          ...blocks,
          {
            id: Date.now().toString(),
            type: "text",
            role: "assistant",
            content: `Error: ${message}`,
          },
        ]);
      } finally {
        setStreaming(false);
      }
    });
  }
  function onRenameChatClick() {
    void Promise.resolve().then(async () => {
      if (!model) return;
      await api.rpc("rename_chat", { id: chatId, model_id: model.modelId });
      const data = await api.getData();
      dataAtom.set(data);
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
    <div className="wrapper">
      <Header
        dataAtom={dataAtom}
        chatId={chatId}
        onCopyChat={onCopyChatClick}
        onNewChat={onReset}
        onRenameChat={onRenameChatClick}
        onShowChats={onGoToChats}
      />
      <div className="content">
        <ChatHistory blocks={blocks} scrollRef={scrollRef} streaming={streaming} />
        <MessageBox
          configAtom={configAtom}
          streaming={streaming}
          onMessage={(content, opts) => onMessage(content, { tools: opts?.tools ?? false })}
          onControlModelChange={onControlModelChange}
        />
      </div>
    </div>
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
  const devModeIndicator = BASE_URL.includes("aa") ? (
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
