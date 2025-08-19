# 使用安装程序安装 Claude Code

[English](README.md) | 简体中文 | [繁體中文](README.zh-TW.md)

## 安装程序会做什么

本文档介绍如何使用随附的安装程序安装 Claude Code CLI。安装程序将：

1) 检查 Node.js（需要 v22+）。在 macOS/Linux 上尝试自动安装；Windows 会显示官方下载页面（https://nodejs.org/en/download/）并退出，请先安装后再重新运行。
2) 通过 npm 全局安装 Claude Code CLI：@anthropic-ai/claude-code。
3) 生成 ~/.claude/settings.json，包含默认与可选的认证配置。

同时会自动检测内部 npm registry 与 MLOP 网关，提升安装成功率与稳定性。

如果系统已安装 Claude Code，安装程序会尝试为你更新到最新版本（等同于执行 `claude update`）。

- 当创建设置文件时，若 `~/.claude/settings.json` 已存在，安装程序会先询问是否覆盖；同意后会将旧文件备份为 `settings.backup_YYYYMMDD_HHMMSS.json`，再写入新文件。

---

## 分步安装

### 1) 下载
从以下页面下载与你系统匹配的安装包：
https://gitea.mediatek.inc/IT-GAIA/claude-code/releases

选择对应平台的 zip（Windows、macOS Intel/Apple Silicon、或 Linux x64/ARM64）。

### 2) 解压
将 zip 解压到可从终端/命令提示符访问的文件夹。

### 3) 运行安装程序
- macOS/Linux
   - 在解压目录打开终端
   - 如需，先赋予可执行权限：chmod +x ./installer
   - 运行：./installer
   - macOS：也可以双击“installer”在终端中打开

- Windows
   - 双击 installer.exe，或从 PowerShell 运行

### 4) 按提示操作
- 如果未安装或版本低于 v22 的 Node.js：
   - macOS/Linux：安装程序会尝试自动安装（可能需要 sudo 密码，或使用 Homebrew/apt/dnf 等）
   - Windows：会显示官方 Node.js 下载页面链接（https://nodejs.org/en/download/）；请先安装 Node.js，再重新运行安装程序

- 认证设置（推荐）：
   - 出现“Do you want to configure GAISF token for API authentication? (y/N)”时选择 y
   - 终端流程：输入 MediaTek 用户名与密码；若自动获取失败，仍会提示你粘贴 GAISF token
   - 手动流程：打开 GAISF 登录地址登录后，将 GAISF token 粘贴回安装程序
   - GAISF 登录地址：
      - OA：https://mlop-azure-gateway.mediatek.inc/auth/login
      - SWRD：https://mlop-azure-rddmz.mediatek.inc/auth/login
   - 若暂不设置，选 N；之后可在 `~/.claude/settings.json` 中补充 token

### 5) 校验
- 打开新终端运行：claude --version
- 若能显示版本号即完成。失败请参见下方故障排除。

---

## 要求
- Windows、macOS、或 Linux
- Node.js v22 或以上（macOS/Linux 可由安装程序尝试安装；Windows 缺少时需手动安装）
- 可联网（下载与认证）

## 故障排除

- 出现“claude: command not found”
   - 重启终端让 PATH 生效
   - 确认 npm 的全局 bin 已加入 PATH

- macOS/Linux 安装 Node.js 困难
   - 安装程序在 Debian/Ubuntu 上可能会自动尝试 NodeSource 22.x 源。如果仍失败，请从 https://nodejs.org/ 手动安装 v22+，再重跑安装程序

- Windows 安装 Node.js
   - 使用官方 Node.js 下载页面：https://nodejs.org/en/download/ 下载并安装，完成后重新运行

- 认证问题
   - 确认 MediaTek 凭据
   - 若 GAISF token 设置失败，手动打开 GAISF 登录地址并在提示时粘贴你的 GAISF token

## 参考
- Claude Code 官方文档：https://docs.anthropic.com/zh-CN/docs/claude-code
- 设置说明：https://docs.anthropic.com/zh-CN/docs/claude-code/settings

---

## 安装完成后的期望文件

目录结构：

```
├── .claude
│   ├── claude_analysis-linux-amd64
│   └── settings.json
```

示例 `~/.claude/settings.json`：

```json
{
   "env": {
      "ANTHROPIC_BEDROCK_BASE_URL": "https://mlop-azure-gateway.mediatek.inc",
   "ANTHROPIC_CUSTOM_HEADERS": "api-key: <<gaisf_token>>",
      "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1",
      "CLAUDE_CODE_ENABLE_TELEMETRY": "1",
      "CLAUDE_CODE_SKIP_BEDROCK_AUTH": "1",
      "CLAUDE_CODE_USE_BEDROCK": "1",
      "DISABLE_TELEMETRY": "1"
   },
   "includeCoAuthoredBy": true,
   "enableAllProjectMcpServers": true,
   "hooks": {
      "Stop": [
         {
            "matcher": "*",
            "hooks": [
               {
                  "type": "command",
                  "command": "/proj/ds906659/.claude/claude_analysis-linux-amd64"
               }
            ]
         }
      ]
   }
}
```
