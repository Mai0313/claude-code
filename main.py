import json
from typing import Literal, Annotated, TypeAlias
import getpass
from pathlib import Path
from datetime import datetime, timezone

import orjsonl
from pydantic import Field, BaseModel, ValidationError
import machineid
from rich.console import Console

# ============================================================================
# Claude Code Analysis Models - data used for analysis stats
# ============================================================================


class ClaudeCodeAnalysisDetailBase(BaseModel):
    """Base detail model with shared required fields."""

    filePath: str
    lineCount: int
    characterCount: int
    timestamp: int


class ClaudeCodeAnalysisWriteDetail(ClaudeCodeAnalysisDetailBase):
    """writeToFileDetails: also stores the full content."""

    content: str = ""


class ClaudeCodeAnalysisReadDetail(ClaudeCodeAnalysisDetailBase):
    """readFileDetails: only the required fields."""

    pass


class ClaudeCodeAnalysisApplyDiffDetail(ClaudeCodeAnalysisDetailBase):
    """applyDiffDetails: keep old_string/new_string as provided in input."""

    old_string: str = ""
    new_string: str = ""


class ClaudeCodeAnalysisRunCommandDetail(ClaudeCodeAnalysisDetailBase):
    """runCommandDetails: stores the bash command and description."""

    command: str = ""
    description: str = ""


class ClaudeCodeAnalysisToolCalls(BaseModel):
    """Counters for how many times each tool was invoked."""

    Read: int = 0
    Write: int = 0
    Edit: int = 0
    TodoWrite: int = 0
    Bash: int = 0


class ClaudeCodeAnalysisRecord(BaseModel):
    """Aggregated stats for a single analysis session."""

    totalUniqueFiles: int
    totalWriteLines: int
    totalReadCharacters: int
    totalWriteCharacters: int
    totalDiffCharacters: int
    writeToFileDetails: list[ClaudeCodeAnalysisWriteDetail]
    readFileDetails: list[ClaudeCodeAnalysisReadDetail]
    applyDiffDetails: list[ClaudeCodeAnalysisApplyDiffDetail]
    runCommandDetails: list[ClaudeCodeAnalysisRunCommandDetail]
    toolCallCounts: ClaudeCodeAnalysisToolCalls
    taskId: str
    timestamp: int
    folderPath: str
    gitRemoteUrl: str


class ClaudeCodeAnalysis(BaseModel):
    """Top-level analysis payload produced by this script."""

    user: str = getpass.getuser()
    extensionName: str = "Claude-Code"
    # We should pull this version the same way the Go tool does.
    # Hardcoded for now to keep the sample simple.
    insightsVersion: str = "0.1.0"
    machineId: str = machineid.id()
    records: list[ClaudeCodeAnalysisRecord] = []


# ============================================================================
# Claude Code Log Models - parse JSONL conversation records
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


ClaudeCodeLogToolInput: TypeAlias = (
    ClaudeCodeLogContentInputTodoWrite
    | ClaudeCodeLogContentInputRead
    | ClaudeCodeLogContentInputEdit
    | ClaudeCodeLogContentInputBash
    | ClaudeCodeLogContentInputWrite
)


class ClaudeCodeLogContentToolUse(BaseModel):
    type: Literal["tool_use"]
    name: Literal["TodoWrite", "Read", "Edit", "Bash", "Write"]
    id: str
    input: ClaudeCodeLogToolInput


class ClaudeCodeLogContentToolResult(BaseModel):
    type: Literal["tool_result"]
    tool_use_id: str
    content: str


class ClaudeCodeLogContentText(BaseModel):
    type: Literal["text"]
    text: str


ClaudeCodeLogContent: TypeAlias = Annotated[
    ClaudeCodeLogContentText | ClaudeCodeLogContentToolUse | ClaudeCodeLogContentToolResult,
    Field(discriminator="type"),
]


class ClaudeCodeLogMessageUsage(BaseModel):
    input_tokens: int
    cache_creation_input_tokens: int
    cache_read_input_tokens: int
    output_tokens: int


# ============================================================================
# Tool Use Result Models - outputs returned by tools
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


ClaudeCodeLogToolUseResult: TypeAlias = (
    ClaudeCodeLogToolUseResultTodo
    | ClaudeCodeLogToolUseResultCreate
    | ClaudeCodeLogToolUseResultRead
    | ClaudeCodeLogToolUseResultBash
    | ClaudeCodeLogToolUseResultEdit
)


# ============================================================================
# Message Models - user and assistant message shapes
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


ClaudeCodeLogMessage: TypeAlias = Annotated[
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
    conversation_path = Path("./examples/test_conversation.jsonl")
    output_path = Path(f"./examples/parsed/{conversation_path.stem}_parsed.json")
    output_path.parent.mkdir(parents=True, exist_ok=True)

    conversations = orjsonl.load(conversation_path)

    # Accumulators for all records we will emit
    write_details: list[ClaudeCodeAnalysisWriteDetail] = []
    read_details: list[ClaudeCodeAnalysisReadDetail] = []
    apply_diff_details: list[ClaudeCodeAnalysisApplyDiffDetail] = []
    run_details: list[ClaudeCodeAnalysisRunCommandDetail] = []

    tool_counts = ClaudeCodeAnalysisToolCalls()
    unique_files: set[str] = set()

    total_write_lines = 0
    total_read_characters = 0
    total_write_characters = 0
    total_diff_characters = 0

    folder_path = ""
    git_remote_url = ""  # Not available in this sample; leave empty
    task_id = ""
    last_ts_int = 0

    def parse_ts(ts: str) -> int:
        # Example: convert 2025-08-28T12:57:19.002Z to epoch seconds
        try:
            dt = datetime.strptime(ts, "%Y-%m-%dT%H:%M:%S.%fZ").replace(tzinfo=timezone.utc)
            return int(dt.timestamp())
        except Exception:
            try:
                dt = datetime.fromisoformat(ts)
                if dt.tzinfo is None:
                    dt = dt.replace(tzinfo=timezone.utc)
                return int(dt.timestamp())
            except Exception:
                return 0

    for conversation in conversations:
        try:
            claude_code_log = ClaudeCodeLog(**conversation)
        except ValidationError:
            # Skip entries that don't fit the model (e.g., thinking blocks)
            continue

        if not folder_path:
            folder_path = claude_code_log.cwd
        task_id = claude_code_log.sessionId

        ts_int = parse_ts(claude_code_log.timestamp)
        last_ts_int = ts_int or last_ts_int

        # Count tool invocations (assistant tool_use only)
        if isinstance(claude_code_log.message, ClaudeCodeLogAssistantMessage):
            for item in claude_code_log.message.content:
                if isinstance(item, ClaudeCodeLogContentToolUse):
                    if item.name == "Read":
                        tool_counts.Read += 1
                    elif item.name == "Write":
                        tool_counts.Write += 1
                    elif item.name == "Edit":
                        tool_counts.Edit += 1
                    elif item.name == "TodoWrite":
                        tool_counts.TodoWrite += 1
                    elif item.name == "Bash":
                        tool_counts.Bash += 1
                        # Record runCommandDetails from the input (no file; use cwd as filePath)
                        bash_input = item.input
                        if isinstance(bash_input, ClaudeCodeLogContentInputBash):
                            run_details.append(
                                ClaudeCodeAnalysisRunCommandDetail(
                                    filePath=claude_code_log.cwd,
                                    lineCount=0,
                                    characterCount=len(bash_input.command or ""),
                                    timestamp=ts_int,
                                    command=bash_input.command,
                                    description=bash_input.description,
                                )
                            )

        # Fill the various *Details from toolUseResult
        tur = claude_code_log.toolUseResult
        if tur is None:
            continue

        # Read result
        if isinstance(tur, ClaudeCodeLogToolUseResultRead):
            file_path = tur.file.filePath
            content = tur.file.content or ""
            num_lines = tur.file.numLines

            read_details.append(
                ClaudeCodeAnalysisReadDetail(
                    filePath=file_path,
                    lineCount=num_lines,
                    characterCount=len(content),
                    timestamp=ts_int,
                )
            )
            unique_files.add(file_path)
            total_read_characters += len(content)

        # Write (create) result
        elif isinstance(tur, ClaudeCodeLogToolUseResultCreate):
            file_path = tur.filePath
            content = tur.content or ""
            line_count = len(content.splitlines())

            write_details.append(
                ClaudeCodeAnalysisWriteDetail(
                    filePath=file_path,
                    lineCount=line_count,
                    characterCount=len(content),
                    timestamp=ts_int,
                    content=content,
                )
            )
            unique_files.add(file_path)
            total_write_lines += line_count
            total_write_characters += len(content)

        # Edit result (applyDiff)
        elif isinstance(tur, ClaudeCodeLogToolUseResultEdit):
            file_path = tur.filePath
            new_s = tur.newString or ""
            old_s = tur.oldString or ""
            line_count = len(new_s.splitlines())

            apply_diff_details.append(
                ClaudeCodeAnalysisApplyDiffDetail(
                    filePath=file_path,
                    lineCount=line_count,
                    characterCount=len(new_s),
                    timestamp=ts_int,
                    old_string=old_s,
                    new_string=new_s,
                )
            )
            unique_files.add(file_path)
            total_diff_characters += len(new_s)

    record = ClaudeCodeAnalysisRecord(
        totalUniqueFiles=len(unique_files),
        totalWriteLines=total_write_lines,
        totalReadCharacters=total_read_characters,
        totalWriteCharacters=total_write_characters,
        totalDiffCharacters=total_diff_characters,
        writeToFileDetails=write_details,
        readFileDetails=read_details,
        applyDiffDetails=apply_diff_details,
        runCommandDetails=run_details,
        toolCallCounts=tool_counts,
        taskId=task_id,
        timestamp=last_ts_int,
        folderPath=folder_path,
        gitRemoteUrl=git_remote_url,
    )

    analysis = ClaudeCodeAnalysis(records=[record])

    with output_path.open("w", encoding="utf-8") as f:
        json.dump(analysis.model_dump(mode="json"), f, ensure_ascii=False, indent=4)


if __name__ == "__main__":
    analyze_conversations()
