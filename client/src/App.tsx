import "./styles/globals.css";

import React, { useState } from "react";
import { ChatHistory, MessageBox } from "./components";
import { streamCompletion } from "./utils";

type ChatHistoryItem = {
  id: string;
  role: "user" | "assistant";
  thinking?: string;
  content: string;
};

const App: React.FC = () => {
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
          _model.modelId,
          _model.personalityId,
          _history,
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
    setHistory([]);
  }
  function onControlModelChange(modelId: string, personalityId: string) {
    setModel({ modelId, personalityId });
  }
  return (
    <div className="container">
      <ChatHistory history={history} />
      <MessageBox
        onMessage={onMessage}
        onControlReset={onControlReset}
        onControlModelChange={onControlModelChange}
      />
    </div>
  );
};

export default App;
