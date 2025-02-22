import { z } from "zod";

const ToolCallBlock = z.object({
  id: z.string(),
  type: z.literal("tool_call"),
  name: z.string(),
  args: z.string(),
});
type ToolCallBlock = z.infer<typeof ToolCallBlock>;

export { ToolCallBlock };
