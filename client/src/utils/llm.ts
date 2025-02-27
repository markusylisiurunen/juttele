import { z } from "zod";
import { AnyBlock } from "../blocks";
import { Tool } from "../tools";

const toolCallMessage = z.object({
  jsonrpc: z.literal("2.0"),
  id: z.number(),
  method: z.literal("tool_call"),
  params: z.object({
    name: z.string(),
    args: z.string(),
  }),
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
  useTools: boolean,
  content: string,
  tools: Tool[],
  onMessage: (message: StreamMessage) => void
): Promise<void> {
  const wsBaseUrl = baseUrl.replace(/^http/, "ws");
  const wsUrl = `${wsBaseUrl}/chats/${chatId}?api_key=${encodeURIComponent(apiKey)}`;
  return new Promise((resolve, reject) => {
    const socket = new WebSocket(wsUrl);
    socket.onopen = () => {
      socket.send(
        JSON.stringify({
          model_id: modelId,
          personality_id: personalityId,
          content: content,
          tools: tools.map((tool) => ({ name: tool.Name, spec: tool.Spec })),
          use_tools: useTools,
        })
      );
    };
    socket.onmessage = (event) => {
      try {
        const parsed = StreamMessage.safeParse(JSON.parse(event.data));
        if (!parsed.success) {
          console.error(`received an unexpected message: ${event.data}`);
          return;
        }
        const data = parsed.data;
        if (data.method === "tool_call") {
          const tool = tools.find((tool) => tool.Name === data.params.name);
          if (tool) {
            void Promise.resolve().then(async () => {
              try {
                const result = await tool.Call(data.params.args);
                socket.send(
                  JSON.stringify({
                    jsonrpc: "2.0",
                    result: result,
                    id: data.id,
                  })
                );
              } catch (error) {
                console.error(`error calling tool "${tool.Name}":`, error);
                socket.send(
                  JSON.stringify({
                    jsonrpc: "2.0",
                    error: {
                      code: 0,
                      message: error instanceof Error ? error.message : "Something went wrong.",
                    },
                    id: data.id,
                  })
                );
              }
            });
          } else {
            socket.send(
              JSON.stringify({
                jsonrpc: "2.0",
                error: {
                  code: 0,
                  message: `Tool "${data.params.name}" not found.`,
                },
                id: data.id,
              })
            );
          }
          return;
        }
        onMessage(parsed.data);
      } catch (error) {
        console.error("error parsing websocket message:", error);
      }
    };
    let socketErrorOccurred = false;
    socket.onerror = (error) => {
      socketErrorOccurred = true;
      console.error("WebSocket error:", error);
      reject(new Error(`WebSocket error`));
    };
    socket.onclose = (event) => {
      if (event.code === 1000 || event.code === 1001 || event.code === 1005) {
        resolve();
      } else if (event.code === 1006 && !socketErrorOccurred) {
        resolve();
      } else {
        reject(new Error(`WebSocket closed with code: ${event.code}, reason: ${event.reason}`));
      }
    };
  });
}

export { streamCompletion, StreamMessage };
