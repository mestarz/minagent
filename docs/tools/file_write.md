# Built-in Tool: file_write

## 1. 功能摘要 (Summary)
`file_write` 工具用于将指定的内容写入一个文件。如果文件不存在，它将被创建；如果文件已存在，其内容将被完全覆盖。

## 2. MCP Schema
这是提供给语言模型（LLM）的工具接口定义。

```json
{
  "name": "file_write",
  "description": "Writes content to a specified file, creating it if it doesn't exist, or overwriting it if it does.",
  "parameters": {
    "type": "object",
    "properties": {
      "path": {
        "type": "string",
        "description": "The path to the file to write to."
      },
      "content": {
        "type": "string",
        "description": "The content to write into the file."
      }
    },
    "required": ["path", "content"]
  }
}
```

## 3. 参数详解 (Parameters)

| 参数名 (Name) | 类型 (Type) | 描述 (Description) | 是否必须 (Required) |
| :---------- | :-------- | :------------------- | :---------------- |
| `path`      | `string`  | 要写入的文件的路径。     | 是 (Yes)          |
| `content`   | `string`  | 要写入文件的文本内容。   | 是 (Yes)          |

## 4. 返回结果 (Returns)

- **成功 (Success)**: 返回一个确认消息，包含写入的字节数，例如：`"Successfully wrote 56 bytes to src/main.go"`。
- **失败 (Failure)**: 返回一个描述错误的字符串，例如：
    - 如果路径是一个目录: `"failed to write to file /path/to/dir: write /path/to/dir: is a directory"`
    - 如果没有写入权限: `"failed to write to file /path/to/protected/file: open /path/to/protected/file: permission denied"`

## 5. 使用示例 (Example)

**场景**: 用户希望创建一个新的 Go 文件。

1.  **用户 Prompt**:
    `"Create a new file named 'main.go' and write a simple hello world program into it."`

2.  **LLM 生成的 `tool_call`**:
    ```json
    {
      "name": "file_write",
      "arguments": {
        "path": "main.go",
        "content": "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}\n"
      }
    }
    ```

3.  **`file_write` 工具的输出 (返回给 LLM)**:
    ```
    Successfully wrote 74 bytes to main.go
    ```

## 6. 实现细节 (Implementation Notes)
- 该工具在 Go 中通过 `os.WriteFile` 实现，权限模式默认为 `0644`。
- 这是一个破坏性操作，因为它会覆盖已存在的文件。LLM 在调用此工具前应通过 `file_read` 或 `list_directory` 确认文件状态，或者得到用户的明确指令。

```