package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/eiannone/keyboard"
	"golang.org/x/term"
)

// ========== 颜色常量 ==========
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
	ColorBold   = "\033[1m"
)

var (
	projectDir     string
	originalDir    string
	currentBranch  string
	remoteName     string
	remoteURL      string
	authURL        string
	configFile     string
	githubUsername string
	githubToken    string
)

// ========== 随机颜色 ==========
func randomColor() string {
	colors := []string{
		ColorRed, ColorGreen, ColorYellow, ColorBlue, ColorMagenta, ColorCyan,
	}
	return colors[rand.Intn(len(colors))]
}

// ========== 计算字符串显示宽度（支持中文/emoji双宽） ==========
func getStringDisplayWidth(s string) int {
	width := 0
	for _, r := range s {
		if utf8.RuneLen(r) > 1 {
			width += 2
		} else {
			width += 1
		}
	}
	return width
}

// ========== 移除ANSI颜色代码 ==========
func removeColorCodes(s string) string {
	re := regexp.MustCompile("\033\\[[0-9;]*m")
	return re.ReplaceAllString(s, "")
}

// ========== 居中打印文本 ==========
func printCentered(text string, termWidth int) {
	clean := removeColorCodes(text)
	w := getStringDisplayWidth(clean)
	if w >= termWidth {
		fmt.Println(text)
		return
	}
	padding := (termWidth - w) / 2
	fmt.Println(strings.Repeat(" ", padding) + text)
}

// ========== 获取终端宽度 ==========
func getTerminalWidth() int {
	width := 80
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
		width = w
	}
	return width
}

// ========== 清屏（跨平台） ==========
func clearScreen() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		// Linux, macOS, Termux, Android 等使用 ANSI 清屏
		fmt.Print("\033[2J\033[H")
		return
	}
	cmd.Stdout = os.Stdout
	cmd.Run()
}

// ========== 清屏并显示标题 ==========
func clearScreenAndShowBanner(termWidth int, bannerColor string) {
	clearScreen()
	
	// 艺术字 - 固定宽度，手动居中
	artWidth := 62 // 艺术字实际宽度
	pad := (termWidth - artWidth) / 2
	if pad < 0 { pad = 0 }
	padding := strings.Repeat(" ", pad)
	
	fmt.Println()
	fmt.Println(padding + bannerColor + "  ██████╗ ██╗████████╗    ██████╗ ██╗   ██╗███████╗██╗  ██╗" + ColorReset)
	fmt.Println(padding + bannerColor + " ██╔════╝ ██║╚══██╔══╝    ██╔══██╗██║   ██║██╔════╝██║  ██║" + ColorReset)
	fmt.Println(padding + bannerColor + " ██║  ███╗██║   ██║       ██████╔╝██║   ██║███████╗███████║" + ColorReset)
	fmt.Println(padding + bannerColor + " ██║   ██║██║   ██║       ██╔═══╝ ██║   ██║╚════██║██╔══██║" + ColorReset)
	fmt.Println(padding + bannerColor + " ╚██████╔╝██║   ██║       ██║     ╚██████╔╝███████║██║  ██║" + ColorReset)
	fmt.Println(padding + bannerColor + "  ╚═════╝ ╚═╝   ╚═╝       ╚═╝      ╚═════╝ ╚══════╝╚═╝  ╚═╝" + ColorReset)
	fmt.Println()
}

// ========== 菜单选项结构 ==========
type menuOption struct {
	key   string
	text  string
	color string
}

func main() {
	rand.Seed(time.Now().UnixNano())
	originalDir, _ = os.Getwd()
	termWidth := getTerminalWidth()

	// 查找 .git 目录
	projectDir = findGitRepo(originalDir)
	if projectDir == "" {
		clearScreenAndShowBanner(termWidth, ColorCyan)
		fmt.Println("[信息] 未找到 Git 仓库")
		if !confirm("是否在此目录初始化新仓库") {
			fmt.Println("[退出]")
			pause()
			return
		}
		initRepo()
	}

	os.Chdir(projectDir)

	// 获取当前分支
	currentBranch = runGit("branch", "--show-current")
	if currentBranch == "" {
		fmt.Println("[信息] 无分支，创建默认分支 main...")
		runGit("checkout", "-b", "main")
		currentBranch = "main"
	}

	// 检测远程仓库
	remotes := runGit("remote")
	if remotes == "" {
		remoteName = "origin"
		configFile = filepath.Join(projectDir, "git-config.txt")
		readConfig()
		if remoteURL == "" {
			fmt.Print("[输入] 请输入仓库地址: ")
			remoteURL = readLine()
			if remoteURL == "" {
				fmt.Println("[错误] 仓库地址不能为空")
				pause()
				os.Exit(1)
			}
		}
		runGit("remote", "add", remoteName, remoteURL)
	} else {
		remoteName = strings.Split(remotes, "\n")[0]
		remoteURL = strings.TrimSpace(runGit("remote", "get-url", remoteName))
	}

	// 修复 URL
	remoteURL = strings.ReplaceAll(remoteURL, "github.com<用户名>", "github.com/<用户名>")
	remoteURL = strings.ReplaceAll(remoteURL, "github.com//", "github.com/")
	if !strings.HasPrefix(remoteURL, "http://") && !strings.HasPrefix(remoteURL, "https://") {
		remoteURL = "https://" + remoteURL
	}

	// 读取认证信息
	if configFile == "" {
		configFile = filepath.Join(projectDir, "git-config.txt")
		readConfig()
	}

	if githubUsername == "" {
		fmt.Print("[输入] 请输入 GitHub 用户名: ")
		githubUsername = readLine()
	}
	if githubToken == "" {
		fmt.Print("[输入] 请输入 GitHub Token: ")
		githubToken = readPassword()
		fmt.Println()
	}

	if githubUsername == "" || githubToken == "" {
		fmt.Println("[错误] 用户名和 Token 不能为空")
		pause()
		return
	}

	// 构建认证URL
	cleanURL := remoteURL
	if idx := strings.Index(cleanURL, "@"); idx != -1 {
		cleanURL = cleanURL[idx+1:]
	}
	if !strings.HasPrefix(cleanURL, "https://") && !strings.HasPrefix(cleanURL, "http://") {
		cleanURL = "https://" + cleanURL
	}
	authURL = buildAuthURL(cleanURL, githubUsername, githubToken)

	// 优化 Git 参数
	gitConfig("local", "http.postBuffer", "524288000")
	gitConfig("local", "http.maxRequestBuffer", "524288000")
	gitConfig("local", "http.lowSpeedLimit", "0")
	gitConfig("local", "http.lowSpeedTime", "999999")
	gitConfig("local", "core.compression", "9")
	gitConfig("local", "pack.windowMemory", "256m")
	gitConfig("local", "pack.packSizeLimit", "256m")

	// 初始化键盘
	if err := keyboard.Open(); err != nil {
		fmt.Printf("%s[错误] 无法初始化键盘输入: %v%s\n", ColorRed, err, ColorReset)
		fmt.Println("[提示] 将使用传统数字输入模式")
		fallbackMenuLoop(termWidth)
		return
	}
	defer keyboard.Close()

	// 交互式菜单循环
	interactiveMenuLoop(termWidth)
}

// ========== 交互式菜单（方向键+高亮） ==========
func interactiveMenuLoop(termWidth int) {
	options := []menuOption{
		{"1", "普通推送", ColorGreen},
		{"2", "强制覆盖推送", ColorRed},
		{"3", "安全强制推送", ColorYellow},
		{"4", "查看提交日志", ColorBlue},
		{"5", "查看未提交更改", ColorCyan},
		{"6", "测试连接", ColorGreen},
		{"7", "清空配置登录信息", ColorRed},
		{"8", "重写提交历史（屏蔽其他账户）", ColorMagenta},
		{"9", "查看提交作者统计", ColorBlue},
		{"A", "添加并提交更改", ColorGreen},
		{"B", "拉取远程更改", ColorCyan},
		{"C", "获取远程分支", ColorBlue},
		{"D", "切换分支", ColorYellow},
		{"E", "创建并切换新分支", ColorCyan},
		{"F", "撤销上次提交", ColorRed},
		{"G", "查看分支列表", ColorBlue},
		{"H", "查看远程信息", ColorCyan},
		{"I", "硬重置到远程", ColorRed},
		{"J", "合并分支", ColorYellow},
		{"K", "删除本地分支", ColorRed},
		{"L", "从历史删除已跟踪的无关文件", ColorMagenta},
		{"0", "退出", ColorRed},
	}

	selected := 0
	bannerColor := randomColor()

	for {
		// 清屏 + 显示艺术字（随机颜色）
		clearScreenAndShowBanner(termWidth, bannerColor)

		// 显示信息栏
		infoLine := fmt.Sprintf(" 用户: %s%s%s | 分支: %s%s%s | 远程: %s%s%s",
			ColorCyan, githubUsername, ColorReset,
			ColorYellow, currentBranch, ColorReset,
			ColorGreen, remoteName, ColorReset)
		fmt.Println(infoLine)
		fmt.Println(strings.Repeat("─", termWidth))
		fmt.Println()

		// 显示菜单选项
		for i, opt := range options {
			if i == selected {
				fmt.Printf(" %s➤ %s[%s%s%s] %s%s%s\n",
					ColorCyan, ColorReset,
					ColorBold+opt.color, opt.key, ColorReset,
					ColorBold+ColorWhite, opt.text, ColorReset)
			} else {
				fmt.Printf("   %s[%s] %s%s\n",
					opt.color, opt.key, opt.text, ColorReset)
			}
		}

		fmt.Println()
		fmt.Printf(" %s⬆/⬇ 方向键选择  |  数字/字母直接跳转  |  Enter执行  |  ESC退出%s\n", ColorYellow, ColorReset)

		// 读取按键
		char, key, err := keyboard.GetKey()
		if err != nil {
			continue
		}

		switch key {
		case keyboard.KeyArrowUp:
			selected--
			if selected < 0 {
				selected = len(options) - 1
			}
			bannerColor = options[selected].color
		case keyboard.KeyArrowDown:
			selected++
			if selected >= len(options) {
				selected = 0
			}
			bannerColor = options[selected].color
		case keyboard.KeyEnter:
			executeOption(options[selected].key, termWidth)
			if options[selected].key == "0" {
				os.Chdir(originalDir)
				return
			}
		case keyboard.KeyEsc:
			os.Chdir(originalDir)
			return
		default:
			upper := strings.ToUpper(string(char))
			for i, opt := range options {
				if opt.key == upper {
					selected = i
					bannerColor = opt.color
					executeOption(opt.key, termWidth)
					if opt.key == "0" {
						os.Chdir(originalDir)
						return
					}
					break
				}
			}
		}
	}
}

// ========== 回退菜单（无键盘库支持时使用） ==========
func fallbackMenuLoop(termWidth int) {
	for {
		clearScreenAndShowBanner(termWidth, ColorCyan)
		fmt.Printf(" 用户: %s | 分支: %s | 远程: %s\n", githubUsername, currentBranch, remoteName)
		fmt.Println(strings.Repeat("─", termWidth))
		fmt.Println()
		fmt.Println("  [1] 普通推送    [2] 强制覆盖推送  [3] 安全强制推送")
		fmt.Println("  [4] 查看日志    [5] 查看更改      [6] 测试连接")
		fmt.Println("  [7] 清空配置    [8] 重写历史      [9] 作者统计")
		fmt.Println("  [A] 添加提交    [B] 拉取远程      [C] 获取分支")
		fmt.Println("  [D] 切换分支    [E] 创建分支      [F] 撤销提交")
		fmt.Println("  [G] 分支列表    [H] 远程信息      [I] 硬重置")
		fmt.Println("  [J] 合并分支    [K] 删除分支      [L] 删除无关文件")
		fmt.Println("  [0] 退出")
		fmt.Println()
		fmt.Print(" 请选择 [0-9,A-L]: ")
		choice := strings.ToUpper(readLine())
		if choice == "0" {
			os.Chdir(originalDir)
			return
		}
		executeOption(choice, termWidth)
	}
}

// ========== 执行选项 ==========
func executeOption(choice string, termWidth int) {
	// 清屏显示执行中
	clearScreenAndShowBanner(termWidth, ColorGreen)

	switch choice {
	case "1":
		normalPush()
	case "2":
		forcePush()
	case "3":
		forceLeasePush()
	case "4":
		showLog()
	case "5":
		showDiff()
	case "6":
		testConnection()
	case "7":
		clearToken()
	case "8":
		rewriteHistory()
	case "9":
		showAuthors()
	case "A":
		addCommit()
	case "B":
		pullRemote()
	case "C":
		fetchRemote()
	case "D":
		switchBranch()
	case "E":
		createBranch()
	case "F":
		undoCommit()
	case "G":
		showBranches()
	case "H":
		showRemoteInfo()
	case "I":
		hardReset()
	case "J":
		mergeBranch()
	case "K":
		deleteBranch()
	case "L":
		removeTrackedFiles()
	default:
		fmt.Println("[错误] 无效选择")
	}

	if choice != "0" {
		fmt.Println()
		fmt.Printf("%s按任意键返回菜单...%s", ColorYellow, ColorReset)
		keyboard.GetKey()
	}
}

// ========== 原有功能函数（保持不变） ==========

func normalPush() {
	fmt.Println("==========================================")
	fmt.Println("           普通推送")
	fmt.Println("==========================================")
	fmt.Println()
	if hasUncommittedChanges() {
		fmt.Println("[警告] 检测到未提交的更改！")
		runGitShow("status", "--short")
		fmt.Println()
		if !confirm("是否继续推送已提交的更改") {
			return
		}
	}
	fmt.Printf("[执行] git push %s %s ...\n", remoteName, currentBranch)
	out, err := runGitOutput("push", authURL, currentBranch)
	if err != nil {
		fmt.Println("[失败] 推送被拒绝")
		fmt.Println(out)
	} else {
		fmt.Println("[成功] 推送完成！")
	}
}

func forcePush() {
	fmt.Println("==========================================")
	fmt.Println("        强制覆盖推送")
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Println("[警告] 此操作会覆盖远程历史！")
	fmt.Println()
	fmt.Println("[信息] 本地提交:")
	runGitShow("log", "--oneline", "-5")
	fmt.Println()
	fmt.Println("[信息] 远程提交:")
	showRemoteLog(5)
	fmt.Println()
	if !confirm("确认强制覆盖远程仓库") {
		return
	}
	fmt.Println()
	fmt.Printf("[执行] git push --force %s %s ...\n", remoteName, currentBranch)
	out, err := runGitOutput("push", "--force", authURL, currentBranch)
	if err != nil {
		fmt.Println("[失败] 强制推送失败")
		fmt.Println(out)
	} else {
		fmt.Println("[成功] 强制推送完成！")
	}
}

func forceLeasePush() {
	fmt.Println("==========================================")
	fmt.Println("        安全强制推送")
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Println("[说明] 仅当远程没有新提交时才会强制推送")
	fmt.Println()
	fmt.Println("[信息] 本地 vs 远程差异:")
	showRemoteDiff()
	fmt.Println()
	if !confirm("确认安全强制推送") {
		return
	}
	fmt.Println()
	out, err := runGitOutput("push", "--force-with-lease", authURL, currentBranch)
	if err != nil {
		fmt.Println("[失败] 推送被拒绝！远程可能有新提交")
		fmt.Println(out)
	} else {
		fmt.Println("[成功] 安全强制推送完成！")
	}
}

func showLog() {
	fmt.Println("==========================================")
	fmt.Println("           提交日志")
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Println("[本地提交] (最近10条):")
	runGitShow("log", "--oneline", "-10")
	fmt.Println()
	fmt.Println("[远程提交] (最近5条):")
	showRemoteLog(5)
	fmt.Println()
}

func showDiff() {
	fmt.Println("==========================================")
	fmt.Println("           未提交更改")
	fmt.Println("==========================================")
	fmt.Println()
	runGitShow("status")
	fmt.Println()
	fmt.Println("==========================================")
	runGitShow("diff", "--stat")
	fmt.Println()
}

func testConnection() {
	fmt.Println("==========================================")
	fmt.Println("           测试连接")
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Println("[测试] 验证认证信息...")
	maskedURL := maskToken(authURL)
	fmt.Println("[URL]", maskedURL)
	fmt.Println()
	_, err := runGitOutput("ls-remote", "--heads", authURL, currentBranch)
	if err != nil {
		fmt.Println("[失败] 连接失败！请检查:")
		fmt.Println("  - 用户名和令牌是否正确")
		fmt.Println("  - 令牌是否有 repo 权限")
		fmt.Println("  - 仓库地址是否可访问")
	} else {
		fmt.Println("[成功] 连接正常，认证有效！")
		fmt.Println("[信息] 远程分支存在，可以推送")
	}
	fmt.Println()
}

func clearToken() {
	fmt.Println("==========================================")
	fmt.Println("        清空配置登录信息")
	fmt.Println("==========================================")
	fmt.Println()
	if _, err := os.Stat(configFile); err == nil {
		fmt.Println("[信息] 正在清空", configFile, "中的登录信息...")
		content := "GITHUB_USERNAME=\nGITHUB_TOKEN=\n"
		os.WriteFile(configFile, []byte(content), 0644)
		fmt.Println("[完成] 配置已清空！")
		fmt.Println("[提示] 文件保留，用户名和令牌已设为空值")
		fmt.Println("[提示] 下次运行将提示手动输入")
	} else {
		fmt.Println("[提示] 未找到 git-config.txt，无需清空")
	}
}

func showAuthors() {
	fmt.Println("==========================================")
	fmt.Println("        提交作者统计")
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Println("[信息] 当前仓库所有提交的作者统计：")
	fmt.Println()
	out := runGit("log", "--format=%an <%ae>")
	lines := strings.Split(out, "\n")
	counts := make(map[string]int)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			counts[line]++
		}
	}
	for author, count := range counts {
		fmt.Printf("  %4d  %s\n", count, author)
	}
	fmt.Println()
	fmt.Println("[提示] 上方列出的是所有提交过的作者")
	fmt.Println("[提示] 选择 [8] 可以重写历史")
}

func rewriteHistory() {
	fmt.Println("==========================================")
	fmt.Println("     重写提交历史（屏蔽其他账户）")
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Println("[警告] 此操作会重写所有 commit 的 hash 值！")
	fmt.Println("[警告] 远程仓库的历史将被完全替换！")
	fmt.Println()
	fmt.Println("[当前提交作者统计]:")
	showAuthorsSimple()
	fmt.Println()
	fmt.Println("[步骤 1/4] 输入要屏蔽的作者信息")
	fmt.Println()
	fmt.Println("[提示] 上方列表中显示的即为所有提交作者")
	fmt.Print("要屏蔽的作者名 (如 <旧用户名>): ")
	oldAuthor := readLine()
	fmt.Print("要屏蔽的作者邮箱 (如 <旧邮箱@example.com>): ")
	oldEmail := readLine()
	if oldAuthor == "" {
		fmt.Println("[错误] 作者名不能为空")
		return
	}
	defaultAuthor := githubUsername
	defaultEmail := runGit("config", "user.email")
	if defaultEmail == "" {
		defaultEmail = runGit("config", "user.name")
	}
	if defaultAuthor == "" {
		defaultAuthor = defaultEmail
	}
	fmt.Println()
	fmt.Println("[步骤 2/4] 输入新的作者信息")
	fmt.Println()
	if defaultAuthor != "" {
		fmt.Println("[默认] 作者名:", defaultAuthor)
	}
	if defaultEmail != "" {
		fmt.Println("[默认] 邮箱  :", defaultEmail)
	}
	fmt.Println()
	fmt.Printf("新作者名 [直接回车=%s]: ", defaultAuthor)
	newAuthor := readLine()
	fmt.Printf("新作者邮箱 [直接回车=%s]: ", defaultEmail)
	newEmail := readLine()
	if newAuthor == "" {
		if defaultAuthor == "" {
			fmt.Println("[错误] 新作者名不能为空")
			return
		}
		newAuthor = defaultAuthor
	}
	if newEmail == "" {
		if defaultEmail == "" {
			fmt.Println("[错误] 新作者邮箱不能为空")
			return
		}
		newEmail = defaultEmail
	}
	fmt.Println()
	fmt.Println("[步骤 3/4] 确认替换信息")
	fmt.Println()
	fmt.Println("  旧作者名:", oldAuthor)
	fmt.Println("  旧邮箱  :", oldEmail)
	fmt.Println("  ─────────────────────────")
	fmt.Println("  新作者名:", newAuthor)
	fmt.Println("  新邮箱  :", newEmail)
	fmt.Println()
	if !confirm("确认执行替换") {
		return
	}
	fmt.Println()
	fmt.Println("[步骤 4/4] 正在执行替换...")
	fmt.Println()
	fmt.Println("[备份] 创建备份分支 rewrite-backup ...")
	runGit("branch", "rewrite-backup")
	fmt.Println("[完成] 备份分支已创建")
	fmt.Println()
	fmt.Println("[执行] git filter-branch 替换作者信息...")
	fmt.Println("[提示] 这可能需要一些时间，请耐心等待...")
	fmt.Println()
	var envFilter string
	if oldEmail == "" {
		envFilter = fmt.Sprintf(`
if [ "$GIT_AUTHOR_NAME" = "%s" ]; then
	export GIT_AUTHOR_NAME="%s"
	export GIT_AUTHOR_EMAIL="%s"
	export GIT_COMMITTER_NAME="%s"
	export GIT_COMMITTER_EMAIL="%s"
fi`, oldAuthor, newAuthor, newEmail, newAuthor, newEmail)
	} else {
		envFilter = fmt.Sprintf(`
if [ "$GIT_AUTHOR_NAME" = "%s" ] || [ "$GIT_AUTHOR_EMAIL" = "%s" ]; then
	export GIT_AUTHOR_NAME="%s"
	export GIT_AUTHOR_EMAIL="%s"
	export GIT_COMMITTER_NAME="%s"
	export GIT_COMMITTER_EMAIL="%s"
fi`, oldAuthor, oldEmail, newAuthor, newEmail, newAuthor, newEmail)
	}
	cmd := exec.Command("git", "filter-branch", "--env-filter", envFilter, "--force", "HEAD")
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println()
		fmt.Println("[失败] git filter-branch 执行失败！")
		fmt.Println("[恢复] 可以从备份分支恢复: git checkout rewrite-backup")
		return
	}
	fmt.Println()
	fmt.Println("[成功] 作者信息替换完成！")
	fmt.Println()
	fmt.Println("[验证] 新的作者统计：")
	showAuthorsSimple()
	fmt.Println()
	fmt.Println("[下一步] 请选择 [2] 强制覆盖推送")
	fmt.Println("[提示] GitHub 会在几小时内更新 Contributors 列表")
}

func addCommit() {
	fmt.Println("==========================================")
	fmt.Println("        添加并提交更改")
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Println("[当前更改状态]:")
	runGitShow("status", "--short")
	fmt.Println()
	fmt.Print("输入提交信息: ")
	msg := readLine()
	if msg == "" {
		msg = "Update"
	}
	fmt.Println("[执行] git add -A ...")
	runGit("add", "-A")
	fmt.Printf("[执行] git commit -m \"%s\" ...\n", msg)
	_, err := runGitOutput("commit", "-m", msg)
	if err != nil {
		fmt.Println("[提示] 提交失败，可能没有更改或已提交")
	} else {
		fmt.Println("[成功] 提交完成！")
	}
}

func pullRemote() {
	fmt.Println("==========================================")
	fmt.Println("        拉取远程更改")
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Printf("[执行] git pull %s %s ...\n", remoteName, currentBranch)
	out, err := runGitOutput("pull", authURL, currentBranch)
	if err != nil {
		fmt.Println("[失败] 拉取失败")
		fmt.Println("[提示] 可能存在冲突，请手动解决")
		fmt.Println(out)
	} else {
		fmt.Println("[成功] 拉取完成！")
	}
}

func fetchRemote() {
	fmt.Println("==========================================")
	fmt.Println("        获取远程分支")
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Printf("[执行] git fetch %s ...\n", remoteName)
	out, err := runGitOutput("fetch", authURL)
	if err != nil {
		fmt.Println("[失败] 获取失败")
		fmt.Println(out)
	} else {
		fmt.Println("[成功] 远程分支已获取！")
		fmt.Println()
		fmt.Println("[远程分支列表]:")
		runGitShow("branch", "-r")
	}
}

func switchBranch() {
	fmt.Println("==========================================")
	fmt.Println("        切换分支")
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Println("[可用分支]:")
	runGitShow("branch", "-a")
	fmt.Println()
	fmt.Print("输入要切换的分支名: ")
	branch := readLine()
	if branch == "" {
		return
	}
	_, err := runGitOutput("checkout", branch)
	if err != nil {
		fmt.Println("[失败] 切换失败")
	} else {
		currentBranch = strings.TrimSpace(runGit("branch", "--show-current"))
		fmt.Println("[成功] 已切换到分支:", currentBranch)
	}
}

func createBranch() {
	fmt.Println("==========================================")
	fmt.Println("        创建并切换新分支")
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Print("输入新分支名: ")
	branch := readLine()
	if branch == "" {
		return
	}
	_, err := runGitOutput("checkout", "-b", branch)
	if err != nil {
		fmt.Println("[失败] 创建失败")
	} else {
		currentBranch = strings.TrimSpace(runGit("branch", "--show-current"))
		fmt.Println("[成功] 已创建并切换到:", currentBranch)
	}
}

func undoCommit() {
	fmt.Println("==========================================")
	fmt.Println("        撤销上次提交")
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Println("[当前最新提交]:")
	runGitShow("log", "--oneline", "-1")
	fmt.Println()
	fmt.Println("[说明] 此操作撤销最后一次提交，但保留文件更改到暂存区")
	fmt.Println()
	if !confirm("确认撤销上次提交") {
		return
	}
	_, err := runGitOutput("reset", "--soft", "HEAD~1")
	if err != nil {
		fmt.Println("[失败]")
	} else {
		fmt.Println("[成功] 上次提交已撤销，更改保留在暂存区")
	}
}

func showBranches() {
	fmt.Println("==========================================")
	fmt.Println("        分支列表")
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Println("[本地分支]:")
	runGitShow("branch", "-v")
	fmt.Println()
	fmt.Println("[远程分支]:")
	runGitShow("branch", "-r", "-v")
	fmt.Println()
	fmt.Println("[当前分支]:", currentBranch)
}

func showRemoteInfo() {
	fmt.Println("==========================================")
	fmt.Println("        远程仓库信息")
	fmt.Println("==========================================")
	fmt.Println()
	runGitShow("remote", "-v")
	fmt.Println()
	out, _ := runGitOutput("remote", "show", remoteName)
	if out == "" {
		fmt.Println("[无法获取详细信息]")
	} else {
		fmt.Println(out)
	}
}

func hardReset() {
	fmt.Println("==========================================")
	fmt.Println("        硬重置到远程版本")
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Println("[警告] 此操作会丢弃所有本地未推送的更改！")
	fmt.Println("[当前分支]:", currentBranch)
	fmt.Println()
	if !confirm(fmt.Sprintf("确认硬重置到远程 %s/%s", remoteName, currentBranch)) {
		return
	}
	runGit("fetch", authURL, currentBranch)
	_, err := runGitOutput("reset", "--hard", remoteName+"/"+currentBranch)
	if err != nil {
		fmt.Println("[失败]")
	} else {
		fmt.Println("[成功] 本地已重置为远程版本！")
	}
}

func mergeBranch() {
	fmt.Println("==========================================")
	fmt.Println("        合并分支")
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Println("[当前分支]:", currentBranch)
	fmt.Println("[可用分支]:")
	runGitShow("branch", "-v")
	fmt.Println()
	fmt.Print("输入要合并到当前分支的分支名: ")
	branch := readLine()
	if branch == "" {
		return
	}
	_, err := runGitOutput("merge", branch)
	if err != nil {
		fmt.Println()
		fmt.Println("[失败] 合并冲突！请手动解决")
	} else {
		fmt.Println("[成功] 合并完成！")
	}
}

func deleteBranch() {
	fmt.Println("==========================================")
	fmt.Println("        删除本地分支")
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Println("[本地分支]:")
	runGitShow("branch", "-v")
	fmt.Println()
	fmt.Print("输入要删除的分支名: ")
	branch := readLine()
	if branch == "" {
		return
	}
	if strings.EqualFold(branch, currentBranch) {
		fmt.Println("[错误] 不能删除当前所在分支！")
		return
	}
	if !confirm(fmt.Sprintf("确认删除分支 %s", branch)) {
		return
	}
	_, err := runGitOutput("branch", "-D", branch)
	if err != nil {
		fmt.Println("[失败] 删除失败")
	} else {
		fmt.Println("[成功] 分支已删除")
	}
}

func removeTrackedFiles() {
	fmt.Println("==========================================")
	fmt.Println("     从历史中删除已跟踪的无关文件")
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Println("[警告] 此操作会重写所有 commit 的 hash 值！")
	fmt.Println("[警告] 远程仓库的历史将被完全替换！")
	fmt.Println()
	fmt.Println("[说明] .gitignore 只能忽略未跟踪的新文件")
	fmt.Println("[说明] 对于已错误提交的文件，需要重写历史才能彻底删除")
	fmt.Println()
	fmt.Println("[扫描] 检测已跟踪的常见问题文件...")
	fmt.Println()
	trackedFiles := scanTrackedFiles()
	if len(trackedFiles) == 0 {
		fmt.Println("[信息] 未发现已跟踪的无关文件，.gitignore 工作正常")
		return
	}
	fmt.Println("[发现] 以下文件已跟踪（会被提交到仓库）：")
	fmt.Println()
	for _, f := range trackedFiles {
		fmt.Printf("  - %s\n", f)
	}
	fmt.Println()
	if !confirm("是否删除这些文件（重写历史）") {
		return
	}
	patterns := buildDeletePatterns(trackedFiles)
	fmt.Println()
	fmt.Println("[备份] 创建备份分支 delete-files-backup ...")
	runGit("branch", "delete-files-backup")
	fmt.Println("[完成] 备份分支已创建")
	fmt.Println()
	fmt.Println("[执行] git filter-branch 删除文件...")
	fmt.Println("[提示] 这可能需要一些时间，请耐心等待...")
	fmt.Println()
	var rmCmd string
	if len(patterns) == 1 {
		rmCmd = fmt.Sprintf("git rm --cached --ignore-unmatch %s", patterns[0])
	} else {
		rmCmd = fmt.Sprintf("git rm --cached --ignore-unmatch %s", strings.Join(patterns, " "))
	}
	cmd := exec.Command("git", "filter-branch", "--force", "--index-filter", rmCmd, "--prune-empty", "--", "HEAD")
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println()
		fmt.Println("[失败] 删除失败！")
		fmt.Println("[恢复] 可以从备份分支恢复: git checkout delete-files-backup")
		return
	}
	fmt.Println()
	fmt.Println("[成功] 所有无关文件已从历史中彻底删除！")
	fmt.Println()
	fmt.Println("[下一步] 请选择 [2] 强制覆盖推送")
	fmt.Println("[提示] 本地文件仍然保留，只是从 Git 历史中移除")
}

// ========== 辅助函数 ==========

func findGitRepo(start string) string {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func initRepo() {
	fmt.Println()
	fmt.Println("==========================================")
	fmt.Println("           仓库初始化向导")
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Println("[执行] git init ...")
	if err := runGitShow("init"); err != nil {
		fmt.Println("[错误] git init 失败")
		pause()
		os.Exit(1)
	}
	projectDir, _ = os.Getwd()
	fmt.Println("[信息] 仓库已初始化:", projectDir)
	remoteName = "origin"
	if remoteURL == "" {
		fmt.Println("[错误] 未配置仓库地址，无法初始化")
		pause()
		os.Exit(1)
	}
	fmt.Printf("[执行] git remote add %s %s ...\n", remoteName, remoteURL)
	runGit("remote", "add", remoteName, remoteURL)
	if _, err := os.Stat(".gitignore"); err != nil {
		fmt.Println("[信息] 创建默认 .gitignore ...")
		content := ".gradle/\n/local.properties\n/.idea/\n.DS_Store\n/build/\n/app/build/\n*.apk\n*.tar.gz\n*.zip\n*.bat\ntoken.txt\n"
		os.WriteFile(".gitignore", []byte(content), 0644)
	}
	runGit("add", "-A")
	_, err := runGitOutput("commit", "-m", "Initial commit")
	if err != nil {
		fmt.Println("[提示] 无可提交内容或提交失败")
	} else {
		fmt.Println("[完成] 首次提交已创建")
	}
	fmt.Println("[信息] 仓库初始化完成")
	pause()
}

func readConfig() {
	data, err := os.ReadFile(configFile)
	if err != nil {
		fmt.Println("[提示] 未找到 git-config.txt，自动生成配置模板...")
		// 自动生成空的配置模板
		template := "GITHUB_USERNAME=\nGITHUB_TOKEN=\nREPO_URL=\n"
		if writeErr := os.WriteFile(configFile, []byte(template), 0644); writeErr != nil {
			fmt.Printf("[警告] 无法创建 git-config.txt: %v\n", writeErr)
		} else {
			fmt.Println("[完成] 已生成空配置模板:", configFile)
			fmt.Println("[提示] 请填写 GITHUB_USERNAME、GITHUB_TOKEN 和 REPO_URL 后重新运行")
		}
		return
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "GITHUB_USERNAME=") {
			githubUsername = strings.TrimPrefix(line, "GITHUB_USERNAME=")
		}
		if strings.HasPrefix(line, "GITHUB_TOKEN=") {
			githubToken = strings.TrimPrefix(line, "GITHUB_TOKEN=")
		}
		if strings.HasPrefix(line, "REPO_URL=") {
			remoteURL = strings.TrimPrefix(line, "REPO_URL=")
		}
	}
	fmt.Println("[信息] 已从 git-config.txt 读取认证信息")
}

func buildAuthURL(url, username, token string) string {
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	if idx := strings.Index(url, "@"); idx != -1 {
		url = url[idx+1:]
	}
	return fmt.Sprintf("https://%s:%s@%s", username, token, url)
}

func maskToken(url string) string {
	if idx := strings.Index(url, "ghp_"); idx != -1 {
		end := idx + 4
		for end < len(url) && url[end] != '@' {
			end++
		}
		return url[:idx+4] + "***" + url[end:]
	}
	return url
}

func hasUncommittedChanges() bool {
	cmd := exec.Command("git", "diff-index", "--quiet", "HEAD", "--")
	cmd.Dir = projectDir
	err := cmd.Run()
	return err != nil
}

func showRemoteLog(count int) {
	tempRef := fmt.Sprintf("refs/remotes/_temp_remote/%s", currentBranch)
	runGit("fetch", authURL, currentBranch+":"+tempRef)
	runGitShow("log", "--oneline", fmt.Sprintf("-%d", count), tempRef)
	runGit("update-ref", "-d", tempRef)
}

func showRemoteDiff() {
	tempRef := fmt.Sprintf("refs/remotes/_temp_remote/%s", currentBranch)
	runGit("fetch", authURL, currentBranch+":"+tempRef)
	runGitShow("log", "--oneline", tempRef+"..HEAD")
	runGit("update-ref", "-d", tempRef)
}

func showAuthorsSimple() {
	out := runGit("log", "--format=%an <%ae>")
	lines := strings.Split(out, "\n")
	counts := make(map[string]int)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			counts[line]++
		}
	}
	for author, count := range counts {
		fmt.Printf("  %4d  %s\n", count, author)
	}
}

func scanTrackedFiles() []string {
	var result []string
	patterns := []string{
		"*.exe", "*.dll", "*.so", "*.dylib",
		"*.zip", "*.tar.gz", "*.tar", "*.rar", "*.7z",
		"*.apk", "*.aab", "*.ap_",
		"*.ipa",
		"*.jks", "*.keystore",
		"token.txt", "git-config.txt",
		"release-key.jks", "release-key.txt",
	}
	for _, pattern := range patterns {
		out := runGit("ls-files", pattern)
		if out != "" {
			lines := strings.Split(out, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line != "" {
					result = append(result, line)
				}
			}
		}
	}
	seen := make(map[string]bool)
	var unique []string
	for _, f := range result {
		if !seen[f] {
			seen[f] = true
			unique = append(unique, f)
		}
	}
	return unique
}

func buildDeletePatterns(files []string) []string {
	extMap := make(map[string]bool)
	var singles []string
	for _, f := range files {
		ext := filepath.Ext(f)
		if ext != "" {
			extMap["*"+ext] = true
		} else {
			singles = append(singles, f)
		}
	}
	var patterns []string
	for p := range extMap {
		patterns = append(patterns, p)
	}
	patterns = append(patterns, singles...)
	return patterns
}

func runGit(args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = projectDir
	out, _ := cmd.CombinedOutput()
	return strings.TrimSpace(string(out))
}

func runGitShow(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runGitOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = projectDir
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func gitConfig(scope, key, value string) {
	cmd := exec.Command("git", "config", "--"+scope, key, value)
	cmd.Dir = projectDir
	cmd.Run()
}

func confirm(msg string) bool {
	fmt.Printf("%s [Y/N]: ", msg)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToUpper(input))
	return input == "Y" || input == "YES"
}

func readLine() string {
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func readPassword() string {
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return readLine()
	}
	return string(bytePassword)
}

func pause() {
	fmt.Println()
	fmt.Print("按任意键继续...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
