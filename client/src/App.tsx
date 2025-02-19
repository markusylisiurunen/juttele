import "./styles/globals.css";

import { Rows3Icon, SquarePenIcon } from "lucide-react";
import React, { useEffect, useMemo, useRef, useState } from "react";
import { API, ConfigResponse, DataResponse } from "./api";
import { ChatHistory, MessageBox } from "./components";
import { ChatHistoryItem } from "./entities";
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
  const [history, setHistory] = useState([] as ChatHistoryItem[]);
  useEffect(() => {
    const data = dataAtom.get();
    const chat = data.chats.find((chat) => chat.id === chatId);
    if (!chat) return;
    setTitle(chat.title);
    const history = [] as ChatHistoryItem[];
    for (const item of chat.history) {
      if (item.kind === "message" && item.data.role === "user") {
        history.push({
          id: Date.now().toString() + "_" + history.length,
          role: "user",
          content: item.data.content,
        });
      }
      if (item.kind === "message" && item.data.role === "assistant") {
        history.push({
          id: Date.now().toString() + "_" + history.length,
          role: "assistant",
          content: item.data.content,
        });
      }
    }
    setHistory(history);
  }, [chatId]);
  function onMessage(content: string) {
    const _model = model;
    if (!_model) return;
    const userId = Date.now().toString();
    const assistantId = userId + "_assistant";
    setHistory((history) => [...history, { id: userId, role: "user", content }]);
    setHistory((history) => [...history, { id: assistantId, role: "assistant", content: "" }]);
    void Promise.resolve().then(async () => {
      try {
        const _history = history.map(({ role, content }) => ({ role, content }));
        _history.push({ role: "user", content });
        await streamCompletion(
          BASE_URL,
          API_KEY,
          chatId,
          _model.modelId,
          _model.personalityId,
          content,
          (thinkingDelta) => {
            setHistory((history) =>
              history.map((i) => {
                if (i.id === assistantId) {
                  return { ...i, thinking: (i.thinking ?? "") + thinkingDelta };
                }
                return i;
              })
            );
          },
          (contentDelta) => {
            setHistory((history) =>
              history.map((i) => {
                if (i.id === assistantId) {
                  return { ...i, content: i.content + contentDelta };
                }
                return i;
              })
            );
          },
          (error) => {
            setHistory((history) =>
              history.map((item) => {
                if (item.id === assistantId) {
                  return { ...item, content: `Error: ${error}` };
                }
                return item;
              })
            );
          }
        );
      } catch (error) {
        setHistory((history) =>
          history.map((item) => {
            if (item.id === assistantId) {
              return { ...item, content: `Error: ${error}` };
            }
            return item;
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
        <ChatHistory history={history} />
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
