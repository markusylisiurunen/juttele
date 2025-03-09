# Protocol

This document specifies the JSON-RPC 2.0 protocol used for communication between Juttele clients and servers. Juttele is a real-time chat application that enables AI-assisted conversations with streaming responses and client-side tool execution.

## Core concepts

### Chats

A chat represents a conversation session containing an ordered sequence of blocks. Each chat has a unique identifier, timestamp, title, and an array of blocks.

```json
{
  "id": "01957a7d-9bba-7651-9361-5b2f9327482a",
  "ts": "2025-03-09T12:01:00.538Z",
  "title": "A conversation about AGI implications",
  "blocks": [
    {
      "id": "01957ac5-3560-7b19-b185-2ffbdd97f724",
      "ts": "2025-03-09T12:01:20.702Z",
      "hash": "2426039091713404309",
      "type": "thinking",
      "content": "Okay, let me see...",
      "duration": 7467
    },
    {
      "id": "01957ac5-5dce-718e-b7c7-af71a3dc78d9",
      "ts": "2025-03-09T12:01:29.001Z",
      "hash": "6866007823030491827",
      "type": "text",
      "role": "assistant",
      "content": "Here is the answer to your question."
    }
  ]
}
```

### Blocks

Blocks are the fundamental content units within a chat. Each block has a specific type that determines its structure and purpose.

All blocks include these properties:

- `id`: Unique identifier (string, required)
- `ts`: Creation timestamp (string, required)
- `hash`: Content hash used for detecting changes (number, required)
- `type`: Block type identifier (string, required)

#### Block types

##### Thinking block

Represents the model's thought process, typically shown while generating a response.

```json
{
  "id": "01957ac5-3560-7b19-b185-2ffbdd97f724",
  "ts": "2025-03-09T12:00:00.000Z",
  "hash": "2426039091713404309",
  "type": "thinking",
  "content": "Okay, let me see...",
  "duration": 7467
}
```

##### Text block

Contains message content with role attribution.

```json
{
  "id": "01957ac5-3560-7b19-b185-2ffbdd97f724",
  "ts": "2025-03-09T12:00:00.000Z",
  "hash": "2426039091713404309",
  "type": "text",
  "role": "assistant",
  "content": "Here is the answer to your question."
}
```

##### Tool block

Represents a tool execution with inputs and outputs. The error field is null on success.

Success example:

```json
{
  "id": "01957ac5-3560-7b19-b185-2ffbdd97f724",
  "ts": "2025-03-09T12:00:00.000Z",
  "hash": "2426039091713404309",
  "type": "tool",
  "name": "calculator",
  "args": "{\"op\": \"*\", \"a\": 10, \"b\": 4.2}",
  "result": "42",
  "error": null
}
```

Error example:

```json
{
  "id": "01957ac5-3560-7b19-b185-2ffbdd97f724",
  "ts": "2025-03-09T12:00:00.000Z",
  "hash": "2426039091713404309",
  "type": "tool",
  "name": "calculator",
  "args": "{\"op\": \"/\", \"a\": 10, \"b\": 0}",
  "result": null,
  "error": {
    "code": -32603,
    "message": "Division by zero."
  }
}
```

##### Error block

Indicates an error in the chat session. The error field contains the error message.

```json
{
  "id": "01957ac5-3560-7b19-b185-2ffbdd97f724",
  "ts": "2025-03-09T12:00:00.000Z",
  "hash": "2426039091713404309",
  "type": "error",
  "error": {
    "code": -32603,
    "message": "An error occurred."
  }
}
```

## Communication protocol

### Session initialization

The client initiates a session by connecting to the server's WebSocket endpoint and sending an init notification:

```json
{
  "jsonrpc": "2.0",
  "method": "init",
  "params": {
    "chat": "01957a7d-9bba-7651-9361-5b2f9327482a",
    "content": "Hello, how are you?",
    "config": {
      "model": "anthropic_12956409826453017813",
      "personality": "6650923681385875244",
      "tools": false
    }
  }
}
```

### Block streaming

As the server processes the request, it streams response blocks to the client using `block` notifications:

```json
{
  "jsonrpc": "2.0",
  "method": "block",
  "params": {
    "id": "01957ac5-3560-7b19-b185-2ffbdd97f724",
    "ts": "2025-03-09T12:01:20.702Z",
    "hash": "2426039091713404309",
    "type": "thinking",
    "content": "Okay, let me see...",
    "duration": 7467
  }
}
```

Clients should process these notifications as follows:

- If the block ID is new, create a new block
- If the block ID already exists, update the existing block with the new content

### Client-side tool execution

The protocol supports executing tools on the client side, enabling integration with local capabilities like file system access or calculations.

#### Tool request (server to client)

When the model needs to use a client-side tool, the server sends a tool request:

```json
{
  "jsonrpc": "2.0",
  "method": "tool",
  "params": {
    "name": "calculator",
    "args": "{\"op\": \"*\", \"a\": 10, \"b\": 4.2}"
  },
  "id": 1
}
```

#### Tool response (client to server)

The client executes the requested tool and responds with either a success or error message.

Success response:

```json
{
  "jsonrpc": "2.0",
  "result": "42",
  "id": 1
}
```

Error response:

```json
{
  "jsonrpc": "2.0",
  "error": {
    "code": -32603,
    "message": "An error occurred while executing the tool."
  },
  "id": 1
}
```
