#!/usr/bin/env bun

import { existsSync, readFileSync } from "node:fs";
import { exit } from "node:process";

export type ToolUse = {
  type: string;
  name?: string;
  input?: Record<string, any>;
  id?: string;
};

export type ToolResult = {
  type: string;
  tool_use_id?: string;
  content?: any;
  is_error?: boolean;
};

export type ContentItem = {
  type: string;
  text?: string;
  tool_use_id?: string;
  content?: any;
  is_error?: boolean;
  name?: string;
  input?: Record<string, any>;
  id?: string;
};

export type Message = {
  content: ContentItem[];
  usage?: {
    input_tokens?: number;
    output_tokens?: number;
  };
};

export type Turn = {
  type: string;
  subtype?: string;
  message?: Message;
  tools?: any[];
  cost_usd?: number;
  duration_ms?: number;
  result?: string;
};

export type GroupedContent = {
  type: string;
  tools_count?: number;
  data?: Turn;
  text_parts?: string[];
  tool_calls?: { tool_use: ToolUse; tool_result?: ToolResult }[];
  usage?: Record<string, number>;
};

// 头尾保留的截断：长 trace 的 tail（栈底 / 报错位置）往往比 head 重要。
export function truncateMiddle(s: string, max: number): string {
  if (s.length <= max) return s;
  const headLen = Math.floor((max - 40) * 0.5);
  const tailLen = max - 40 - headLen;
  const omitted = s.length - headLen - tailLen;
  return `${s.substring(0, headLen)}\n\n… [省略 ${omitted} 字符] …\n\n${s.substring(s.length - tailLen)}`;
}

// 把任意内容（string / array / object）抽成可读字符串，复用 formatResultContent 的解包逻辑。
export function extractResultText(content: any): string {
  if (content == null) return "";
  try {
    const parsed = typeof content === "string" ? JSON.parse(content) : content;
    if (
      Array.isArray(parsed) &&
      parsed.length > 0 &&
      typeof parsed[0] === "object" &&
      parsed[0]?.type === "text"
    ) {
      return String(parsed[0]?.text || "").trim();
    }
  } catch {
    // not JSON, fall through
  }
  return String(content).trim();
}

// 给折叠块用的 fence 选择：大多数情况直接 ``` 即可；如果内容里出现了三个反引号，用四个。
export function fenceFor(s: string): string {
  return s.includes("```") ? "````" : "```";
}

export function detectContentType(content: any): string {
  const contentStr = String(content).trim();

  // Check for JSON
  if (contentStr.startsWith("{") && contentStr.endsWith("}")) {
    try {
      JSON.parse(contentStr);
      return "json";
    } catch {
      // Fall through
    }
  }

  if (contentStr.startsWith("[") && contentStr.endsWith("]")) {
    try {
      JSON.parse(contentStr);
      return "json";
    } catch {
      // Fall through
    }
  }

  // Check for code-like content
  const codeKeywords = [
    "def ",
    "class ",
    "import ",
    "from ",
    "function ",
    "const ",
    "let ",
    "var ",
  ];
  if (codeKeywords.some((keyword) => contentStr.includes(keyword))) {
    if (
      contentStr.includes("def ") ||
      contentStr.includes("import ") ||
      contentStr.includes("from ")
    ) {
      return "python";
    }
    if (
      ["function ", "const ", "let ", "var ", "=>"].some((js) =>
        contentStr.includes(js),
      )
    ) {
      return "javascript";
    }
    return "python"; // default for code
  }

  // Check for shell/bash output
  const shellIndicators = ["ls -", "cd ", "mkdir ", "rm ", "$ ", "# "];
  if (
    contentStr.startsWith("/") ||
    contentStr.includes("Error:") ||
    contentStr.startsWith("total ") ||
    shellIndicators.some((indicator) => contentStr.includes(indicator))
  ) {
    return "bash";
  }

  // Check for diff format
  if (
    contentStr.startsWith("@@") ||
    contentStr.includes("+++ ") ||
    contentStr.includes("--- ")
  ) {
    return "diff";
  }

  // Check for HTML/XML
  if (contentStr.startsWith("<") && contentStr.endsWith(">")) {
    return "html";
  }

  // Check for markdown
  const mdIndicators = ["# ", "## ", "### ", "- ", "* ", "```"];
  if (mdIndicators.some((indicator) => contentStr.includes(indicator))) {
    return "markdown";
  }

  // Default to plain text
  return "text";
}

export function formatResultContent(content: any): string {
  if (!content) {
    return "*(无输出)*\n\n";
  }

  let contentStr: string;

  // Check if content is a list with "type": "text" structure
  try {
    let parsedContent: any;
    if (typeof content === "string") {
      parsedContent = JSON.parse(content);
    } else {
      parsedContent = content;
    }

    if (
      Array.isArray(parsedContent) &&
      parsedContent.length > 0 &&
      typeof parsedContent[0] === "object" &&
      parsedContent[0]?.type === "text"
    ) {
      // Extract the text field from the first item
      contentStr = parsedContent[0]?.text || "";
    } else {
      contentStr = String(content).trim();
    }
  } catch {
    contentStr = String(content).trim();
  }

  // 长内容用头尾截断：失败时 tail 通常是最关键的信息（栈底 / 报错原因）。
  contentStr = truncateMiddle(contentStr, 3000);

  // Detect content type
  const contentType = detectContentType(contentStr);

  // Handle JSON content specially - pretty print it
  if (contentType === "json") {
    try {
      // Try to parse and pretty print JSON
      const parsed = JSON.parse(contentStr);
      contentStr = JSON.stringify(parsed, null, 2);
    } catch {
      // Keep original if parsing fails
    }
  }

  // Format with appropriate syntax highlighting
  if (
    contentType === "text" &&
    contentStr.length < 100 &&
    !contentStr.includes("\n")
  ) {
    // Short text results don't need code blocks
    return `**→** ${contentStr}\n\n`;
  }
  return `**结果:**\n\`\`\`${contentType}\n${contentStr}\n\`\`\`\n\n`;
}

// 工具专属渲染器：把高频工具的输入/输出格式化成"它本来应有的样子"，避免一律 `Parameters: {json}` 的噪声。
// 未命中专属渲染器的工具走 fallback（formatGenericTool），保持向后兼容。

export function renderErrorBlock(content: any): string {
  const text = extractResultText(content) || String(content);
  const truncated = truncateMiddle(text, 3000);
  const fence = fenceFor(truncated);
  return `❌ **错误:**\n\n${fence}\n${truncated}\n${fence}\n\n`;
}

export function renderCollapsed(
  summary: string,
  body: string,
  lang = "",
): string {
  const truncated = truncateMiddle(body, 6000);
  const fence = fenceFor(truncated);
  return `<details>\n<summary>${summary}</summary>\n\n${fence}${lang}\n${truncated}\n${fence}\n\n</details>\n\n`;
}

function formatTaskTool(toolUse: ToolUse, toolResult?: ToolResult): string {
  const input = toolUse.input || {};
  const subagentType = input.subagent_type || "general-purpose";
  const description = input.description || "(无描述)";
  const prompt = String(input.prompt || "").trim();

  let out = `### 🤖 子 Agent → \`${subagentType}\` · ${description}\n\n`;

  if (prompt) {
    out += renderCollapsed(`子 Agent 输入（${prompt.length} 字符）`, prompt);
  }

  if (toolResult) {
    if (toolResult.is_error) {
      out += renderErrorBlock(toolResult.content);
    } else {
      const text = extractResultText(toolResult.content);
      out += renderCollapsed(`子 Agent 输出（${text.length} 字符）`, text);
    }
  }

  return out;
}

function formatBashTool(toolUse: ToolUse, toolResult?: ToolResult): string {
  const input = toolUse.input || {};
  const command = String(input.command || "").trim();
  const description = input.description ? ` — ${input.description}` : "";
  const bg = input.run_in_background ? " _(后台运行)_" : "";

  let out = `### 💻 Bash${description}${bg}\n\n`;
  if (command) {
    out += `\`\`\`bash\n${command}\n\`\`\`\n\n`;
  }

  if (toolResult) {
    if (toolResult.is_error) {
      out += renderErrorBlock(toolResult.content);
    } else {
      const text = extractResultText(toolResult.content);
      if (!text) {
        out += "*(无输出)*\n\n";
      } else {
        const truncated = truncateMiddle(text, 3000);
        const fence = fenceFor(truncated);
        out += `${fence}\n${truncated}\n${fence}\n\n`;
      }
    }
  }
  return out;
}

function formatEditTool(toolUse: ToolUse, toolResult?: ToolResult): string {
  const input = toolUse.input || {};
  const path = input.file_path || "(未知路径)";
  const replaceAll = input.replace_all ? " _(replace_all)_" : "";
  const oldStr = String(input.old_string ?? "");
  const newStr = String(input.new_string ?? "");

  let out = `### ✏️ Edit \`${path}\`${replaceAll}\n\n`;
  const oldBlock = oldStr
    .split("\n")
    .map((l) => `- ${l}`)
    .join("\n");
  const newBlock = newStr
    .split("\n")
    .map((l) => `+ ${l}`)
    .join("\n");
  const diff = truncateMiddle(`${oldBlock}\n${newBlock}`, 3000);
  out += `\`\`\`diff\n${diff}\n\`\`\`\n\n`;

  // Edit 成功的 result 永远是 "The file ... has been updated"，没意义；只在 error 时输出。
  if (toolResult?.is_error) {
    out += renderErrorBlock(toolResult.content);
  }
  return out;
}

function formatWriteTool(toolUse: ToolUse, toolResult?: ToolResult): string {
  const input = toolUse.input || {};
  const path = input.file_path || "(未知路径)";
  const content = String(input.content ?? "");
  const lineCount = content === "" ? 0 : content.split("\n").length;

  let out = `### 📝 Write \`${path}\` _(${lineCount} 行)_\n\n`;
  out += renderCollapsed(`文件内容`, content);

  if (toolResult?.is_error) {
    out += renderErrorBlock(toolResult.content);
  }
  return out;
}

function formatReadTool(toolUse: ToolUse, toolResult?: ToolResult): string {
  const input = toolUse.input || {};
  const path = input.file_path || "(未知路径)";
  const range =
    input.offset != null || input.limit != null
      ? ` _(offset=${input.offset ?? 0}, limit=${input.limit ?? "∞"})_`
      : "";

  let out = `### 📖 Read \`${path}\`${range}\n\n`;

  if (toolResult) {
    if (toolResult.is_error) {
      out += renderErrorBlock(toolResult.content);
    } else {
      const text = extractResultText(toolResult.content);
      out += renderCollapsed(`文件内容（${text.length} 字符）`, text);
    }
  }
  return out;
}

function formatTodoWriteTool(toolUse: ToolUse): string {
  const input = toolUse.input || {};
  const todos = Array.isArray(input.todos) ? input.todos : [];
  if (todos.length === 0) {
    return "### 📋 待办 _(空)_\n\n";
  }

  let out = "### 📋 待办\n\n";
  for (const t of todos) {
    const status = String(t?.status || "pending");
    const text = String(t?.content || "");
    const icon =
      status === "completed" ? "✅" : status === "in_progress" ? "🔄" : "⬜";
    out += `- ${icon} ${text}\n`;
  }
  out += "\n";
  // TodoWrite 的 tool_result 一律是 "Todos updated"，丢弃。
  return out;
}

function formatGrepTool(toolUse: ToolUse, toolResult?: ToolResult): string {
  const input = toolUse.input || {};
  const pattern = input.pattern || "";
  const path = input.path ? ` 在 \`${input.path}\` 中` : "";
  const glob = input.glob ? ` _(glob: ${input.glob})_` : "";

  let out = `### 🔍 Grep \`${pattern}\`${path}${glob}\n\n`;
  if (toolResult) {
    if (toolResult.is_error) {
      out += renderErrorBlock(toolResult.content);
    } else {
      const text = extractResultText(toolResult.content);
      const lineCount = text ? text.split("\n").length : 0;
      out += renderCollapsed(`结果（${lineCount} 行）`, text);
    }
  }
  return out;
}

function formatGlobTool(toolUse: ToolUse, toolResult?: ToolResult): string {
  const input = toolUse.input || {};
  const pattern = input.pattern || "";
  const path = input.path ? ` 在 \`${input.path}\` 中` : "";

  let out = `### 🌐 Glob \`${pattern}\`${path}\n\n`;
  if (toolResult) {
    if (toolResult.is_error) {
      out += renderErrorBlock(toolResult.content);
    } else {
      const text = extractResultText(toolResult.content);
      const lineCount = text ? text.split("\n").length : 0;
      out += renderCollapsed(`匹配（${lineCount} 个文件）`, text);
    }
  }
  return out;
}

function formatWebFetchTool(toolUse: ToolUse, toolResult?: ToolResult): string {
  const input = toolUse.input || {};
  const url = input.url || "";
  const prompt = input.prompt ? ` — ${input.prompt}` : "";

  let out = `### 🌍 WebFetch [${url}](${url})${prompt}\n\n`;
  if (toolResult) {
    if (toolResult.is_error) {
      out += renderErrorBlock(toolResult.content);
    } else {
      const text = extractResultText(toolResult.content);
      out += renderCollapsed(`抓取内容（${text.length} 字符）`, text);
    }
  }
  return out;
}

function formatWebSearchTool(
  toolUse: ToolUse,
  toolResult?: ToolResult,
): string {
  const input = toolUse.input || {};
  const query = input.query || "";

  let out = `### 🔎 WebSearch \`${query}\`\n\n`;
  if (toolResult) {
    if (toolResult.is_error) {
      out += renderErrorBlock(toolResult.content);
    } else {
      const text = extractResultText(toolResult.content);
      out += renderCollapsed(`搜索结果（${text.length} 字符）`, text);
    }
  }
  return out;
}

function formatSkillTool(toolUse: ToolUse, toolResult?: ToolResult): string {
  const input = toolUse.input || {};
  const skill = input.skill || "(未知)";
  const args = input.args ? ` — ${input.args}` : "";

  let out = `### 🎯 Skill \`${skill}\`${args}\n\n`;
  if (toolResult) {
    if (toolResult.is_error) {
      out += renderErrorBlock(toolResult.content);
    } else {
      const text = extractResultText(toolResult.content);
      if (text) {
        out += renderCollapsed(`Skill 输出（${text.length} 字符）`, text);
      }
    }
  }
  return out;
}

function formatGenericTool(toolUse: ToolUse, toolResult?: ToolResult): string {
  const toolName = toolUse.name || "unknown_tool";
  const toolInput = toolUse.input || {};

  let result = `### 🔧 \`${toolName}\`\n\n`;

  if (Object.keys(toolInput).length > 0) {
    result += "**参数:**\n```json\n";
    result += JSON.stringify(toolInput, null, 2);
    result += "\n```\n\n";
  }

  if (toolResult) {
    if (toolResult.is_error) {
      result += renderErrorBlock(toolResult.content);
    } else {
      result += formatResultContent(toolResult.content);
    }
  }
  return result;
}

export function formatToolWithResult(
  toolUse: ToolUse,
  toolResult?: ToolResult,
): string {
  switch (toolUse.name) {
    case "Task":
      return formatTaskTool(toolUse, toolResult);
    case "Bash":
      return formatBashTool(toolUse, toolResult);
    case "Edit":
      return formatEditTool(toolUse, toolResult);
    case "Write":
      return formatWriteTool(toolUse, toolResult);
    case "Read":
      return formatReadTool(toolUse, toolResult);
    case "TodoWrite":
      return formatTodoWriteTool(toolUse);
    case "Grep":
      return formatGrepTool(toolUse, toolResult);
    case "Glob":
      return formatGlobTool(toolUse, toolResult);
    case "WebFetch":
      return formatWebFetchTool(toolUse, toolResult);
    case "WebSearch":
      return formatWebSearchTool(toolUse, toolResult);
    case "Skill":
      return formatSkillTool(toolUse, toolResult);
    default:
      return formatGenericTool(toolUse, toolResult);
  }
}

export function groupTurnsNaturally(data: Turn[]): GroupedContent[] {
  const groupedContent: GroupedContent[] = [];
  const toolResultsMap = new Map<string, ToolResult>();

  // First pass: collect all tool results by tool_use_id
  for (const turn of data) {
    if (turn.type === "user") {
      const content = turn.message?.content || [];
      for (const item of content) {
        if (item.type === "tool_result" && item.tool_use_id) {
          toolResultsMap.set(item.tool_use_id, {
            type: item.type,
            tool_use_id: item.tool_use_id,
            content: item.content,
            is_error: item.is_error,
          });
        }
      }
    }
  }

  // Second pass: process turns and group naturally
  for (const turn of data) {
    const turnType = turn.type || "unknown";

    if (turnType === "system") {
      // 只渲染 init（带可用工具数）。其它 system 子类型是 CLI 内部遥测
      // （如 thinking_tokens），对人类可读的运行轨迹无意义——原样 JSON 倒进
      // Step Summary 只是噪声，直接丢弃。
      if ((turn.subtype || "") === "init") {
        const tools = turn.tools || [];
        groupedContent.push({
          type: "system_init",
          tools_count: tools.length,
        });
      }
    } else if (turnType === "assistant") {
      const message = turn.message || { content: [] };
      const content = message.content || [];
      const usage = message.usage || {};

      // Process content items
      const textParts: string[] = [];
      const toolCalls: { tool_use: ToolUse; tool_result?: ToolResult }[] = [];

      for (const item of content) {
        const itemType = item.type || "";

        if (itemType === "text") {
          textParts.push(item.text || "");
        } else if (itemType === "tool_use") {
          const toolUseId = item.id;
          const toolResult = toolUseId
            ? toolResultsMap.get(toolUseId)
            : undefined;
          toolCalls.push({
            tool_use: {
              type: item.type,
              name: item.name,
              input: item.input,
              id: item.id,
            },
            tool_result: toolResult,
          });
        }
      }

      if (textParts.length > 0 || toolCalls.length > 0) {
        groupedContent.push({
          type: "assistant_action",
          text_parts: textParts,
          tool_calls: toolCalls,
          usage: usage,
        });
      }
    } else if (turnType === "user") {
      // Handle user messages that aren't tool results
      const message = turn.message || { content: [] };
      const content = message.content || [];
      const textParts: string[] = [];

      for (const item of content) {
        if (item.type === "text") {
          textParts.push(item.text || "");
        }
      }

      if (textParts.length > 0) {
        groupedContent.push({
          type: "user_message",
          text_parts: textParts,
        });
      }
    } else if (turnType === "result") {
      groupedContent.push({
        type: "final_result",
        data: turn,
      });
    }
  }

  return groupedContent;
}

export function formatGroupedContent(groupedContent: GroupedContent[]): string {
  let markdown = "## Claude Code 执行轨迹\n\n";

  for (const item of groupedContent) {
    const itemType = item.type;

    if (itemType === "system_init") {
      markdown += `## 🚀 系统初始化\n\n**可用工具:** ${item.tools_count} 个\n\n---\n\n`;
    } else if (itemType === "assistant_action") {
      // Add text content first (if any) - no header needed
      for (const text of item.text_parts || []) {
        if (text.trim()) {
          markdown += `${text}\n\n`;
        }
      }

      // Add tool calls with their results
      for (const toolCall of item.tool_calls || []) {
        markdown += formatToolWithResult(
          toolCall.tool_use,
          toolCall.tool_result,
        );
      }

      // Add usage info if available
      const usage = item.usage || {};
      if (Object.keys(usage).length > 0) {
        const inputTokens = usage.input_tokens || 0;
        const cacheCreationTokens = usage.cache_creation_input_tokens || 0;
        const cacheReadTokens = usage.cache_read_input_tokens || 0;
        const totalInputTokens =
          inputTokens + cacheCreationTokens + cacheReadTokens;
        const outputTokens = usage.output_tokens || 0;
        markdown += `*Token 用量: ${totalInputTokens} 输入, ${outputTokens} 输出*\n\n`;
      }

      // Only add separator if this section had content
      if (
        (item.text_parts && item.text_parts.length > 0) ||
        (item.tool_calls && item.tool_calls.length > 0)
      ) {
        markdown += "---\n\n";
      }
    } else if (itemType === "user_message") {
      markdown += "## 👤 用户\n\n";
      for (const text of item.text_parts || []) {
        if (text.trim()) {
          markdown += `${text}\n\n`;
        }
      }
      markdown += "---\n\n";
    } else if (itemType === "final_result") {
      const data = item.data || {};
      const cost = (data as any).total_cost_usd || (data as any).cost_usd || 0;
      const duration = (data as any).duration_ms || 0;
      const resultText = (data as any).result || "";

      markdown += "## ✅ 最终结果\n\n";
      if (resultText) {
        markdown += `${resultText}\n\n`;
      }
      markdown += `**Cost:** $${cost.toFixed(4)} | **Duration:** ${(duration / 1000).toFixed(1)}s\n\n`;
    }
  }

  return markdown;
}

export function formatTurnsFromData(data: Turn[]): string {
  // Group turns naturally
  const groupedContent = groupTurnsNaturally(data);

  // Generate markdown
  const markdown = formatGroupedContent(groupedContent);

  return markdown;
}

function main(): void {
  // Get the JSON file path from command line arguments
  const args = process.argv.slice(2);
  if (args.length === 0) {
    console.error("用法: format-turns.ts <json 文件>");
    exit(1);
  }

  const jsonFile = args[0];
  if (!jsonFile) {
    console.error("错误: 未提供 JSON 文件");
    exit(1);
  }

  if (!existsSync(jsonFile)) {
    console.error(`错误: 找不到文件 ${jsonFile}`);
    exit(1);
  }

  try {
    // Read the JSON file
    const fileContent = readFileSync(jsonFile, "utf-8");
    const data: Turn[] = JSON.parse(fileContent);

    // Group turns naturally
    const groupedContent = groupTurnsNaturally(data);

    // Generate markdown
    const markdown = formatGroupedContent(groupedContent);

    // Print to stdout (so it can be captured by shell)
    console.log(markdown);
  } catch (error) {
    console.error(`处理文件出错: ${error}`);
    exit(1);
  }
}

if (import.meta.main) {
  main();
}
