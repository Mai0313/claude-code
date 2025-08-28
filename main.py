from typing import Union, Literal, Annotated
import getpass
from pathlib import Path

import orjsonl
from pydantic import Field, BaseModel, ValidationError
import machineid
from rich.console import Console

# ============================================================================
# Claude Code Analysis Models - 用於代碼分析統計
# ============================================================================


class ClaudeCodeAnalysisDetail(BaseModel):
    """代碼分析的詳細資訊，包含檔案路徑、字符數等"""

    filePath: str
    characterCount: int
    timestamp: int
    aiOutputContent: str = ""
    fileContent: str = ""


class ClaudeCodeAnalysisToolCalls(BaseModel):
    """工具調用次數統計"""

    Read: int = 0
    Write: int = 0
    Edit: int = 0
    TodoWrite: int = 0
    Bash: int = 0


class ClaudeCodeAnalysisRecord(BaseModel):
    """單次分析任務的完整記錄"""

    totalUniqueFiles: int
    totalWriteLines: int
    totalReadCharacters: int
    totalWriteCharacters: int
    totalDiffCharacters: int
    writeToFileDetails: list[ClaudeCodeAnalysisDetail]
    readFileDetails: list[ClaudeCodeAnalysisDetail]
    applyDiffDetails: list[ClaudeCodeAnalysisDetail]
    toolCallCounts: ClaudeCodeAnalysisToolCalls
    taskId: str
    timestamp: int
    folderPath: str
    gitRemoteUrl: str


class ClaudeCodeAnalysis(BaseModel):
    """Claude Code 分析的主要資料結構"""

    user: str = getpass.getuser()
    extensionName: str = "Claude-Code"
    # 這裡應該透過跟go 一樣的方式來取得版本 而不是寫死 但測試可以先寫死
    insightsVersion: str = "0.1.0"
    machineId: str = machineid.id()
    records: list[ClaudeCodeAnalysisRecord] = []


# ============================================================================
# Claude Code Log Models - 用於解析 JSONL 對話記錄
# ============================================================================


class ClaudeCodeLogContentInputBash(BaseModel):
    command: str
    description: str


class ClaudeCodeLogContentInputEdit(BaseModel):
    file_path: str
    old_string: str
    new_string: str


class ClaudeCodeLogContentInputRead(BaseModel):
    file_path: str


class ClaudeCodeLogContentInputTodoWriteItem(BaseModel):
    content: str
    status: Literal["in_progress", "pending", "completed"]
    activeForm: str


class ClaudeCodeLogContentInputTodoWrite(BaseModel):
    todos: list[ClaudeCodeLogContentInputTodoWriteItem]


class ClaudeCodeLogContentInputWrite(BaseModel):
    file_path: str
    content: str


ClaudeCodeLogToolInput = Union[
    ClaudeCodeLogContentInputTodoWrite,
    ClaudeCodeLogContentInputRead,
    ClaudeCodeLogContentInputEdit,
    ClaudeCodeLogContentInputBash,
    ClaudeCodeLogContentInputWrite,
]


class ClaudeCodeLogContentToolUse(BaseModel):
    type: Literal["tool_use"]
    name: Literal["TodoWrite", "Read", "Edit", "Bash", "Write"]  # 嚴格限制已知工具類型
    id: str
    input: ClaudeCodeLogToolInput


class ClaudeCodeLogContentToolResult(BaseModel):
    type: Literal["tool_result"]
    tool_use_id: str
    content: str


class ClaudeCodeLogContentText(BaseModel):
    type: Literal["text"]
    text: str


ClaudeCodeLogContent = Annotated[
    ClaudeCodeLogContentText | ClaudeCodeLogContentToolUse | ClaudeCodeLogContentToolResult,
    Field(discriminator="type"),
]


class ClaudeCodeLogMessageUsage(BaseModel):
    input_tokens: int
    cache_creation_input_tokens: int
    cache_read_input_tokens: int
    output_tokens: int


# ============================================================================
# Tool Use Result Models - 不同工具的執行結果
# ============================================================================


class ClaudeCodeLogToolUseResultTodo(BaseModel):
    oldTodos: list[ClaudeCodeLogContentInputTodoWriteItem]
    newTodos: list[ClaudeCodeLogContentInputTodoWriteItem]


class ClaudeCodeLogToolUseResultCreate(BaseModel):
    type: Literal["create"]
    filePath: str
    content: str
    structuredPatch: list


class ClaudeCodeLogToolUseResultFile(BaseModel):
    filePath: str
    content: str
    numLines: int
    startLine: int
    totalLines: int


class ClaudeCodeLogToolUseResultRead(BaseModel):
    type: Literal["text"]
    file: ClaudeCodeLogToolUseResultFile


class ClaudeCodeLogToolUseResultBash(BaseModel):
    stdout: str
    stderr: str
    interrupted: bool
    isImage: bool


class ClaudeCodeLogToolUseResultEditPatch(BaseModel):
    oldStart: int
    oldLines: int
    newStart: int
    newLines: int
    lines: list[str]


class ClaudeCodeLogToolUseResultEdit(BaseModel):
    filePath: str
    oldString: str
    newString: str
    originalFile: str
    structuredPatch: list[ClaudeCodeLogToolUseResultEditPatch]
    userModified: bool
    replaceAll: bool


ClaudeCodeLogToolUseResult = Union[
    ClaudeCodeLogToolUseResultTodo,
    ClaudeCodeLogToolUseResultCreate,
    ClaudeCodeLogToolUseResultRead,
    ClaudeCodeLogToolUseResultBash,
    ClaudeCodeLogToolUseResultEdit,
]


# ============================================================================
# Message Models - 使用者和助手的訊息結構
# ============================================================================


class ClaudeCodeLogUserMessage(BaseModel):
    role: Literal["user"]
    content: str | list[ClaudeCodeLogContent]


class ClaudeCodeLogAssistantMessage(BaseModel):
    id: str
    type: Literal["message"]
    role: Literal["assistant"]
    model: str
    content: list[ClaudeCodeLogContent]
    stop_reason: str | None = None
    stop_sequence: str | None = None
    usage: ClaudeCodeLogMessageUsage


ClaudeCodeLogMessage = Annotated[
    ClaudeCodeLogUserMessage | ClaudeCodeLogAssistantMessage, Field(discriminator="role")
]


class ClaudeCodeLog(BaseModel):
    parentUuid: str | None
    isSidechain: bool
    userType: str
    cwd: str
    sessionId: str
    version: str
    gitBranch: str
    type: Literal["user", "assistant"]
    uuid: str
    timestamp: str
    message: ClaudeCodeLogMessage
    toolUseResult: ClaudeCodeLogToolUseResult | None = None


console = Console()


def analyze_conversations() -> None:
    conversation_folder = Path("./examples/logs")
    conversation_paths = conversation_folder.rglob("*.jsonl")
    for conversation_path in conversation_paths:
        conversations = orjsonl.load(conversation_path)
        for conversation in conversations:
            try:
                claude_code_log = ClaudeCodeLog(**conversation)
                console.print(claude_code_log)
            except ValidationError as e:
                console.print(e)
                console.print(conversation_path.as_posix())
                break  # 遇到錯誤就停止，方便調試


if __name__ == "__main__":
    analyze_conversations()
