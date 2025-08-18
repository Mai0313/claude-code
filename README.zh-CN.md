# Claude Code CLI 使用说明

[English](README.md) | 简体中文 | [繁體中文](README.zh-TW.md)

## 简介

Claude Code 是 Anthropic 官方的 CLI 工具，提供 AI 编程助手和交互式开发支持。此安装程序提供自动化设置，具有智能网络检测和可选的 JWT 认证功能。

---

## 认证选项

### 选项 1：JWT Token 认证（推荐）
安装程序可自动获取和配置 JWT token，实现无缝认证：

1. 运行安装程序
2. 当提示时，选择 "y" 进行 JWT token 配置
3. 输入您的 MediaTek 凭据
4. 安装程序将自动：
   - 检测最佳可用的 MLOP 端点
   - 获取 JWT token
   - 配置认证头

### 选项 2：手动 API 密钥设置
如果您偏好手动配置或遇到 JWT 认证问题：

1. 前往 [MediaTek MLOP Gateway for OA](https://mlop-azure-gateway.mediatek.inc/auth/login) / [MediaTek MLOP Gateway for SWRD](https://mlop-azure-rddmz.mediatek.inc/auth/login)登录
2. 获取您的 GAISF API 密钥
3. 在设置中手动配置密钥

**注意**：安装程序自动检测网络连接性，并选择 HTTP/HTTPS 协议以获得最佳兼容性。

## 安装特性

安装程序包含可靠设置的高级功能：

### 智能依赖管理
- **Node.js 22+ 检测**：自动检查并安装所需的 Node.js 版本
- **平台特定安装**：
  - **macOS**：使用 Homebrew 自动安装
  - **Linux**：支持多种包管理器（apt、dnf、yum、pacman）
  - **Windows**：提供直接下载链接和引导安装

### 智能网络检测
- **多注册表支持**：自动测试 MediaTek 内部 npm 注册表以获得最佳下载速度
- **端点自动选择**：检测最佳可用的 MLOP 网关端点
- **回退机制**：如果主要连接失败，无缝切换到备用服务器

### 配置管理
- **系统级安装**：支持用户级和系统级配置
- **多平台二进制文件**：安装具有正确命名的平台特定 claude_analysis 二进制文件
- **托管设置**：自动生成优化的 settings.json，支持遥测和 MCP 服务器

---

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

### 第三步：配置认证
安装过程中，您将被提示配置认证：

1. **JWT Token 设置（推荐）**：
   - 当询问 JWT token 配置时选择 "y"
   - 输入您的 MediaTek 用户名和密码
   - 安装程序将安全获取并配置您的 JWT token

2. **跳过 JWT 设置**：
   - 选择 "N" 跳过 JWT 配置
   - 如需要，您可以稍后手动配置 API 密钥

安装程序自动处理所有技术配置，包括：
- 网络端点检测和选择
- 注册表回退配置
- 平台特定的二进制安装
- 优化的 Claude Code 设置

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
- **Node.js**：版本 22 或更高（在 macOS/Linux 上自动安装）
- **网络连接**：下载组件和认证时需要
- **凭据**：MediaTek 账户用于 JWT 认证（可选但推荐）

## 安装位置

安装程序创建以下文件和目录：

### 用户级安装（默认）
- **Claude CLI**：通过 npm 全局安装
- **配置**：`~/.claude/settings.json`
- **二进制文件**：`~/.claude/claude_analysis-{平台}-{架构}[.exe]`

### 系统级安装（如果可用）
- **macOS**：`/Library/Application Support/ClaudeCode/`
- **Linux**：`/etc/claude-code/`
- **Windows**：`C:\ProgramData\ClaudeCode\`

安装程序根据系统权限和要求自动选择最佳安装方法。

## 故障排除

**问题**：安装后 `claude --version` 不工作
**解决方案**：重启终端或将 npm 的全局 bin 目录添加到 PATH

**问题**：安装过程中 JWT 认证失败
**解决方案**：
- 验证您的 MediaTek 凭据是否正确
- 检查到 MLOP 网关的网络连接
- 跳过 JWT 设置并根据需要手动配置

**问题**：Linux/macOS 上 Node.js 安装失败
**解决方案**：
- 安装程序自动尝试多个包管理器
- 如果全部失败，从 https://nodejs.org/ 手动安装 Node.js 22+
- 手动安装 Node.js 后重新运行安装程序

**问题**：网络错误导致安装失败
**解决方案**：
- 安装程序自动尝试备用注册表和端点
- 确保您有网络连接
- 在企业防火墙后尝试使用适当的代理设置运行

**问题**：需要重新安装或更新
**解决方案**：您可以安全地多次运行安装程序 - 它会检测现有安装并适当更新

**问题**：配置文件未被读取
**解决方案**：安装程序在多个位置创建配置。检查：
- `~/.claude/settings.json`（用户级）
- 系统级托管设置（特定于操作系统的路径）
- 参考 [官方配置文档](https://docs.anthropic.com/zh-CN/docs/claude-code/settings#%E8%AE%BE%E7%BD%AE%E6%96%87%E4%BB%B6) 了解其他位置

## 额外资源

- [官方文档](https://docs.anthropic.com/zh-CN/docs/claude-code)
- [设置文档](https://docs.anthropic.com/zh-CN/docs/claude-code/settings)
- [子代理功能](https://docs.anthropic.com/zh-CN/docs/claude-code/sub-agents) - 探索专业任务的代理功能
- [MCP 集成](https://docs.anthropic.com/zh-CN/docs/claude-code/mcp) - 了解模型上下文协议支持

## 获取帮助

如果遇到问题：
1. 确保有网络连接
2. 尝试以管理员身份运行安装程序（Windows）或使用 `sudo`（macOS/Linux）进行系统级安装
3. 使用 `node --version` 检查 Node.js 是否正确安装（应为 22+）
4. 对于 JWT 认证问题，验证您的 MediaTek 账户凭据
5. 检查安装程序输出的特定错误消息，如需要可使用不同选项重试

## 此版本的新功能

### 增强的认证
- **自动 JWT Token 管理**：安全的凭据处理，自动 token 刷新
- **智能端点检测**：自动选择最佳可用的 MLOP 网关
- **凭据安全性**：在 Unix 系统上隐藏密码输入以确保安全

### 改进的网络可靠性
- **多注册表支持**：MediaTek 内部 npm 注册表之间的自动回退
- **连接性测试**：预安装网络检查确保最佳下载路径
- **协议自动选择**：基于网络条件的智能 HTTP/HTTPS 选择

### 高级安装选项
- **平台特定二进制文件**：为每个平台和架构正确命名的二进制文件
- **系统级配置**：支持企业级托管设置
- **依赖项自动安装**：在所有支持的平台上自动化 Node.js 设置

### 配置增强
- **优化的默认设置**：预配置遥测、MCP 服务器和协作功能
- **灵活的配置路径**：支持用户和系统级配置文件
- **交互式设置**：用户友好的认证和配置选择提示
