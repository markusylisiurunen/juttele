type ChatHistoryItem = {
  id: string;
  role: "user" | "assistant";
  thinking?: string;
  content: string;
};

export { type ChatHistoryItem };
