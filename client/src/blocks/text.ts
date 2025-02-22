import { z } from "zod";

const TextBlock = z.object({
  id: z.string(),
  type: z.literal("text"),
  role: z.union([z.literal("user"), z.literal("assistant")]),
  content: z.string(),
});
type TextBlock = z.infer<typeof TextBlock>;

export { TextBlock };
