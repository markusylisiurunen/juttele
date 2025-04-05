import { BaseDirectory, dirname, homeDir, join, resolve } from "@tauri-apps/api/path";
import { exists, readTextFile, writeTextFile } from "@tauri-apps/plugin-fs";
import { ChildProcess, Command } from "@tauri-apps/plugin-shell";
import { z } from "zod";
import { type Tool } from "./index";

function makeGrepTool(baseDir: string): Tool {
  return {
    Name: "grep",
    Spec: {
      name: "grep",
      description: [
        `- Fast content search tool that works with any codebase size`,
        `- Searches file contents using regular expressions`,
        `- Supports full regex syntax (eg. "log.*Error", "function\\s+\\w+", etc.)`,
        `- Filter files by pattern with the include parameter (eg. "*.js", "*.{ts,tsx}"?)`,
        `- Returns matching file paths`,
        `- Use this tool when you need to find files containing specific patterns`,
      ].join("\n"),
      parameters: {
        type: "object",
        properties: {
          pattern: {
            type: "string",
            description: `The regular expression pattern to search for in file contents.`,
          },
          path: {
            type: "string",
            description: `The directory to search in. Defaults to the current working directory.`,
          },
          include: {
            type: "string",
            description: `File pattern to include in the search (e.g. "*.js", "*.{ts,tsx}")`,
          },
        },
        required: ["pattern"],
        additionalProperties: false,
      },
    },
    Call: async (args) => {
      let pattern: string;
      let path: string | undefined;
      let include: string | undefined;
      try {
        const parsed = z
          .object({
            pattern: z.string(),
            path: z.string().optional(),
            include: z.string().optional(),
          })
          .parse(JSON.parse(args));
        pattern = parsed.pattern;
        path = parsed.path;
        include = parsed.include;
      } catch (error) {
        console.error("error parsing arguments:", error);
        throw new Error("Error parsing grep arguments.");
      }
      try {
        await assertGitRoot(baseDir);
        // build the search path
        const searchPath = path ? await join(baseDir, path) : baseDir;
        const absoluteSearchPath = await resolve(await homeDir(), searchPath);
        // prepare ripgrep command arguments
        const rgArgs = ["-li", pattern]; // -l: only file names, -i: case insensitive
        if (include) rgArgs.push("--glob", include);
        rgArgs.push(".");
        // execute ripgrep command
        const command = Command.create("rg", rgArgs, { cwd: absoluteSearchPath });
        const output = await command.execute();
        if (output.code !== 0 && output.stderr) {
          console.error(`ripgrep error: ${output.stderr}`);
          throw new Error(`Error executing grep: ${output.stderr}`);
        }
        // process results
        const matches = output.stdout
          .split("\n")
          .filter((line) => line.trim().length > 0)
          .map((line) => (line.startsWith("./") ? line.slice(2) : line));
        const MAX_RESULTS = 100;
        const truncated = matches.length > MAX_RESULTS;
        const results = matches.slice(0, MAX_RESULTS);
        return JSON.stringify({
          num_files: matches.length,
          file_names: results,
          truncated: truncated,
          message: truncated
            ? `Found ${matches.length} files (showing first ${MAX_RESULTS})`
            : `Found ${matches.length} file${matches.length === 1 ? "" : "s"}`,
        });
      } catch (error) {
        console.error("error during grep:", error);
        throw new Error(`Error executing grep: ${error}`);
      }
    },
  };
}

function makeListFilesTool(baseDir: string): Tool {
  return {
    Name: "list_files",
    Spec: {
      name: "list_files",
      description: [
        "Lists all files in the current directory.",
        "Only to be used when the user explicitly asks to.",
      ].join(" "),
      parameters: {
        type: "object",
        properties: {},
        required: [],
        additionalProperties: false,
      },
    },
    Call: async () => {
      await assertGitRoot(baseDir);
      const files = await listNonIgnoredFiles(baseDir);
      if (files.length === 0) {
        return "No git-tracked files found.";
      }
      return JSON.stringify(files);
    },
  };
}

function makeReadFileTool(baseDir: string): Tool {
  return {
    Name: "read_file",
    Spec: {
      name: "read_file",
      description: [
        "Reads a file in the current directory.",
        "Only to be used when the user explicitly asks to.",
      ].join(" "),
      parameters: {
        type: "object",
        properties: {
          file_path: {
            type: "string",
            description: "The relative path to the file to read.",
          },
        },
        required: ["file_path"],
        additionalProperties: false,
      },
    },
    Call: async (args) => {
      let filePath: string;
      try {
        const parsed = z.object({ file_path: z.string() }).parse(JSON.parse(args));
        filePath = parsed.file_path;
      } catch (error) {
        console.error("error parsing arguments:", error);
        throw new Error("Error parsing arguments.");
      }
      await assertGitRoot(baseDir);
      const content = await readFileContent(baseDir, filePath);
      return JSON.stringify({ content: content });
    },
  };
}

function makeWriteFileTool(baseDir: string): Tool {
  return {
    Name: "write_file",
    Spec: {
      name: "write_file",
      description: [
        "Write to a file in the current directory.",
        "If the file does not exist, it will be created.",
        "Only to be used when the user explicitly asks to.",
      ].join(" "),
      parameters: {
        type: "object",
        properties: {
          file_path: {
            type: "string",
            description: "The relative path to the file to read.",
          },
          content: {
            type: "string",
            description: "The content to write to the file.",
          },
        },
        required: ["file_path", "content"],
        additionalProperties: false,
      },
    },
    Call: async (args) => {
      let filePath: string;
      let content: string;
      try {
        const parsed = z
          .object({ file_path: z.string(), content: z.string() })
          .parse(JSON.parse(args));
        filePath = parsed.file_path;
        content = parsed.content;
      } catch (error) {
        console.error("error parsing arguments:", error);
        throw new Error("Error parsing arguments.");
      }
      await assertGitRoot(baseDir);
      await writeFileContent(baseDir, filePath, content);
      return JSON.stringify({ ok: true });
    },
  };
}

export { makeGrepTool, makeListFilesTool, makeReadFileTool, makeWriteFileTool };

//---

async function assertGitRoot(startPath: string): Promise<void> {
  try {
    let currentPath = await resolve(await homeDir(), startPath);
    let previousPath = "";
    while (currentPath !== previousPath) {
      const gitDir = await join(currentPath, ".git");
      if (await exists(gitDir)) return;
      previousPath = currentPath;
      currentPath = await dirname(currentPath);
    }
    throw new Error("Not in a git repository. This tool only lists git-tracked files.");
  } catch (error) {
    console.error("error finding git root:", error);
    throw new Error("Not in a git repository. This tool only lists git-tracked files.");
  }
}

async function listNonIgnoredFiles(baseDir: string): Promise<string[]> {
  let output: ChildProcess<string>;
  try {
    const cwd = await join(await homeDir(), baseDir);
    const command = Command.create(
      "git",
      ["ls-files", "--cached", "--others", "--exclude-standard"],
      { cwd }
    );
    output = await command.execute();
  } catch (error) {
    console.error("error executing git ls-files:", error);
    throw new Error("Error listing files.");
  }
  if (output.code !== 0) {
    console.error(`git ls-files failed: ${output.stderr}`);
    throw new Error("Error listing files.");
  }
  const nonEmptyFiles = output.stdout.split("\n").filter((v) => v.length > 0);
  return nonEmptyFiles;
}

function shouldSkipFile(filePath: string): boolean {
  // prettier-ignore
  const binaryExtensions = [
    ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".ico", ".webp", // images
    ".mp3", ".mp4", ".avi", ".mov", ".wav", // audio/video
    ".zip", ".tar", ".gz", ".7z", ".rar", // archives
    ".exe", ".dll", ".so", ".dylib", ".class", ".pyc", // compiled/binary
    ".ttf", ".otf", ".woff", ".woff2", // fonts
    ".pdf", ".doc", ".xls", // other
  ];
  const extension = filePath.substring(filePath.lastIndexOf(".")).toLowerCase();
  return binaryExtensions.includes(extension);
}

async function readFileContent(baseDir: string, filePath: string): Promise<string> {
  try {
    const path = await join(baseDir, filePath);
    const content = await readTextFile(path, { baseDir: BaseDirectory.Home });
    return content;
  } catch (error) {
    console.error("error reading file:", error);
    throw new Error("Error reading file.");
  }
}

async function writeFileContent(baseDir: string, filePath: string, content: string): Promise<void> {
  try {
    const path = await join(baseDir, filePath);
    await writeTextFile(path, content, { baseDir: BaseDirectory.Home });
  } catch (error) {
    console.error("error writing file:", error);
    throw new Error("Error writing file.");
  }
}
