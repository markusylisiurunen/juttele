import { z } from "zod";
import { AnyBlock } from "../blocks";

const toolCallMessage = z.object({
  jsonrpc: z.literal("2.0"),
  id: z.number(),
  method: z.literal("tool_call"),
  params: z.record(z.unknown()),
});
const blockNotification = z.object({
  jsonrpc: z.literal("2.0"),
  method: z.literal("block"),
  params: AnyBlock,
});
const StreamMessage = z.union([toolCallMessage, blockNotification]);
type StreamMessage = z.infer<typeof StreamMessage>;

async function streamCompletion(
  baseUrl: string,
  apiKey: string,
  chatId: number,
  modelId: string,
  personalityId: string,
  includeTools: boolean,
  content: string,
  onMessage: (message: StreamMessage) => void
): Promise<void> {
  const resp = await fetch(`${baseUrl}/chats/${chatId}`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${apiKey}`,
    },
    body: JSON.stringify({
      model_id: modelId,
      personality_id: personalityId,
      include_tools: includeTools,
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
          const parsed = StreamMessage.safeParse(JSON.parse(data));
          if (!parsed.success) {
            console.error(`received an unexpected message: ${data}`);
            continue;
          }
          onMessage(parsed.data);
        }
      }
    }
  } finally {
    reader.releaseLock();
  }
}

export { streamCompletion, StreamMessage };
