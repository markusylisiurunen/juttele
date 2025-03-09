import { z } from "zod";

const ToolBlock = z.object({
  id: z.string(),
  ts: z.string().datetime(),
  hash: z.string(),
  type: z.literal("tool"),
  name: z.string(),
  args: z.string(),
  result: z.string().optional(),
  error: z.object({ code: z.number(), message: z.string() }).optional(),
});
type ToolBlock = z.infer<typeof ToolBlock>;

export { ToolBlock };
