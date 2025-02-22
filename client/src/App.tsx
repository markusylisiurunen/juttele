import "./styles/globals.css";

import { Rows3Icon, SquarePenIcon } from "lucide-react";
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
  onNewChatClick: () => void;
};
const AppHeader: React.FC<AppHeaderProps> = ({ title, onChatsClick, onNewChatClick }) => {
  return (
    <div className="app-header">
      <div>
        <button onClick={onChatsClick}>
          <Rows3Icon size={16} />
        </button>
        <span>{title}</span>
      </div>
      <div>
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
    void Promise.resolve().then(async () => {
      try {
        await streamCompletion(
          BASE_URL,
          API_KEY,
          chatId,
          _model.modelId,
          _model.personalityId,
          content,
          () => {
            // setBlocks((history) =>
            //   history.map((i) => {
            //     if (i.id === botId) {
            //       return { ...i, thinking: (i.thinking ?? "") + thinkingDelta };
            //     }
            //     return i;
            //   })
            // );
          },
          (contentDelta) => {
            setBlocks((history) =>
              history.map((i) => {
                if (i.id === botId && i.type === "text") {
                  return { ...i, content: i.content + contentDelta };
                }
                return i;
              })
            );
          },
          (tool, args) => {
            const id = Date.now().toString();
            let _args: Record<string, unknown> = {};
            try {
              _args = JSON.parse(args);
            } catch (error) {}
            setBlocks((h) => [...h, { id: id, type: "tool_call", name: tool, args: _args }]);
            botId = id + "_assistant";
            setBlocks((h) => [...h, { id: botId, type: "text", role: "assistant", content: "" }]);
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
  function onControlModelChange(modelId: string, personalityId: string) {
    setModel({ modelId, personalityId });
  }
  return (
    <>
      <AppHeader title={title} onChatsClick={onGoToChats} onNewChatClick={onReset} />
      <div className="app-container">
        <ChatHistory blocks={blocks} />
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
