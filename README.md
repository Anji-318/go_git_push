

> **版本**: v1.1  
> **语言**: Go  
> **平台**: Windows / Linux / macOS / Termux / Android  
> **作者**: 基于社区需求定制开发  
> **发布日期**: 2026-07-11

---

## 📌 项目概述

Git Push Tool 是一款面向开发者的 **交互式 Git 管理终端工具**，以 Go 语言编写，提供赛博朋克风格的可视化操作界面。支持键盘方向键导航、一键式 Git 操作、自动认证注入、提交历史重写（屏蔽其他账户贡献者）等高级功能。

### 核心定位
- 替代繁琐的命令行 Git 操作
- 解决多账户提交历史混乱问题
- 提供安全的历史重写与强制推送能力
- 支持 Token 认证自动注入，无需每次手动输入密码

---

## ✨ 功能特性

### 1. 交互式菜单系统
- **方向键导航**: ⬆/⬇ 选择菜单项，Enter 执行
- **快捷跳转**: 直接输入数字/字母键直达对应功能
- **ESC 退出**: 一键安全退出
- **动态主题**: 菜单高亮颜色随选中项变化，Banner 随机颜色

### 2. 推送管理
| 功能 | 快捷键 | 说明 |
|------|--------|------|
| 普通推送 | `1` | 标准 `git push`，自动检测未提交更改 |
| 强制覆盖推送 | `2` | `git push --force`，覆盖远程历史 |
| 安全强制推送 | `3` | `git push --force-with-lease`，仅在远程无新提交时生效 |

### 3. 提交与变更
| 功能 | 快捷键 | 说明 |
|------|--------|------|
| 添加并提交 | `A` | `git add -A` + `git commit`，支持自定义提交信息 |
| 查看未提交更改 | `5` | 显示 `git status` + `git diff --stat` |
| 撤销上次提交 | `F` | `git reset --soft HEAD~1`，保留更改到暂存区 |

### 4. 分支管理
| 功能 | 快捷键 | 说明 |
|------|--------|------|
| 切换分支 | `D` | 列出所有分支并切换 |
| 创建并切换新分支 | `E` | `git checkout -b <branch>` |
| 删除本地分支 | `K` | 安全删除，禁止删除当前分支 |
| 合并分支 | `J` | `git merge`，自动检测冲突 |
| 查看分支列表 | `G` | 本地 + 远程分支详情 |

### 5. 远程操作
| 功能 | 快捷键 | 说明 |
|------|--------|------|
| 拉取远程更改 | `B` | `git pull`，自动处理冲突提示 |
| 获取远程分支 | `C` | `git fetch`，列出远程分支 |
| 硬重置到远程 | `I` | `git reset --hard`，丢弃本地未推送更改 |
| 查看远程信息 | `H` | `git remote -v` + `git remote show` |
| 测试连接 | `6` | 验证 Token 认证与远程仓库连通性 |

### 6. 历史重写（核心特色）
| 功能 | 快捷键 | 说明 |
|------|--------|------|
| 重写提交历史 | `8` | 使用 `git filter-branch` 替换指定作者的提交信息 |
| 查看作者统计 | `9` | 统计所有提交作者及其提交次数 |
| 删除无关文件 | `L` | 从历史中彻底删除已跟踪的敏感/无关文件 |

**重写历史功能详解**:
1. 自动扫描仓库所有提交作者
2. 输入要屏蔽的旧作者名/邮箱
3. 输入新的作者信息（默认读取当前 git config）
4. 自动创建 `rewrite-backup` 备份分支
5. 执行 `git filter-branch --env-filter` 替换作者
6. 完成后提示使用 `[2] 强制覆盖推送` 同步到远程

### 7. 安全与配置
| 功能 | 快捷键 | 说明 |
|------|--------|------|
| 清空配置登录信息 | `7` | 清空 `git-config.txt` 中的用户名和 Token，保留文件结构 |
| Token 自动注入 | — | 通过 `buildAuthURL` 将 Token 嵌入 HTTPS URL，无需交互式密码输入 |

---

## 🛠 技术架构

### 依赖库
| 库 | 用途 |
|----|------|
| `github.com/eiannone/keyboard` | 跨平台键盘事件监听（方向键、ESC、字母键） |
| `golang.org/x/term` | 终端尺寸获取、密码隐藏输入 |

### 核心模块
```
┌─────────────────────────────────────────────┐
│              main() 入口函数                 │
│  - 查找/初始化 Git 仓库                       │
│  - 读取 git-config.txt 认证配置              │
│  - 构建认证 URL (Token 注入)                  │
│  - 优化 Git 传输参数                          │
├─────────────────────────────────────────────┤
│         interactiveMenuLoop()               │
│  - 方向键导航 + 高亮渲染                      │
│  - 按键事件分发                             │
├─────────────────────────────────────────────┤
│         fallbackMenuLoop()                  │
│  - 无键盘库支持时的数字输入回退               │
├─────────────────────────────────────────────┤
│         executeOption()                     │
│  - 22 个功能选项的分发执行                    │
└─────────────────────────────────────────────┘
```

### 辅助工具函数
- `runGit()` / `runGitShow()` / `runGitOutput()` — Git 命令执行封装
- `findGitRepo()` — 向上递归查找 `.git` 目录
- `buildAuthURL()` — 构建 `https://user:token@github.com/...` 认证 URL
- `maskToken()` — 日志输出时脱敏 Token
- `scanTrackedFiles()` — 扫描已跟踪的敏感文件（exe/apk/zip/token 等）
- `buildDeletePatterns()` — 构建 `filter-branch` 删除模式

---

## 📦 安装与使用

### 环境要求
- Go 1.18+
- Git 已安装并配置
- GitHub 个人访问令牌 (PAT) 或 Fine-grained Token

### 安装步骤

```bash
# 1. 克隆或下载项目
cd /path/to/project

# 2. 初始化 Go 模块（如未初始化）
go mod init git-push-tool
go get github.com/eiannone/keyboard
go get golang.org/x/term

# 3. 编译
go build -o git_push.exe git_push.go        # Windows
go build -o git_push git_push.go            # Linux/macOS

# 4. 运行
./git_push.exe
```

### 配置文件 `git-config.txt`
首次运行会在项目目录自动生成：
```
GITHUB_USERNAME=your_username
GITHUB_TOKEN=ghp_xxxxxxxxxxxxxxxxxxxx
REPO_URL=https://github.com/your_username/your_repo.git
```

**安全提示**: `git-config.txt` 已加入 `.gitignore` 扫描列表，不会被提交到仓库。

### 认证方式
- **自动读取**: 优先读取 `git-config.txt` 中的配置
- **手动输入**: 无配置文件时交互式提示输入
- **Token 注入**: 运行时动态构建认证 URL，无需每次输入密码

---

## 🎮 操作指南

### 启动界面
![启动界面](./png/启动界面.png)

### 常用工作流

#### 场景 1: 日常提交推送
1. 运行 `./git_push.exe`
2. 按 `A` → 输入提交信息 → 自动 add + commit
3. 按 `1` → 普通推送到远程

#### 场景 2: 屏蔽其他账户贡献者
1. 按 `9` → 查看所有提交作者
2. 按 `8` → 输入要屏蔽的旧作者名/邮箱
3. 确认新作者信息（默认当前 git config）
4. 按 `2` → 强制覆盖推送
5. GitHub Contributors 列表将在数小时内更新

#### 场景 3: 删除历史中的敏感文件
1. 按 `L` → 自动扫描已跟踪的敏感文件
2. 确认删除 → 自动创建备份分支
3. 执行 `filter-branch` 重写历史
4. 按 `2` → 强制覆盖推送

#### 场景 4: 初始化新仓库
1. 在空目录运行 `./git_push.exe`
2. 提示"未找到 Git 仓库" → 确认初始化
3. 输入仓库地址 → 自动 `git init` + `git remote add`
4. 创建首次提交

---

## ⚠️ 注意事项

### 1. 强制推送风险
- `[2] 强制覆盖推送` 会 **永久覆盖远程提交历史**
- 执行前会显示本地/远程最近 5 条提交供确认
- 建议先在 `[9]` 查看作者统计，确认无误后再操作

### 2. 历史重写不可逆
- `[8] 重写提交历史` 和 `[L] 删除无关文件` 会改变所有 commit 的 hash
- 工具会自动创建 `rewrite-backup` / `delete-files-backup` 备份分支
- 如操作失败，可执行 `git checkout rewrite-backup` 恢复

### 3. Token 安全
- Token 通过 URL 注入方式传递，**不会存储在 Git 配置中**
- `git-config.txt` 包含敏感信息，确保已加入 `.gitignore`
- 日志输出中 Token 会被脱敏显示（`ghp_***`）

### 4. 跨平台兼容性
- Windows: 使用 `cmd /c cls` 清屏
- Linux/macOS/Termux/Android: 使用 ANSI 转义码清屏
- 键盘库初始化失败时自动回退到数字输入模式

### 5. 仓库初始化
- 工具会自动向上递归查找 `.git` 目录
- 支持在当前目录或父目录中定位仓库
- 新仓库初始化时会自动生成默认 `.gitignore`

---

## 📋 完整菜单速查表

| 快捷键 | 功能 | 对应 Git 命令 |
|--------|------|--------------|
| `1` | 普通推送 | `git push origin <branch>` |
| `2` | 强制覆盖推送 | `git push --force origin <branch>` |
| `3` | 安全强制推送 | `git push --force-with-lease origin <branch>` |
| `4` | 查看提交日志 | `git log --oneline` |
| `5` | 查看未提交更改 | `git status` + `git diff --stat` |
| `6` | 测试连接 | `git ls-remote --heads` |
| `7` | 清空配置登录信息 | 清空 `git-config.txt` |
| `8` | 重写提交历史 | `git filter-branch --env-filter` |
| `9` | 查看提交作者统计 | `git log --format=%an <%ae>` |
| `A` | 添加并提交更改 | `git add -A` + `git commit -m` |
| `B` | 拉取远程更改 | `git pull origin <branch>` |
| `C` | 获取远程分支 | `git fetch origin` |
| `D` | 切换分支 | `git checkout <branch>` |
| `E` | 创建并切换新分支 | `git checkout -b <branch>` |
| `F` | 撤销上次提交 | `git reset --soft HEAD~1` |
| `G` | 查看分支列表 | `git branch -v` + `git branch -r -v` |
| `H` | 查看远程信息 | `git remote -v` + `git remote show` |
| `I` | 硬重置到远程 | `git reset --hard origin/<branch>` |
| `J` | 合并分支 | `git merge <branch>` |
| `K` | 删除本地分支 | `git branch -D <branch>` |
| `L` | 删除历史无关文件 | `git filter-branch --index-filter` |
| `0` | 退出 | — |

---

## 🔧 Git 传输优化参数

工具启动时自动配置以下 Git 参数以提升大文件推送稳定性：

```bash
git config --local http.postBuffer 524288000      # 500MB 缓冲区
git config --local http.maxRequestBuffer 524288000
git config --local http.lowSpeedLimit 0             # 不限速
git config --local http.lowSpeedTime 999999       # 超长超时
git config --local core.compression 9               # 最大压缩
git config --local pack.windowMemory 256m
git config --local pack.packSizeLimit 256m
```

---

## 📝 更新日志

### v1.1 (2026-07-11)
- ✅ 初始版本发布
- ✅ 22 项 Git 操作功能
- ✅ 方向键交互式菜单
- ✅ Token 认证自动注入
- ✅ 提交历史重写（屏蔽其他账户）
- ✅ 敏感文件历史清理
- ✅ 跨平台支持（Windows/Linux/macOS/Termux/Android）
- ✅ 自动备份分支机制
- ✅ Git 传输参数优化

---

## 🤝 使用建议

1. **首次使用**: 先执行 `[6] 测试连接` 确认认证有效
2. **日常开发**: 使用 `[A] 添加提交` + `[1] 普通推送` 组合
3. **多人协作**: 推送前先用 `[B] 拉取远程` 避免冲突
4. **历史清理**: 定期使用 `[L] 删除无关文件` 保持仓库整洁
5. **账户安全**: 使用 `[8] 重写历史` 屏蔽不再使用的旧账户提交

---

> **提示**: 本工具所有危险操作（强制推送、历史重写、硬重置）均会要求二次确认，确保操作安全。
