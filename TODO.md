請幫我繼續完成 main.py

這腳本是一個範例腳本 不需要太複雜
這腳本會將 tests/test_conversation.jsonl parse ​成 examples/claude_code_log.json 這種格式
請你將此次更新著重在輸出文件裡的 records 裡面的 *Details

將 newContent oldContent 要刪除, 因為我打算將這些資訊分為 writeToFileDetails, readFileDetails, applyDiffDetails, etc...

下面所有欄位都會包含 filePath, lineCount, characterCount 和 timestamp 這三個必填欄位

writeToFileDetails: 除了必填欄位以外 要存放 ClaudeCodeLogContentInputWrite 的 content, 也要考慮到 ClaudeCodeLogToolUseResultCreate 這些資訊

readFileDetails: 這裡只需要必填欄位即可, 也要考慮到 ClaudeCodeLogToolUseResultRead

applyDiffDetails: 要存放 ClaudeCodeLogContentInputEdit 的 old_string 和 new_string 也要考慮到 ClaudeCodeLogToolUseResultEdit

runCommandDetails: 要存放 ClaudeCodeLogContentInputBash 的 command 和 description 也要考慮到 ClaudeCodeLogToolUseResultBash

有一些資訊可能要從 ClaudeCodeLogToolUseResult 裡面取得
