import "./styles/globals.css";

import React, { useEffect, useMemo, useRef, useState } from "react";
import { API, ConfigResponse, DataResponse } from "./api";
import { ChatHistory, MessageBox } from "./components";
import { ChatHistoryItem } from "./entities";
import { assertNever, atom, Atom, streamCompletion, useAtomWithSelector } from "./utils";

const BASE_URL = import.meta.env.VITE_API_BASE_URL;
const API_KEY = import.meta.env.VITE_API_KEY;

type AppProps = {
  api: API;
  configAtom: Atom<ConfigResponse>;
  dataAtom: Atom<DataResponse>;
  chatId: number;
  onGoToChats: () => void;
  onReset: () => void;
};
const App: React.FC<AppProps> = ({ configAtom, dataAtom, chatId, onGoToChats, onReset }) => {
  const [model, setModel] = useState<{ modelId: string; personalityId: string }>();
  const [history, setHistory] = useState([] as ChatHistoryItem[]);
  useEffect(() => {
    const data = dataAtom.get();
    const chat = data.chats.find((chat) => chat.id === chatId);
    if (!chat) return;
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
    <div className="container">
      <ChatHistory history={history} />
      <MessageBox
        configAtom={configAtom}
        onMessage={onMessage}
        onControlGoToChats={onGoToChats}
        onControlReset={onReset}
        onControlModelChange={onControlModelChange}
      />
    </div>
  );
};

type ChatProps = {
  api: API;
  configAtom: Atom<ConfigResponse>;
  dataAtom: Atom<DataResponse>;
  onGoToIndex: () => void;
  onGoToChat: (chatId: number) => void;
};
const Chats: React.FC<ChatProps> = ({ dataAtom, onGoToIndex, onGoToChat }) => {
  const chats = useAtomWithSelector(dataAtom, (data) =>
    data.chats
      .map((chat) => ({
        id: chat.id,
        createdAt: new Date(chat.created_at),
        title: chat.title,
      }))
      .toSorted((a, b) => {
        return b.createdAt.getTime() - a.createdAt.getTime();
      })
  );
  return (
    <div className="container">
      <div style={{ overflowY: "auto" }}>
        <button onClick={() => onGoToIndex()}>Back</button>
        {chats.map((chat) => (
          <div key={chat.id}>
            <button
              onClick={() => {
                onGoToChat(chat.id);
              }}
            >
              {chat.title}
            </button>
          </div>
        ))}
      </div>
    </div>
  );
};

const AppWrapper: React.FC = () => {
  const api = useMemo(() => new API(BASE_URL, API_KEY), []);
  const [configAtom, setConfigAtom] = useState<Atom<ConfigResponse>>();
  const [dataAtom, setDataAtom] = useState<Atom<DataResponse>>();
  const [chatId, setChatId] = useState<number>();
  const [route, setRoute] = useState<"index" | "chats">("index");
  const _init = useRef(false);
  async function init() {
    if (_init.current) return;
    _init.current = true;
    try {
      const [config, data, chat] = await Promise.all([
        api.getConfig(),
        api.getData(),
        api.rpc("create_chat", {}) as Promise<{ chat_id: number }>,
      ]);
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
    case "index":
      return (
        <App
          api={api}
          configAtom={configAtom}
          dataAtom={dataAtom}
          chatId={chatId}
          onGoToChats={() => {
            setRoute("chats");
          }}
          onReset={() => {
            setConfigAtom(undefined);
            setDataAtom(undefined);
            setChatId(undefined);
          }}
        />
      );
    case "chats":
      return (
        <Chats
          api={api}
          configAtom={configAtom}
          dataAtom={dataAtom}
          onGoToIndex={() => {
            setRoute("index");
          }}
          onGoToChat={(chatId) => {
            setChatId(chatId);
            setRoute("index");
          }}
        />
      );
    default:
      assertNever(route);
  }
};

export default AppWrapper;
