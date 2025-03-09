import { z } from "zod";

const ThinkingBlock = z.object({
  id: z.string(),
  ts: z.string().datetime(),
  hash: z.string(),
  type: z.literal("thinking"),
  content: z.string(),
});
type ThinkingBlock = z.infer<typeof ThinkingBlock>;

export { ThinkingBlock };
