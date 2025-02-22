import { z } from "zod";
import { TextBlock } from "./text";
import { ThinkingBlock } from "./thinking";
import { ToolCallBlock } from "./tool-call";

const AnyBlock = z.union([TextBlock, ToolCallBlock, ThinkingBlock]);
type AnyBlock = z.infer<typeof AnyBlock>;

export { AnyBlock, TextBlock, ThinkingBlock, ToolCallBlock };
