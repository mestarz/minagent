# Built-in Tool: list_directory

## 1. 功能摘要 (Summary)
`list_directory` 工具用于列出指定目录下的文件和子目录。

## 2. MCP Schema
这是提供给语言模型（LLM）的工具接口定义。

```json
{
  "name": "list_directory",
  "description": "Lists the contents (files and subdirectories) of a specified directory.",
  "parameters": {
    "type": "object",
    "properties": {
      "path": {
        "type": "string",
        "description": "The path to the directory to list. Defaults to the current directory '.'."
      },
      "recursive": {
        "type": "boolean",
        "description": "Whether to list contents recursively. Defaults to false.",
        "default": false
      }
    },
    "required": ["path"]
  }
}
```

## 3. 参数详解 (Parameters)

| 参数名 (Name) | 类型 (Type) | 描述 (Description) | 是否必须 (Required) |
| :---------- | :-------- | :--------------------- | :---------------- |
| `path`      | `string`  | 要列出内容的目录路径。   | 是 (Yes)          |
| `recursive` | `boolean` | 是否递归地列出所有子目录的内容。默认为 `false`。 | 否 (No)           |

## 4. 返回结果 (Returns)

- **成功 (Success)**: 返回一个格式化的字符串，列出目录下的条目。每个条目占一行，目录以 `/` 结尾。
    - **非递归 (Non-Recursive)**:
      ```
      Contents of directory 'src/':
      main.go
      utils/
      config.go
      ```
    - **递归 (Recursive)**:
      ```
      Contents of directory 'src/':
      main.go
      utils/
      utils/helpers.go
      config.go
      ```
- **失败 (Failure)**: 返回一个描述错误的字符串，例如：
    - 如果目录不存在: `"failed to read directory /path/to/nonexistent: open /path/to/nonexistent: no such file or directory"`
    - 如果路径是一个文件: `"failed to read directory /path/to/file.txt: readdirent /path/to/file.txt: not a directory"`

## 5. 使用示例 (Example)

**场景**: 用户想了解当前项目结构。

1.  **用户 Prompt**:
    `"Can you show me all the files in the current directory, including subdirectories?"`

2.  **LLM 生成的 `tool_call`**:
    ```json
    {
      "name": "list_directory",
      "arguments": {
        "path": ".",
        "recursive": true
      }
    }
    ```

3.  **`list_directory` 工具的输出 (返回给 LLM)**:
    ```
    Contents of directory '.':
    README.md
    go.mod
    go.sum
    cmd/
    cmd/minagent/
    cmd/minagent/main.go
    internal/
    ...
    ```

## 6. 实现细节 (Implementation Notes)
- **非递归模式**: 内部使用 `os.ReadDir`。
- **递归模式**: 内部使用 `filepath.WalkDir`，提供相对于所查路径的相对路径，以保持输出的简洁性。
