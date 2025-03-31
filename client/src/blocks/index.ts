import { z } from "zod";
import { ErrorBlock } from "./error";
import { TextBlock } from "./text";
import { ThinkingBlock } from "./thinking";
import { ToolBlock } from "./tool";

const AnyBlock = z.union([ErrorBlock, TextBlock, ToolBlock, ThinkingBlock]);
type AnyBlock = z.infer<typeof AnyBlock>;

export { AnyBlock, ErrorBlock, TextBlock, ThinkingBlock, ToolBlock };
