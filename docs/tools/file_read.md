# Built-in Tool: file_read

## 1. 功能摘要 (Summary)
`file_read` 工具用于读取指定文件的全部内容，并将其作为字符串返回。

## 2. MCP Schema
这是提供给语言模型（LLM）的工具接口定义。

```json
{
  "name": "file_read",
  "description": "Reads the entire content of a specified file.",
  "parameters": {
    "type": "object",
    "properties": {
      "path": {
        "type": "string",
        "description": "The path to the file to read."
      }
    },
    "required": ["path"]
  }
}
```

## 3. 参数详解 (Parameters)

| 参数名 (Name) | 类型 (Type) | 描述 (Description) | 是否必须 (Required) |
| :---------- | :-------- | :----------------- | :---------------- |
| `path`      | `string`  | 要读取的文件的路径。   | 是 (Yes)          |

## 4. 返回结果 (Returns)

- **成功 (Success)**: 返回文件的完整内容，以 UTF-8 字符串的形式。
- **失败 (Failure)**: 返回一个描述错误的字符串，例如：
    - 如果文件不存在: `"failed to read file a/b/c.txt: open a/b/c.txt: no such file or directory"`
    - 如果路径不是一个文件 (例如是一个目录): `"failed to read file /path/to/dir: read /path/to/dir: is a directory"`

## 5. 使用示例 (Example)

**场景**: 用户希望查看 `config.yaml` 文件的内容。

1.  **用户 Prompt**:
    `"Please read the content of the 'config.yaml' file and tell me what's inside."`

2.  **LLM 生成的 `tool_call`**:
    ```json
    {
      "name": "file_read",
      "arguments": {
        "path": "config.yaml"
      }
    }
    ```

3.  **`file_read` 工具的输出 (返回给 LLM)**:
    ```
    llm:
      provider: "generic-http"
      model_name: "local-mock-model"
      endpoint: "http://localhost:8080/v1/mcp-generate"
    ...
    ```

## 6. 实现细节 (Implementation Notes)
- 该工具在 Go 中通过 `os.ReadFile` 实现，这是一个安全且标准的文件读取方法。
- 返回内容的大小受限于 agent 的上下文窗口和内存限制。对于非常大的文件，Agent 可能只能处理部分内容。
