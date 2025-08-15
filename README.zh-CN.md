# Claude Code CLI 使用说明

[English](README.md) | 简体中文 | [繁體中文](README.zh-TW.md)

## 简介

Claude Code 是 Anthropic 官方的 CLI 工具，提供 AI 编程助手和交互式开发支持。

---

## 获取 API 密钥

**重要**：使用 Claude Code 前，请先完成以下步骤：

1. 前往 [MediaTek MLOP Gateway for OA](https://mlop-azure-gateway.mediatek.inc/auth/login) / [MediaTek MLOP Gateway for SWRD](https://mlop-azure-rddmz.mediatek.inc/auth/login)登录
2. 获取您的 GAISF API 密钥
3. 妥善保存密钥以供后续使用

**注意**：由于 SSL 证书设置问题，本说明文档中的网址使用 HTTP 而非 HTTPS，以确保在不同网络环境下的兼容性。

---

**设置说明：**
- 请将 `<<您的 GAISF API 密钥>>` 替换为您的实际 API 密钥
- `ANTHROPIC_BEDROCK_BASE_URL` 设置为使用 HTTP 而非 HTTPS，这是由于 SSL 证书设置需求
- **重要提醒**：在某些情况下，Claude Code 可能无法读取 `~/.claude/settings.json` 配置文件。如果您遇到设置问题，请参考 [官方配置文件说明文档](https://docs.anthropic.com/zh-CN/docs/claude-code/settings#%E8%AE%BE%E7%BD%AE%E6%96%87%E4%BB%B6) 了解其他配置文件位置和故障排除步骤

## 安装

### 第一步：下载
从以下页面下载最新的安装包：
https://gitea.mediatek.inc/IT-GAIA/claude-code-monitor/releases

### 第二步：解压并运行
1. 解压下载的 zip 文件
2. 在解压后的文件夹中打开终端/命令提示符
3. 运行安装程序：
   - **Linux/macOS**：`./installer`
   - **Windows**：`installer.exe`

### 第三步：按照提示操作
安装程序会引导您完成设置过程并显示进度信息。

## Windows 用户重要提醒

**⚠️ Windows 用户必须先手动安装 Node.js！**

如果您没有安装 Node.js，安装程序会：
1. 向您显示 Node.js 的直接下载链接
2. 退出，让您先安装 Node.js
3. 要求您在安装 Node.js 后重新运行安装程序

**Windows 快速步骤：**
1. 从安装程序提供的链接下载 Node.js
2. 安装 Node.js MSI 安装包
3. 重新启动命令提示符
4. 再次运行安装程序

## 系统要求

- **操作系统**：Windows、macOS 或 Linux
- **Node.js**：LTS 版本（在 macOS/Linux 上会自动安装）
- **网络连接**：下载组件时需要

## 安装文件位置

成功安装后，您会发现：
- Claude Code CLI 全局可用（尝试 `claude --version`）
- 配置文件位于您的主目录下的 `.claude/` 文件夹

## 故障排除

**问题**：安装后 `claude --version` 不工作
**解决方案**：重启终端或将 npm 的全局 bin 目录添加到 PATH

**问题**：第一次安装失败
**解决方案**：安装程序会自动使用备用服务器重试 - 请等待完成

**问题**：需要重新安装或更新
**解决方案**：您可以安全地多次运行安装程序 - 不会破坏任何东西

## 额外资源

- [官方文档](https://docs.anthropic.com/zh-CN/docs/claude-code)
- [设置文档](https://docs.anthropic.com/zh-CN/docs/claude-code/settings)
- [子代理功能](https://docs.anthropic.com/zh-CN/docs/claude-code/sub-agents) - 探索专业任务的代理功能
- [MCP 集成](https://docs.anthropic.com/zh-CN/docs/claude-code/mcp) - 了解模型上下文协议支持

## 获取帮助

如果遇到问题：
1. 确保有网络连接
2. 尝试以管理员身份运行安装程序（Windows）或使用 `sudo`（macOS/Linux）进行系统级安装
3. 使用 `node --version` 检查 Node.js 是否正确安装
