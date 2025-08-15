# Claude Code 安装程序

一个简单的安装工具，帮助您设置 Claude Code 和开发活动分析功能。

## 这个安装程序能做什么？

安装程序会自动：
1. **安装 Node.js**（如果尚未安装）- Claude Code 运行所需
2. **安装 Claude Code CLI** - 主要应用程序
3. **设置分析集成** - 跟踪您的编程活动以获得洞察
4. **自动配置一切** - 无需手动配置

## 使用方法

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
- 分析组件已准备好跟踪您的开发活动

## 故障排除

**问题**：安装后 `claude --version` 不工作
**解决方案**：重启终端或将 npm 的全局 bin 目录添加到 PATH

**问题**：第一次安装失败
**解决方案**：安装程序会自动使用备用服务器重试 - 请等待完成

**问题**：需要重新安装或更新
**解决方案**：您可以安全地多次运行安装程序 - 不会破坏任何东西

## 获取帮助

如果遇到问题：
1. 确保有网络连接
2. 尝试以管理员身份运行安装程序（Windows）或使用 `sudo`（macOS/Linux）进行系统级安装
3. 使用 `node --version` 检查 Node.js 是否正确安装
