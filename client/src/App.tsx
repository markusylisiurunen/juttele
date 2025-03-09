import "./styles/globals.css";

import React, { useRef, useState } from "react";
import { DataResponse } from "./api";
import { AnyBlock } from "./blocks";
import { ChatHistory, Header, MessageBox } from "./components";
import { AppProvider } from "./contexts";
import { useApp, useDev, useMount } from "./hooks";
import { makeListFilesTool, makeReadFileTool, makeWriteFileTool } from "./tools";
import { assertNever, Atom, streamCompletion, useAtomWithSelector } from "./utils";

const BASE_URL = import.meta.env.VITE_API_BASE_URL;
const API_KEY = import.meta.env.VITE_API_KEY;
const FS_BASE_DIR = import.meta.env.VITE_FS_BASE_DIR;

function upsertBlock(dataAtom: Atom<DataResponse>, chatId: number, block: AnyBlock) {
  dataAtom.set((data) => {
    return {
      ...data,
      chats: data.chats.map((chat) => {
        if (chat.id !== chatId) return chat;
        const blocks = chat.blocks;
        const idx = blocks.findIndex((i) => i.id === block.id);
        if (idx === -1) return { ...chat, blocks: [...blocks, block] };
        return { ...chat, blocks: [...blocks.slice(0, idx), block, ...blocks.slice(idx + 1)] };
      }),
    };
  });
}

type AppProps = {
  chatId: number;
  onGoToChats: () => void;
  onReset: () => void;
};
const App: React.FC<AppProps> = ({ chatId, onGoToChats, onReset }) => {
  const app = useApp();
  const dev = useDev();
  const scrollRef = useRef<HTMLDivElement>(null);
  const [model, setModel] = useState<{ modelId: string; personalityId: string }>();
  const [streaming, setStreaming] = useState(false);
  const blocks = useAtomWithSelector(app.data, (data) => {
    const chat = data.chats.find((chat) => chat.id === chatId);
    if (!chat) return [];
    return chat.blocks;
  });
  function onMessage(content: string, opts: { tools: boolean }) {
    void Promise.resolve().then(async () => {
      if (!model) return;
      upsertBlock(app.data, chatId, {
        id: Date.now().toString(),
        ts: new Date().toISOString(),
        hash: "",
        type: "text",
        role: "user",
        content: content,
      });
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
              upsertBlock(app.data, chatId, msg.params);
            }
          }
        );
      } catch (error) {
        console.error(error);
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
      {dev ? <div className="dev" /> : null}
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
      return new Date(b.ts).getTime() - new Date(a.ts).getTime();
    });
    return sortedChats.map((chat) => ({
      id: chat.id,
      title: chat.title,
      message: chat.blocks.find((i) => i.type === "text")?.content ?? null,
    }));
  });
  return (
    <div className="chat-list">
      {chats.map((chat) => (
        <button key={chat.id} onClick={() => onGoToApp(chat.id)}>
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
