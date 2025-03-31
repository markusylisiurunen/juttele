import { z } from "zod";
import { AnyBlock } from "../blocks";

const ConfigResponse = z.object({
  models: z.array(
    z.object({
      id: z.string(),
      name: z.string(),
      personalities: z.array(
        z.object({
          id: z.string(),
          name: z.string(),
        })
      ),
    })
  ),
});
type ConfigResponse = z.infer<typeof ConfigResponse>;

const DataResponse = z.object({
  chats: z.array(
    z.object({
      id: z.number(),
      ts: z.string().datetime(),
      title: z.string(),
      blocks: z.array(AnyBlock),
    })
  ),
});
type DataResponse = z.infer<typeof DataResponse>;

function makeGetConfig(baseUrl: string, token: string) {
  return async (): Promise<ConfigResponse> => {
    const resp = await fetch(`${baseUrl}/config`, {
      method: "GET",
      headers: {
        Accept: "application/json",
        Authorization: `Bearer ${token}`,
      },
    });
    const data = await resp.json();
    return ConfigResponse.parse(data);
  };
}

function makeGetData(baseUrl: string, token: string) {
  return async (): Promise<DataResponse> => {
    const resp = await fetch(`${baseUrl}/data`, {
      method: "GET",
      headers: {
        Accept: "application/json",
        Authorization: `Bearer ${token}`,
      },
    });
    const data = await resp.json();
    return DataResponse.parse(data);
  };
}

function makeRpc(baseUrl: string, token: string) {
  return async (op: string, args: Record<string, unknown>): Promise<unknown> => {
    const resp = await fetch(`${baseUrl}/rpc`, {
      method: "POST",
      headers: {
        Accept: "application/json",
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ op, args }),
    });
    const data = await resp.json();
    return data;
  };
}

class API {
  constructor(private baseUrl: string, private token: string) {}

  async getConfig() {
    return makeGetConfig(this.baseUrl, this.token)();
  }

  async getData() {
    return makeGetData(this.baseUrl, this.token)();
  }

  async rpc(op: string, args: Record<string, unknown>) {
    return makeRpc(this.baseUrl, this.token)(op, args);
  }
}

export { API, ConfigResponse, DataResponse };
