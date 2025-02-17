async function streamCompletion(
  chatId: number,
  modelId: string,
  personalityId: string,
  content: string,
  onThinking: (delta: string) => void,
  onContent: (delta: string) => void,
  onError: (error: string) => void
): Promise<void> {
  const resp = await fetch(`${import.meta.env.VITE_API_BASE_URL}/chats/${chatId}`, {
    method: "POST",
    headers: { Authorization: `Bearer ${import.meta.env.VITE_API_KEY}` },
    body: JSON.stringify({
      model_id: modelId,
      personality_id: personalityId,
      content: content,
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
