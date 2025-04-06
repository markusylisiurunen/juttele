import { BaseDirectory, dirname, homeDir, join, resolve } from "@tauri-apps/api/path";
import { exists, readTextFile, writeTextFile } from "@tauri-apps/plugin-fs";
import { ChildProcess, Command } from "@tauri-apps/plugin-shell";
import { z } from "zod";
import { type Tool } from "./index";

function makeEditFileTool(baseDir: string): Tool {
  return {
    Name: "edit_file",
    Spec: {
      name: "edit_file",
      description: [
        "Edit file contents by replacing a specific section.",
        "The tool works as follows:",
        "1. First, it finds the FIRST line containing your 'unique_string' (the string must exist within a single line, not across multiple lines)",
        "2. Starting from that line, it counts 'num_lines' lines (including the first line with the unique string)",
        "3. It replaces these exact lines with your new 'content' (which can have any number of lines)",
        "4. Example: If 'unique_string' is found on line 10 and 'num_lines' is 3, then lines 10-12 will be replaced with your new content",
        "5. The tool will fail if: the unique string isn't found, or if it appears in multiple lines",
        "6. All indentation in the replacement content must be provided explicitly",
        "7. If you encounter errors with this tool, fall back to using the 'write_file' tool to replace the entire file content",
        '8. Example usage: Original file contains: function hello() {\\n  console.log("Hello");\\n  console.log("World");\\n  return true;\\n}. Using edit_file with unique_string="console.log(\\"Hello\\")" and num_lines=2 and content="  console.log(\\"Hello, World!\\");" would result in: function hello() {\\n  console.log("Hello, World!");\\n  return true;\\n}',
      ].join(" "),
      parameters: {
        type: "object",
        properties: {
          file_path: {
            type: "string",
            description: "The relative path to the file to read.",
          },
          unique_string: {
            type: "string",
            description: "A unique string to identify the first line to edit.",
          },
          num_lines: {
            type: "integer",
            description: "The number of lines to replace with the new content.",
          },
          content: {
            type: "string",
            description: "The new content to write to the file starting from the unique string.",
          },
        },
        required: ["file_path", "unique_string", "num_lines", "content"],
        additionalProperties: false,
      },
    },
    Call: async (args) => {
      let filePath: string;
      let uniqueString: string;
      let numLines: number;
      let content: string;
      try {
        const parsed = z
          .object({
            file_path: z.string(),
            unique_string: z.string(),
            num_lines: z.number().int(),
            content: z.string(),
          })
          .parse(JSON.parse(args));
        filePath = parsed.file_path;
        uniqueString = parsed.unique_string;
        numLines = parsed.num_lines;
        content = parsed.content;
      } catch (error) {
        console.error("error parsing arguments:", error);
        throw new Error("Error parsing arguments.");
      }
      await assertGitRoot(baseDir);
      const fileContent = await readFileContent(baseDir, filePath);
      const lines = fileContent.split("\n");
      const linesHavingUniqueString = lines.filter((line) => line.includes(uniqueString));
      if (linesHavingUniqueString.length === 0) {
        console.error("unique string not found in file");
        throw new Error("Unique string not found in file.");
      }
      if (linesHavingUniqueString.length > 1) {
        console.error("multiple lines found with the unique string");
        throw new Error("Multiple lines found with the unique string.");
      }
      const lineIndex = lines.indexOf(linesHavingUniqueString[0]);
      const startIndex = Math.max(0, lineIndex);
      const endIndex = Math.min(lines.length, lineIndex + numLines);
      const newLines = lines
        .slice(0, startIndex)
        .concat(content.split("\n"), lines.slice(endIndex));
      const newContent = newLines.join("\n");
      await writeFileContent(baseDir, filePath, newContent);
      return JSON.stringify({ ok: true, edited_content: newContent });
    },
  };
}

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
      let include: string | undefined;
      try {
        const parsed = z
          .object({
            pattern: z.string(),
            include: z.string().optional(),
          })
          .parse(JSON.parse(args));
        pattern = parsed.pattern;
        include = parsed.include;
      } catch (error) {
        console.error("error parsing arguments:", error);
        throw new Error("Error parsing grep arguments.");
      }
      try {
        await assertGitRoot(baseDir);
        // build the search path
        const absoluteSearchPath = await resolve(await homeDir(), baseDir);
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
        const nonIgnoredSet = new Set(await listNonIgnoredFiles(baseDir));
        const matches = output.stdout
          .split("\n")
          .filter((line) => line.trim().length > 0)
          .map((line) => (line.startsWith("./") ? line.slice(2) : line))
          .filter((file) => nonIgnoredSet.has(file));
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
            description: "The relative path to the file to write.",
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

export { makeEditFileTool, makeGrepTool, makeListFilesTool, makeReadFileTool, makeWriteFileTool };

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
