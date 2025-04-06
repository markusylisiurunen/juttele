import "./styles/globals.css";

import React, { useRef, useState } from "react";
import { DataResponse } from "./api";
import styles from "./App.module.css";
import { AnyBlock } from "./blocks";
import { ChatHistory, Header, MessageBox } from "./components";
import { AppProvider } from "./contexts";
import { useApp, useMountOnce } from "./hooks";
import {
  makeEditFileTool,
  makeGrepTool,
  makeListFilesTool,
  makeReadFileTool,
  makeWriteFileTool,
} from "./tools";
import { assertNever, Atom, streamCompletion, useAtomWithSelector } from "./utils";

const BASE_URL = import.meta.env.VITE_API_BASE_URL;
const API_KEY = import.meta.env.VITE_API_KEY;

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

//--------------------------------------------------------------------------------------------------

type AppProps = {
  chatId: number;
  onShowChats: () => void;
  onReset: () => void;
};
const App: React.FC<AppProps> = ({ chatId, onShowChats, onReset }) => {
  const app = useApp();
  const scrollRef = useRef<HTMLDivElement>(null);
  function onMessage(message: string) {
    const { modelId, personalityId, tools, think } = app.generation.get();
    void Promise.resolve().then(async () => {
      upsertBlock(app.data, chatId, {
        id: Date.now().toString(),
        ts: new Date().toISOString(),
        hash: "",
        type: "text",
        role: "user",
        content: message,
      });
      requestAnimationFrame(() => {
        scrollRef.current?.scrollTo({ top: 1_000_000, behavior: "smooth" });
      });
      // stream the completion
      try {
        const baseFileSystemPath = app.settings.get().baseFileSystemPath;
        app.generation.set((state) => ({ ...state, generating: true }));
        await streamCompletion(
          BASE_URL,
          API_KEY,
          chatId,
          modelId,
          personalityId,
          tools,
          think,
          message,
          [
            ...(baseFileSystemPath
              ? [
                  makeEditFileTool(baseFileSystemPath),
                  makeGrepTool(baseFileSystemPath),
                  makeListFilesTool(baseFileSystemPath),
                  makeReadFileTool(baseFileSystemPath),
                  makeWriteFileTool(baseFileSystemPath),
                ]
              : []),
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
        app.generation.set((state) => ({ ...state, generating: false }));
      }
    });
  }
  function onRenameChatClick() {
    void Promise.resolve().then(async () => {
      const { modelId } = app.generation.get();
      await app.api.rpc("rename_chat", { id: chatId, model_id: modelId });
      const data = await app.api.getData();
      app.data.set(data);
    });
  }
  return (
    <div className={styles.app}>
      <Header
        chatId={chatId}
        onNewChat={onReset}
        onRenameChat={onRenameChatClick}
        onShowChats={onShowChats}
      />
      <div className={styles.content}>
        <ChatHistory chatId={chatId} scrollRef={scrollRef} />
        <MessageBox
          onSend={(message) => onMessage(message)}
          onCancel={() => {} /* TODO: cancel the generation */}
        />
      </div>
    </div>
  );
};

//--------------------------------------------------------------------------------------------------

type ChatsProps = {
  onGoToApp: (chatId?: number) => void;
};
const Chats: React.FC<ChatsProps> = ({ onGoToApp }) => {
  const app = useApp();
  const chats = useAtomWithSelector(app.data, (data) => {
    return data.chats
      .toSorted((a, b) => new Date(b.ts).getTime() - new Date(a.ts).getTime())
      .map((chat) => ({ id: chat.id, title: chat.title }));
  });
  return (
    <div className={styles.chats}>
      <div className={styles.content}>
        {chats.map((chat) => (
          <button key={chat.id} onClick={() => onGoToApp(chat.id)}>
            <span>{chat.title}</span>
          </button>
        ))}
      </div>
    </div>
  );
};

//--------------------------------------------------------------------------------------------------

const AppWrapper: React.FC = () => {
  const app = useApp();
  const [chatId, setChatId] = useState<number>();
  const [route, setRoute] = useState<"app" | "chats">("app");
  async function init() {
    const [chat] = await Promise.all([
      app.api.rpc("create_chat", {
        title: `Chat ${new Date().toLocaleString()}`,
      }) as Promise<{ chat_id: number }>,
    ]);
    setChatId(chat.chat_id);
    const data = await app.api.getData();
    app.data.set(data);
  }
  useMountOnce(() => void init());
  if (!chatId) {
    return null;
  }
  function navigateTo(target: typeof route) {
    setRoute(target);
    void Promise.all([
      app.api.getConfig().then((config) => app.config.set(config)),
      app.api.getData().then((data) => app.data.set(data)),
    ]);
  }
  function onReset() {
    setChatId(undefined);
    void Promise.resolve()
      .then(() => init())
      .then(() =>
        Promise.all([
          app.api.getConfig().then((config) => app.config.set(config)),
          app.api.getData().then((data) => app.data.set(data)),
        ])
      );
  }
  function onShowChats() {
    navigateTo("chats");
  }
  function onGoToApp(chatId?: number) {
    if (chatId) setChatId(chatId);
    navigateTo("app");
  }
  switch (route) {
    case "app":
      return <App chatId={chatId} onShowChats={onShowChats} onReset={onReset} />;
    case "chats":
      return <Chats onGoToApp={onGoToApp} />;
    default:
      assertNever(route);
  }
};

//--------------------------------------------------------------------------------------------------

const AppWithProvider: React.FC = () => {
  return (
    <AppProvider>
      <AppWrapper />
    </AppProvider>
  );
};

export default AppWithProvider;
