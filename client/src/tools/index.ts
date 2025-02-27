export interface Tool {
  Name: string;
  Spec: Record<string, unknown>;
  Call(args: string): Promise<string>;
}

export * from "./fs";
