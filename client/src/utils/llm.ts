type Message = {
  role: "user" | "assistant";
  content: string;
};

async function streamCompletion(
  modelId: string,
  personalityId: string,
  history: Message[],
  onThinking: (delta: string) => void,
  onContent: (delta: string) => void,
  onError: (error: string) => void
): Promise<void> {
  const resp = await fetch("http://localhost:8765/stream", {
    method: "POST",
    headers: { Authorization: "Bearer dev" },
    body: JSON.stringify({
      model_id: modelId,
      model_personality_id: personalityId,
      history: history,
    }),
  });
  if (!resp.ok) {
    throw new Error(`unexpected status: ${resp.status}`);
  }
  if (!resp.body) {
    throw new Error("missing response body");
  }
  const reader = resp.body.getReader();
  const decoder = new TextDecoder();
  try {
    while (true) {
      const { value, done }: ReadableStreamReadResult<Uint8Array> = await reader.read();
      if (done) break;
      const chunk = decoder.decode(value);
      const lines = chunk.split("\n").filter((line) => line.trim() !== "");
      for (const line of lines) {
        if (line.startsWith("data: ")) {
          const data = line.slice(6);
          const parsed = JSON.parse(data) as {
            error?: string;
            thinking?: string;
            content?: string;
          };
          if (parsed.error) {
            onError(parsed.error);
          } else if (parsed.thinking) {
            onThinking(parsed.thinking);
          } else if (parsed.content) {
            onContent(parsed.content);
          }
        }
      }
    }
  } finally {
    reader.releaseLock();
  }
}

export { streamCompletion };
