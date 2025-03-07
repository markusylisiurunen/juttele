import "./styles/globals.css";

import React, { useEffect, useRef, useState } from "react";
import { AnyBlock } from "./blocks";
import { ChatHistory, Header, MessageBox } from "./components";
import { AppProvider } from "./contexts";
import { useApp, useMount } from "./hooks";
import { makeListFilesTool, makeReadFileTool, makeWriteFileTool } from "./tools";
import { assertNever, streamCompletion, useAtomWithSelector } from "./utils";

const BASE_URL = import.meta.env.VITE_API_BASE_URL;
const API_KEY = import.meta.env.VITE_API_KEY;
const FS_BASE_DIR = import.meta.env.VITE_FS_BASE_DIR;

type AppProps = {
  chatId: number;
  onGoToChats: () => void;
  onReset: () => void;
};
const App: React.FC<AppProps> = ({ chatId, onGoToChats, onReset }) => {
  const app = useApp();
  const scrollRef = useRef<HTMLDivElement>(null);
  const [model, setModel] = useState<{ modelId: string; personalityId: string }>();
  const [blocks, setBlocks] = useState([] as AnyBlock[]);
  const [streaming, setStreaming] = useState(false);
  useEffect(() => {
    const data = app.data.get();
    const chat = data.chats.find((chat) => chat.id === chatId);
    if (!chat) return;
    const blocks = [] as AnyBlock[];
    for (const item of chat.history) {
      if (item.kind === "message" && item.data.role === "user") {
        blocks.push({
          id: item.id,
          type: "text",
          role: "user",
          content: item.data.content,
        });
      }
      if (item.kind === "message" && item.data.role === "assistant") {
        blocks.push({
          id: item.id,
          type: "text",
          role: "assistant",
          content: item.data.content,
        });
        if (item.data.tool_calls && item.data.tool_calls.length > 0) {
          for (const t of item.data.tool_calls) {
            blocks.push({
              id: item.id,
              type: "tool_call",
              name: t.function.name,
              args: t.function.arguments,
            });
          }
        }
      }
      if (item.kind === "reasoning") {
        blocks.push({
          id: item.id,
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
      await app.api.rpc("rename_chat", { id: chatId, model_id: model.modelId });
      const data = await app.api.getData();
      app.data.set(data);
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
        chatId={chatId}
        onCopyChat={onCopyChatClick}
        onNewChat={onReset}
        onRenameChat={onRenameChatClick}
        onShowChats={onGoToChats}
      />
      <div className="content">
        <ChatHistory blocks={blocks} scrollRef={scrollRef} streaming={streaming} />
        <MessageBox
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
  onGoToApp: (chatId?: number) => void;
};
const ChatList: React.FC<ChatListProps> = ({ onGoToApp }) => {
  const app = useApp();
  const chats = useAtomWithSelector(app.data, (data) => {
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
  const app = useApp();
  const [chatId, setChatId] = useState<number>();
  const [route, setRoute] = useState<"app" | "chatList">("app");
  function navigateTo(target: typeof route) {
    setRoute(target);
    void refresh();
  }
  async function refresh() {
    // const [config, data] = await Promise.all([api.getConfig(), api.getData()]);
    // if (configAtom) configAtom.set(config);
    // if (dataAtom) dataAtom.set(data);
  }
  async function init() {
    const [chat] = await Promise.all([
      app.api.rpc("create_chat", {
        title: `Chat ${new Date().toLocaleString()}`,
      }) as Promise<{ chat_id: number }>,
    ]);
    setChatId(chat.chat_id);
  }
  useMount(() => {
    void init();
  });
  if (!chatId) {
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
            chatId={chatId}
            onGoToChats={() => navigateTo("chatList")}
            onReset={() => {
              // setConfigAtom(undefined);
              // setDataAtom(undefined);
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

export default () => {
  return (
    <AppProvider>
      <AppWrapper />
    </AppProvider>
  );
};
