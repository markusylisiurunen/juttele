import { z } from "zod";

const ErrorBlock = z.object({
  id: z.string(),
  ts: z.string().datetime(),
  hash: z.string(),
  type: z.literal("error"),
  error: z.object({ code: z.number(), message: z.string() }),
});
type ErrorBlock = z.infer<typeof ErrorBlock>;

export { ErrorBlock };
