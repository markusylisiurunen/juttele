import React from "react";
import { AnyBlock } from "../../blocks";
import { Block } from "../Block/Block";
import styles from "./ChatHistory.module.css";

// prettier-ignore
const demoHistory: AnyBlock[] = [
  {
    id: "1",
    type: "text",
    role: "user",
    content: "i need the latest news on AI",
  },
  {
    id: "2",
    type: "thinking",
    content: "Thinking for 16 s...",
  },
  {
    id: "4",
    type: "text",
    role: "assistant",
    content: "Sure, I can help you with that. Please provide more details about the task you need assistance with.",
  },
  {
    id: "5",
    type: "tool_call",
    name: "fetch_data",
    args: {
      query: "latest news on AI"
    },
  },
  {
    id: "6",
    type: "text",
    role: "assistant",
    content: 'Here is the latest news on AI:\n\n## AI in Healthcare\n\nAI is revolutionizing healthcare by improving `diagnostics` and treatment plans. Recent advancements include AI-driven imaging `techniques` and personalized medicine.\n\n## AI in Finance\n\nFinancial institutions are leveraging AI for fraud detection, risk management, and personalized banking experiences. AI algorithms are also being used for high-frequency trading.\n\n## AI in Transportation\n\nSelf-driving cars and AI-powered traffic management systems are making transportation safer and more efficient. Companies like Tesla and Waymo are at the forefront of this innovation.\n\n```json\n{"ok":true}\n```',
  },
  {
    id: "7",
    type: "tool_call",
    name: "summarize_text",
    args: {
      text: "AI is making significant strides in various industries, including healthcare, finance, and transportation. It is enhancing diagnostics, improving financial services, and making transportation safer.",
    },
  },
];

type ChatHistoryProps = {
  blocks: AnyBlock[];
};
const ChatHistory: React.FC<ChatHistoryProps> = ({ blocks }) => {
  return (
    <div className={styles.root}>
      <div className={styles.history}>
        {blocks.map((i) => {
          switch (i.type) {
            case "text":
              return <Block.Text key={i.id} block={i} />;
            case "thinking":
              return <Block.Thinking key={i.id} active={true} block={i} />;
            case "tool_call":
              return <Block.ToolCall key={i.id} block={i} />;
            default:
              return null;
          }
        })}
      </div>
    </div>
  );
};

export { ChatHistory };
