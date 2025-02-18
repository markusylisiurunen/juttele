import "./styles/globals.css";

import React, { useMemo, useRef, useState } from "react";
import { API, ConfigResponse, DataResponse } from "./api";
import { ChatHistory, MessageBox } from "./components";
import { ChatHistoryItem } from "./entities";
import { atom, Atom, streamCompletion } from "./utils";

const BASE_URL = import.meta.env.VITE_API_BASE_URL;
const API_KEY = import.meta.env.VITE_API_KEY;

type AppProps = {
  api: API;
  configAtom: Atom<ConfigResponse>;
  dataAtom: Atom<DataResponse>;
  chatId: number;
  onReset: () => void;
};
const App: React.FC<AppProps> = ({ configAtom, chatId, onReset }) => {
  const [model, setModel] = useState<{ modelId: string; personalityId: string }>();
  const [history, setHistory] = useState([] as ChatHistoryItem[]);
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
  function onControlReset() {
    onReset();
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
        onControlReset={onControlReset}
        onControlModelChange={onControlModelChange}
      />
    </div>
  );
};

const AppWrapper: React.FC = () => {
  const api = useMemo(() => new API(BASE_URL, API_KEY), []);
  const [configAtom, setConfigAtom] = useState<Atom<ConfigResponse>>();
  const [dataAtom, setDataAtom] = useState<Atom<DataResponse>>();
  const [chatId, setChatId] = useState<number>();
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
  return (
    <App
      api={api}
      configAtom={configAtom}
      dataAtom={dataAtom}
      chatId={chatId}
      onReset={() => {
        setConfigAtom(undefined);
        setDataAtom(undefined);
        setChatId(undefined);
      }}
    />
  );
};

export default AppWrapper;
