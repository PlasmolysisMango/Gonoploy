// Package game - menu.go 主菜单 / 联机大厅 / 等待页面的更新与绘制。
//
// 设计基调：「Midnight Pavilion」—— 深夜钴蓝渐变背景上洒落的星点，
// 由古铜金线条构成 Art Deco 风格的边框装饰，搭配大字号衬线标题。
// 强调克制的动画（光标闪烁、星点微光、选中项呼吸），让大厅在静默中
// 仍透出仪式感，而不是充满抖动的商业 banner。
package game

import (
	"fmt"
	"image/color"
	"math"
	"strconv"
	"strings"

	asset "github.com/PlasmolysisMango/Gonopoly/asset/pics"
	assetfont "github.com/PlasmolysisMango/Gonopoly/asset/font"
	"github.com/PlasmolysisMango/Gonopoly/internal/network"
	"github.com/PlasmolysisMango/Gonopoly/internal/render"
	"github.com/PlasmolysisMango/Gonopoly/pkg/protocol"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ===== 调色板 =====
//
// 集中在文件顶部，便于整体配色微调。所有界面元素都从这套色板取色。
var (
	colInkTop    = color.RGBA{8, 11, 32, 255}    // 背景顶部
	colInkBot    = color.RGBA{26, 17, 71, 255}   // 背景底部
	colPanel     = color.RGBA{14, 18, 50, 220}   // 卡片底色（半透明）
	colPanelDeep = color.RGBA{8, 11, 30, 255}    // 输入框内部
	colGold      = color.RGBA{212, 168, 67, 255} // 主金
	colGoldHi    = color.RGBA{245, 213, 110, 255}// 高亮金（选中/标题）
	colGoldDim   = color.RGBA{120, 95, 38, 255}  // 暗金（次要）
	colTeal      = color.RGBA{78, 201, 176, 255} // 成功/已就绪
	colCrimson   = color.RGBA{212, 82, 78, 255}  // 错误
	colCream     = color.RGBA{232, 230, 211, 255}// 主文字
	colMute      = color.RGBA{120, 130, 165, 255}// 提示文字
	colStarA     = color.RGBA{220, 220, 240, 80}
	colStarB     = color.RGBA{255, 232, 170, 110}
)

// ===== 主菜单 =====

func (g *Game) updateMainMenu() {
	g.menuTick++
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) || inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		g.menuCursor = 1 - g.menuCursor
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDigit1) {
		g.menuCursor = 0
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDigit2) {
		g.menuCursor = 1
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		switch g.menuCursor {
		case 0:
			g.mode = ModeLocal
			g.state = StateCharSelect
		case 1:
			g.mode = ModeOnline
			g.state = StateLobby
			g.lobbyMessage = ""
			g.lobbyField = 0
			g.lobbyAction = 0
			if g.lobbyCharName == "" {
				if names := asset.CharacterNames(); len(names) > 0 {
					g.lobbyCharName = names[0]
				}
			}
		}
	}
}

func (g *Game) drawMainMenu(screen *ebiten.Image) {
	drawMenuBackdrop(screen, g.menuTick)

	cx := float32(render.ScreenWidth) / 2

	// 顶部小标志条
	drawCenteredText(screen, g.textHint, "—  GONOPLOY  ·  学园都市经营·策略对战  —",
		colGoldDim, cx, 110)

	// Hero 标题
	drawCenteredText(screen, g.textTitle, "GONOPLOY", colGoldHi, cx, 150)

	// 标题下水平金线 + 中央小菱形
	drawHorizontalAccent(screen, cx, 240, 420)

	// 副标题
	drawCenteredText(screen, g.textBody, "异 能  ·  地 产  ·  风 云 博 弈",
		colCream, cx, 260)

	// 选项面板
	pw, ph := float32(620), float32(240)
	px := cx - pw/2
	py := float32(380)
	drawPanel(screen, px, py, pw, ph, true)

	options := []struct {
		key, label, desc string
	}{
		{"1", "本地游戏", "与 AI 对手共同作战"},
		{"2", "联机游戏", "连接服务器，挑战真人玩家"},
	}
	for i, opt := range options {
		rowY := py + 28 + float32(i)*92
		selected := i == g.menuCursor

		// 高亮指示条（选中时呼吸）
		if selected {
			pulse := 0.5 + 0.5*math.Sin(float64(g.menuTick)*0.12)
			alpha := uint8(120 + int(120*pulse))
			vector.DrawFilledRect(screen, px+18, rowY+6, 5, 64,
				color.RGBA{245, 213, 110, alpha}, false)
		}

		// 编号方框
		bx, by := px+44, rowY
		boxClr := colGoldDim
		labelClr := colCream
		descClr := colMute
		if selected {
			boxClr = colGoldHi
			labelClr = colGoldHi
			descClr = colCream
		}
		vector.StrokeRect(screen, bx, by, 50, 50, 1, boxClr, false)
		drawCenteredText(screen, g.textHead, opt.key, boxClr, bx+25, by+8)

		// 选项文字
		drawText(screen, g.textHead, opt.label, labelClr, bx+72, rowY+2)
		drawText(screen, g.textHint, opt.desc, descClr, bx+72, rowY+44)
	}

	// 底部键位提示栏
	hintY := float32(740)
	hints := []struct{ key, desc string }{
		{"↑↓", "选择"},
		{"Enter", "确认"},
		{"1 / 2", "快捷选择"},
	}
	totalW := keyHintRowWidth(g.textHint, hints)
	hx := cx - totalW/2
	for _, h := range hints {
		hx = drawKeyCap(screen, g.textHint, h.key, h.desc, hx, hintY) + 18
	}

	// 版本水印
	drawText(screen, g.textHint, "v0.14 · midnight pavilion",
		colGoldDim, 40, float32(render.ScreenHeight-40))
}

// ===== 联机大厅 =====

const (
	lobbyFieldServer = iota
	lobbyFieldName
	lobbyFieldRoom
	lobbyFieldChar
)

var lobbyFieldLabels = []string{"服务器地址", "玩家名字", "房间 ID", "角色"}
var lobbyFieldHints = []string{
	"WebSocket 地址，例如 ws://127.0.0.1:8080/ws",
	"将显示给其他玩家的昵称",
	"加入房间时必填；创建房间可留空",
	"使用 ← / → 在角色列表间切换",
}

func (g *Game) updateLobby() {
	g.menuTick++

	// 轮询服务器推送（如果已连接）——以便接收 S2CRoomList 。
	if g.netClient != nil {
		for {
			msg, ok := g.netClient.Poll()
			if !ok {
				break
			}
			g.handleLobbyMessage(*msg)
		}
	}

	// Esc 返回
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.state = StateMainMenu
		g.lobbyMessage = ""
		if g.netClient != nil {
			g.netClient.Close()
			g.netClient = nil
		}
		return
	}

	// F1/F2 创建/加入模式切换（同时把焦点拉回表单）
	if inpututil.IsKeyJustPressed(ebiten.KeyF1) {
		g.lobbyAction = 0
		g.lobbyFocus = 0
		g.lobbyMessage = "已选择：创建房间"
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF2) {
		// 若焦点在房间列表上且有选中项，F2 直接加入该房间
		if g.lobbyFocus == 1 && len(g.roomList) > 0 {
			g.joinSelectedRoom()
		} else {
			g.lobbyAction = 1
			g.lobbyFocus = 0
			g.lobbyMessage = "已选择：加入房间"
		}
	}

	// F3 刷新房间列表（必要时先建立连接）
	if inpututil.IsKeyJustPressed(ebiten.KeyF3) {
		g.refreshRoomList()
	}

	// Tab 在两个面板间切换焦点：表单、房间列表
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			g.lobbyFocus = (g.lobbyFocus + 1) % 2
		} else {
			if g.lobbyFocus == 0 {
				g.lobbyField = (g.lobbyField + 1) % len(lobbyFieldLabels)
			} else if len(g.roomList) > 0 {
				g.roomListIdx = (g.roomListIdx + 1) % len(g.roomList)
			}
		}
	}

	if g.lobbyFocus == 1 {
		// 房间列表区：上下选择，Enter 加入
		g.updateRoomListPanel()
		return
	}

	// 表单区：上下切换字段
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		g.lobbyField = (g.lobbyField - 1 + len(lobbyFieldLabels)) % len(lobbyFieldLabels)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		g.lobbyField = (g.lobbyField + 1) % len(lobbyFieldLabels)
	}

	// 角色字段使用左右
	if g.lobbyField == lobbyFieldChar {
		names := asset.CharacterNames()
		if len(names) > 0 {
			cur := indexOf(names, g.lobbyCharName)
			if cur < 0 {
				cur = 0
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyLeft) {
				cur = (cur - 1 + len(names)) % len(names)
				g.lobbyCharName = names[cur]
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyRight) {
				cur = (cur + 1) % len(names)
				g.lobbyCharName = names[cur]
			}
		}
	} else {
		g.handleLobbyTextInput()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		g.submitLobby()
	}
}

// updateRoomListPanel 处理房间列表区输入。调用前已确认 lobbyFocus==1。
func (g *Game) updateRoomListPanel() {
	if len(g.roomList) > 0 {
		if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
			g.roomListIdx = (g.roomListIdx - 1 + len(g.roomList)) % len(g.roomList)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
			g.roomListIdx = (g.roomListIdx + 1) % len(g.roomList)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		g.joinSelectedRoom()
	}
}

// refreshRoomList 向服务器请求最新的房间列表。
// 如果还未连接，则使用大厅填写的服务器地址先建立连接。
func (g *Game) refreshRoomList() {
	if strings.TrimSpace(g.lobbyServer) == "" {
		g.lobbyMessage = "请填写服务器地址"
		return
	}
	if g.netClient == nil || g.netClient.State() == network.StateDisconnected {
		nc := network.NewNetClient(strings.TrimSpace(g.lobbyServer))
		if err := nc.Connect(); err != nil {
			g.lobbyMessage = "连接失败: " + err.Error()
			return
		}
		g.netClient = nc
	}
	if err := g.netClient.Send(protocol.TypeListRooms, &protocol.C2SListRooms{}); err != nil {
		g.lobbyMessage = "刷新失败: " + err.Error()
		return
	}
	g.lobbyMessage = "正在刷新房间列表…"
	g.lobbyFocus = 1
}

// joinSelectedRoom 加入当前选中的房间。如果该房间游戏已开始，则转为观战。
func (g *Game) joinSelectedRoom() {
	if g.roomListIdx < 0 || g.roomListIdx >= len(g.roomList) {
		g.lobbyMessage = "请先选中一个房间"
		return
	}
	room := g.roomList[g.roomListIdx]
	if room.Started {
		// 诸表示已开始的房间，切换为观战模式
		g.lobbyRoomID = room.RoomID
		g.lobbyAction = 2
		g.lobbyMessage = "加入观战: " + room.RoomID
		g.submitLobby()
		return
	}
	if room.PlayerCount >= room.MaxPlayers {
		g.lobbyMessage = "该房间已满"
		return
	}
	g.lobbyRoomID = room.RoomID
	g.lobbyAction = 1
	g.submitLobby()
}

// handleLobbyMessage 在联机大厅中处理服务器推送的与大厅相关的消息。
func (g *Game) handleLobbyMessage(msg protocol.Message) {
	switch msg.Type {
	case protocol.TypeRoomList:
		list, err := protocol.ParsePayload[protocol.S2CRoomList](msg)
		if err != nil {
			g.lobbyMessage = "解析房间列表失败"
			return
		}
		g.roomList = list.Rooms
		if g.roomListIdx >= len(g.roomList) {
			g.roomListIdx = 0
		}
		g.roomListAt = int64(g.menuTick)
		if len(g.roomList) == 0 {
			g.lobbyMessage = "暂无可用房间，可创建一个"
		} else {
			g.lobbyMessage = fmt.Sprintf("已拉取 %d 个房间", len(g.roomList))
		}
	case protocol.TypeError:
		ev, err := protocol.ParsePayload[protocol.S2CError](msg)
		if err == nil {
			g.lobbyMessage = "错误: " + ev.Message
		}
	}
}

// handleLobbyTextInput 处理键盘文本输入：可见 ASCII + Backspace。
func (g *Game) handleLobbyTextInput() {
	target := g.lobbyTargetField()
	if target == nil {
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		// 按字符（rune）退格，避免拆坏多字节字符
		if len(*target) > 0 {
			r := []rune(*target)
			*target = string(r[:len(r)-1])
		}
		return
	}
	runes := ebiten.AppendInputChars(nil)
	for _, r := range runes {
		if r >= 32 && r < 127 {
			*target += string(r)
		}
	}
}

func (g *Game) lobbyTargetField() *string {
	switch g.lobbyField {
	case lobbyFieldServer:
		return &g.lobbyServer
	case lobbyFieldName:
		return &g.lobbyName
	case lobbyFieldRoom:
		return &g.lobbyRoomID
	}
	return nil
}

func (g *Game) submitLobby() {
	if strings.TrimSpace(g.lobbyServer) == "" {
		g.lobbyMessage = "请填写服务器地址"
		return
	}
	if strings.TrimSpace(g.lobbyName) == "" {
		g.lobbyMessage = "请填写玩家名字"
		return
	}
	if (g.lobbyAction == 1 || g.lobbyAction == 2) && strings.TrimSpace(g.lobbyRoomID) == "" {
		if g.lobbyAction == 2 {
			g.lobbyMessage = "观战房间需要房间 ID"
		} else {
			g.lobbyMessage = "加入房间需要房间 ID"
		}
		return
	}
	if g.lobbyAction != 2 && g.lobbyCharName == "" {
		g.lobbyMessage = "请选择角色"
		return
	}

	if g.netClient != nil && g.netClient.State() == network.StateConnected {
		// 例如在 F3 浏览后复用连接，避免重复 Connect
	} else {
		if g.netClient != nil {
			g.netClient.Close()
			g.netClient = nil
		}
		nc := network.NewNetClient(strings.TrimSpace(g.lobbyServer))
		if err := nc.Connect(); err != nil {
			g.lobbyMessage = "连接失败: " + err.Error()
			return
		}
		g.netClient = nc
	}
	nc := g.netClient
	g.myName = strings.TrimSpace(g.lobbyName)

	if g.lobbyAction == 0 {
		err := nc.Send(protocol.TypeCreateRoom, &protocol.C2SCreateRoom{
			PlayerName: g.myName,
			CharName:   g.lobbyCharName,
		})
		if err != nil {
			g.lobbyMessage = "发送失败: " + err.Error()
			return
		}
		g.isSpectator = false
	} else if g.lobbyAction == 1 {
		err := nc.Send(protocol.TypeJoinRoom, &protocol.C2SJoinRoom{
			RoomID:     strings.TrimSpace(g.lobbyRoomID),
			PlayerName: g.myName,
			CharName:   g.lobbyCharName,
		})
		if err != nil {
			g.lobbyMessage = "发送失败: " + err.Error()
			return
		}
		g.isSpectator = false
	} else {
		// 观战模式
		err := nc.Send(protocol.TypeSpectate, &protocol.C2SSpectate{
			RoomID:     strings.TrimSpace(g.lobbyRoomID),
			PlayerName: g.myName,
		})
		if err != nil {
			g.lobbyMessage = "发送失败: " + err.Error()
			return
		}
		g.isSpectator = true
	}

	g.waitingPlayers = nil
	g.lobbyReady = false
	g.lobbyMessage = "已发送请求，等待服务器响应..."
	g.state = StateWaiting
}

func (g *Game) drawLobby(screen *ebiten.Image) {
	drawMenuBackdrop(screen, g.menuTick)

	cx := float32(render.ScreenWidth) / 2

	// 顶部面包屑
	drawCenteredText(screen, g.textHint,
		"GONOPLOY  /  联机大厅", colGoldDim, cx, 50)

	// 大标题
	drawCenteredText(screen, g.textTitle, "联机大厅", colGoldHi, cx, 78)
	drawHorizontalAccent(screen, cx, 168, 320)

	// 三个模式 Tab：创建 / 加入 / 浏览
	mY := float32(190)
	tabW := float32(200)
	gap := float32(12)
	x0 := cx - (tabW*3+gap*2)/2
	drawTab(screen, g.textBody, "F1   创建房间",
		x0, mY, tabW, 46, g.lobbyFocus == 0 && g.lobbyAction == 0, g.menuTick)
	drawTab(screen, g.textBody, "F2   加入房间",
		x0+tabW+gap, mY, tabW, 46, g.lobbyFocus == 0 && g.lobbyAction == 1, g.menuTick)
	drawTab(screen, g.textBody, "F3   浏览房间",
		x0+(tabW+gap)*2, mY, tabW, 46, g.lobbyFocus == 1, g.menuTick)

	// 双栏布局：左表单、右房间列表
	panelW := float32(720)
	panelH := float32(360)
	panelY := float32(270)
	colGap := float32(40)
	formX := cx - panelW - colGap/2
	listX := cx + colGap/2

	g.drawLobbyFormPanel(screen, formX, panelY, panelW, panelH)
	g.drawLobbyRoomListPanel(screen, listX, panelY, panelW, panelH)

	// 连接状态徽章
	statusY := panelY + panelH + 18
	statusText, statusClr := g.connectionStatus()
	drawStatusBadge(screen, g.textHint, statusText, statusClr, cx, statusY)

	// 大厅消息
	if g.lobbyMessage != "" {
		drawCenteredText(screen, g.textBody, g.lobbyMessage,
			lobbyMessageColor(g.lobbyMessage), cx, statusY+34)
	}

	// 底部键位
	hintY := float32(820)
	hints := []struct{ key, desc string }{
		{"Tab", "下一项"},
		{"↑↓", "选项"},
		{"←→", "切角色"},
		{"F3", "刷新"},
		{"Enter", "提交/加入"},
		{"Esc", "返回"},
	}
	totalW := keyHintRowWidth(g.textHint, hints)
	hx := cx - totalW/2
	for _, h := range hints {
		hx = drawKeyCap(screen, g.textHint, h.key, h.desc, hx, hintY) + 18
	}
}

// drawLobbyFormPanel 绘制表单区：服务器、名字、房间 ID、角色。
func (g *Game) drawLobbyFormPanel(screen *ebiten.Image, px, py, pw, ph float32) {
	drawPanel(screen, px, py, pw, ph, g.lobbyFocus == 0)

	// 面板标题
	drawText(screen, g.textBody, "表单信息", colGoldHi, px+30, py+12)
	vector.DrawFilledRect(screen, px+30, py+44, pw-60, 1, colGoldDim, false)

	values := []string{g.lobbyServer, g.lobbyName, g.lobbyRoomID, g.lobbyCharName}
	for i, label := range lobbyFieldLabels {
		row := py + 60 + float32(i)*72
		focused := g.lobbyFocus == 0 && i == g.lobbyField

		// 序号
		drawText(screen, g.textHint, "0"+strconv.Itoa(i+1),
			colGoldDim, px+24, row)

		// 字段名
		labelClr := colMute
		if focused {
			labelClr = colGoldHi
		}
		drawText(screen, g.textBody, label, labelClr, px+62, row-3)

		// 字段提示
		drawText(screen, g.textHint, lobbyFieldHints[i], colMute, px+220, row+5)

		// 输入框
		boxX := px + 24
		boxY := row + 26
		boxW := pw - 48
		boxH := float32(30)
		boxBorder := colGoldDim
		if focused {
			boxBorder = colGoldHi
		}
		vector.DrawFilledRect(screen, boxX, boxY, boxW, boxH, colPanelDeep, false)
		vector.StrokeRect(screen, boxX, boxY, boxW, boxH, 1, boxBorder, false)

		val := values[i]
		if i == lobbyFieldChar {
			arrowClr := colGoldDim
			if focused {
				arrowClr = colGoldHi
			}
			drawText(screen, g.textBody, "<", arrowClr, boxX+12, boxY+1)
			drawText(screen, g.textBody, ">", arrowClr, boxX+boxW-22, boxY+1)
			drawCenteredText(screen, g.textBody, val, colCream,
				boxX+boxW/2, boxY+1)
		} else {
			drawText(screen, g.textBody, val, colCream, boxX+12, boxY+1)
			if focused && (g.menuTick/30)%2 == 0 {
				vw := measureText(g.textBody, val)
				vector.DrawFilledRect(screen, boxX+12+float32(vw)+1, boxY+6,
					2, boxH-12, colGoldHi, false)
			}
		}
	}
}

// drawLobbyRoomListPanel 绘制可用房间列表面板。
func (g *Game) drawLobbyRoomListPanel(screen *ebiten.Image, px, py, pw, ph float32) {
	drawPanel(screen, px, py, pw, ph, g.lobbyFocus == 1)

	// 面板标题 + 计数
	title := "可用房间"
	drawText(screen, g.textBody, title, colGoldHi, px+30, py+12)
	countTxt := fmt.Sprintf("%d 个房间", len(g.roomList))
	tw := measureText(g.textHint, countTxt)
	drawText(screen, g.textHint, countTxt, colGoldDim,
		px+pw-30-float32(tw), py+18)
	vector.DrawFilledRect(screen, px+30, py+44, pw-60, 1, colGoldDim, false)

	// 列头
	headerY := py + 54
	drawText(screen, g.textHint, "ROOM ID", colGoldDim, px+44, headerY)
	drawText(screen, g.textHint, "玩家", colGoldDim, px+250, headerY)
	drawText(screen, g.textHint, "状态", colGoldDim, px+pw-150, headerY)
	vector.DrawFilledRect(screen, px+30, headerY+22, pw-60, 1,
		color.RGBA{120, 95, 38, 120}, false)

	// 空状态
	if len(g.roomList) == 0 {
		drawCenteredText(screen, g.textHint,
			"暂无可用房间，按 F3 刷新或创建一个",
			colMute, px+pw/2, py+ph/2-10)
		return
	}

	// 表体区域
	rowH := float32(34)
	rowsTop := headerY + 32
	maxRows := int((py + ph - 12 - rowsTop) / rowH)
	if maxRows <= 0 {
		maxRows = 1
	}
	start := 0
	if g.roomListIdx >= maxRows {
		start = g.roomListIdx - maxRows + 1
	}
	end := start + maxRows
	if end > len(g.roomList) {
		end = len(g.roomList)
	}

	for i := start; i < end; i++ {
		r := g.roomList[i]
		row := rowsTop + float32(i-start)*rowH
		selected := i == g.roomListIdx && g.lobbyFocus == 1

		// 交错底纹
		bgAlpha := uint8(20)
		if (i-start)%2 == 1 {
			bgAlpha = 50
		}
		vector.DrawFilledRect(screen, px+30, row-4, pw-60, rowH-2,
			color.RGBA{255, 255, 255, bgAlpha}, false)

		// 选中指示条（呼吸闪烁）
		if selected {
			pulse := 0.55 + 0.45*math.Sin(float64(g.menuTick)*0.16)
			alpha := uint8(140 + int(110*pulse))
			vector.DrawFilledRect(screen, px+30, row-4, 4, rowH-2,
				color.RGBA{245, 213, 110, alpha}, false)
			// 全行高亮描边
			vector.StrokeRect(screen, px+30, row-4, pw-60, rowH-2, 1,
				colGoldHi, false)
		}

		// 房间 ID
		idClr := colCream
		if selected {
			idClr = colGoldHi
		}
		drawText(screen, g.textBody, r.RoomID, idClr, px+44, row-3)

		// 玩家数 / 上限
		counter := fmt.Sprintf("%d / %d", r.PlayerCount, r.MaxPlayers)
		counterClr := colCream
		if r.PlayerCount >= r.MaxPlayers {
			counterClr = colCrimson
		}
		drawText(screen, g.textBody, counter, counterClr, px+250, row-3)

		// 状态点 + 文字
		statusTxt := "等待中"
		statusClr := colTeal
		if r.Started {
			statusTxt = "游戏中"
			statusClr = colGoldDim
		} else if r.PlayerCount >= r.MaxPlayers {
			statusTxt = "已满"
			statusClr = colCrimson
		}
		vector.DrawFilledCircle(screen, px+pw-160, row+8, 4, statusClr, false)
		drawText(screen, g.textBody, statusTxt, statusClr, px+pw-150, row-3)
	}

	// 底部提示与滚动指示
	if end < len(g.roomList) || start > 0 {
		indicator := fmt.Sprintf("↓ 显示 %d-%d / 共 %d", start+1, end, len(g.roomList))
		drawCenteredText(screen, g.textHint, indicator,
			colGoldDim, px+pw/2, py+ph-20)
	}
	if g.roomListAt > 0 {
		ageTicks := int(int64(g.menuTick) - g.roomListAt)
		if ageTicks < 0 {
			ageTicks = 0
		}
		ageSec := ageTicks / 60
		ageTxt := fmt.Sprintf("刷新于 %ds 前", ageSec)
		drawText(screen, g.textHint, ageTxt, colMute, px+30, py+ph-22)
	}
}

// ===== 等待页 =====

func (g *Game) updateWaiting() {
	g.menuTick++

	// 持续轮询服务器消息
	if g.netClient != nil {
		for {
			msg, ok := g.netClient.Poll()
			if !ok {
				break
			}
			g.handleServerMessage(*msg)
		}
	}

	// R 键标记就绪
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		if g.netClient != nil && !g.lobbyReady {
			if err := g.netClient.Send(protocol.TypeReady, &protocol.C2SReady{}); err == nil {
				g.lobbyReady = true
				g.lobbyMessage = "已点亮准备灯"
			} else {
				g.lobbyMessage = "发送失败：" + err.Error()
			}
		}
	}

	// Esc 离开
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g.netClient != nil {
			g.netClient.Close()
			g.netClient = nil
		}
		g.lobbyReady = false
		g.state = StateMainMenu
	}
}

func (g *Game) drawWaiting(screen *ebiten.Image) {
	drawMenuBackdrop(screen, g.menuTick)

	cx := float32(render.ScreenWidth) / 2

	// 标题（带跳动的省略号）
	dots := strings.Repeat(".", 1+(g.menuTick/30)%3)
	drawCenteredText(screen, g.textHint,
		"GONOPLOY  /  联机大厅  /  房间内", colGoldDim, cx, 50)
	drawCenteredText(screen, g.textTitle,
		"等待玩家加入"+dots, colGoldHi, cx, 78)
	drawHorizontalAccent(screen, cx, 170, 320)

	// 房间号大徽章
	drawRoomBadge(screen, g.textHint, g.textHead, g.roomID, cx, 200, g.menuTick)

	// 玩家列表面板
	pw, ph := float32(840), float32(380)
	px := cx - pw/2
	py := float32(330)
	drawPanel(screen, px, py, pw, ph, true)

	drawText(screen, g.textBody, "房间成员",
		colGoldHi, px+30, py+18)
	// 头部分隔
	vector.DrawFilledRect(screen, px+30, py+52, pw-60, 1, colGoldDim, false)

	if len(g.waitingPlayers) == 0 {
		drawCenteredText(screen, g.textHint,
			"暂无玩家信息，等待服务器同步…", colMute, cx, py+ph/2)
	}

	allReady := len(g.waitingPlayers) > 0
	for i, line := range g.waitingPlayers {
		row := py + 70 + float32(i)*52
		// 行底纹
		bgAlpha := uint8(30)
		if i%2 == 1 {
			bgAlpha = 60
		}
		vector.DrawFilledRect(screen, px+30, row-6, pw-60, 44,
			color.RGBA{255, 255, 255, bgAlpha}, false)

		// 解析行：name [char] (status)
		name, char, status := parseWaitingLine(line)
		// 状态点
		dotClr := colCrimson
		switch status {
		case "已就绪":
			dotClr = colTeal
		case "未就绪":
			dotClr = colGoldDim
			allReady = false
		case "离线":
			dotClr = colMute
			allReady = false
		default:
			allReady = false
		}
		vector.DrawFilledCircle(screen, px+50, row+10, 6, dotClr, false)

		// 编号
		drawText(screen, g.textHint, "P"+strconv.Itoa(i+1),
			colGoldDim, px+70, row+2)

		// 玩家名字
		nameClr := colCream
		if name == g.myName {
			nameClr = colGoldHi
		}
		drawText(screen, g.textBody, name, nameClr, px+120, row-2)

		// 角色
		if char != "" {
			drawText(screen, g.textHint, "饰演 "+char, colMute, px+340, row+4)
		}

		// 状态文字
		drawText(screen, g.textBody, status, dotClr, px+pw-150, row-2)

		// 自己的回合，叠加"你"标记
		if name == g.myName {
			drawText(screen, g.textHint, "(你)", colGoldHi, px+120+float32(measureText(g.textBody, name))+8, row+4)
		}
	}

	// 全员就绪提示
	if allReady && len(g.waitingPlayers) >= 2 {
		alpha := uint8(180 + int(60*math.Sin(float64(g.menuTick)*0.18)))
		clr := color.RGBA{78, 201, 176, alpha}
		drawCenteredText(screen, g.textHead,
			"★ 全员就绪 · 游戏即将开始 ★", clr, cx, py+ph+24)
	} else if g.lobbyMessage != "" {
		drawCenteredText(screen, g.textBody, g.lobbyMessage,
			lobbyMessageColor(g.lobbyMessage), cx, py+ph+24)
	}

	// 底部键位
	hintY := float32(820)
	hints := []struct{ key, desc string }{
		{"R", "准备"},
		{"Esc", "离开房间"},
	}
	totalW := keyHintRowWidth(g.textHint, hints)
	hx := cx - totalW/2
	for _, h := range hints {
		hx = drawKeyCap(screen, g.textHint, h.key, h.desc, hx, hintY) + 18
	}

	// 已点过准备的标记
	if g.lobbyReady {
		drawText(screen, g.textHint, "● 你已准备",
			colTeal, 50, float32(render.ScreenHeight-40))
	}
}

// ===== 通用渲染辅助 =====

// drawMenuBackdrop 绘制大厅/菜单的统一背景：
//   1. 垂直渐变（ink → 深紫蓝）
//   2. 倾斜的金色细网格
//   3. 散布的"星点"，按 tick 微弱呼吸
//   4. 四角的金色 L 形装饰条
func drawMenuBackdrop(screen *ebiten.Image, tick int) {
	w := float32(render.ScreenWidth)
	h := float32(render.ScreenHeight)

	// 1. 渐变（按行分块绘制）
	bands := 24
	for i := 0; i < bands; i++ {
		t := float64(i) / float64(bands-1)
		clr := lerpColor(colInkTop, colInkBot, t)
		y := float32(i) * h / float32(bands)
		vector.DrawFilledRect(screen, 0, y, w, h/float32(bands)+1, clr, false)
	}

	// 2. 倾斜网格（从右上到左下）
	gridClr := color.RGBA{255, 255, 255, 8}
	for x := -int(h); x < int(w); x += 64 {
		vector.StrokeLine(screen, float32(x), 0, float32(x)+h, h, 1, gridClr, false)
	}

	// 3. 星点（伪随机但稳定）
	for i := 0; i < 90; i++ {
		x := float32((i*97 + 31) % int(w))
		y := float32((i*53 + 17) % int(h))
		size := float32(1 + ((i*7) % 3))
		// 呼吸
		base := 70.0 + 50.0*math.Sin(float64(tick)*0.04+float64(i)*0.7)
		if base < 30 {
			base = 30
		}
		clr := colStarA
		if i%5 == 0 {
			clr = colStarB
		}
		clr.A = uint8(base)
		vector.DrawFilledCircle(screen, x, y, size, clr, false)
	}

	// 4. 角部 L 形装饰
	drawCornerOrnaments(screen, 28, 28, w-28, h-28)
}

func drawCornerOrnaments(screen *ebiten.Image, x1, y1, x2, y2 float32) {
	clr := color.RGBA{212, 168, 67, 200}
	armLen := float32(48)
	thick := float32(2)
	// top-left
	vector.DrawFilledRect(screen, x1, y1, armLen, thick, clr, false)
	vector.DrawFilledRect(screen, x1, y1, thick, armLen, clr, false)
	// top-right
	vector.DrawFilledRect(screen, x2-armLen, y1, armLen, thick, clr, false)
	vector.DrawFilledRect(screen, x2-thick, y1, thick, armLen, clr, false)
	// bottom-left
	vector.DrawFilledRect(screen, x1, y2-thick, armLen, thick, clr, false)
	vector.DrawFilledRect(screen, x1, y2-armLen, thick, armLen, clr, false)
	// bottom-right
	vector.DrawFilledRect(screen, x2-armLen, y2-thick, armLen, thick, clr, false)
	vector.DrawFilledRect(screen, x2-thick, y2-armLen, thick, armLen, clr, false)

	// 四角额外的小菱形点缀
	for _, p := range [][2]float32{
		{x1 + armLen + 8, y1 + 1},
		{x2 - armLen - 8, y1 + 1},
		{x1 + armLen + 8, y2 - 1},
		{x2 - armLen - 8, y2 - 1},
	} {
		drawDiamond(screen, p[0], p[1], 4, clr)
	}
}

// drawPanel 绘制一个带双线金边的半透明卡片。
func drawPanel(screen *ebiten.Image, x, y, w, h float32, accent bool) {
	// 半透明底
	vector.DrawFilledRect(screen, x, y, w, h, colPanel, false)
	// 外边框
	outer := colGoldDim
	if accent {
		outer = colGold
	}
	vector.StrokeRect(screen, x, y, w, h, 2, outer, false)
	// 内细线
	vector.StrokeRect(screen, x+5, y+5, w-10, h-10, 1,
		color.RGBA{60, 50, 22, 160}, false)
	// 顶部小条形
	stripe := color.RGBA{212, 168, 67, 60}
	vector.DrawFilledRect(screen, x+12, y+1, w-24, 2, stripe, false)
}

// drawTab 绘制一个 Tab 风格按钮。
func drawTab(screen *ebiten.Image, cache *assetfont.TextCache,
	label string, x, y, w, h float32, active bool, tick int) {
	fill := color.RGBA{8, 11, 30, 255}
	border := colGoldDim
	textClr := colMute
	if active {
		// 微弱呼吸的金色填充
		pulse := uint8(50 + int(40*math.Sin(float64(tick)*0.1)))
		fill = color.RGBA{60, 45, 18, pulse + 80}
		border = colGoldHi
		textClr = colGoldHi
	}
	vector.DrawFilledRect(screen, x, y, w, h, fill, false)
	vector.StrokeRect(screen, x, y, w, h, 2, border, false)
	if active {
		// 底部高亮线
		vector.DrawFilledRect(screen, x+4, y+h-3, w-8, 2, colGoldHi, false)
	}
	drawCenteredText(screen, cache, label, textClr, x+w/2, y+h/2-9)
}

// drawHorizontalAccent 标题下的横向金线 + 中央菱形。
func drawHorizontalAccent(screen *ebiten.Image, cx, y, length float32) {
	half := length / 2
	vector.DrawFilledRect(screen, cx-half, y, length, 1, colGold, false)
	drawDiamond(screen, cx, y+1, 6, colGoldHi)
	// 两侧的小菱形
	drawDiamond(screen, cx-half-12, y+1, 4, colGoldDim)
	drawDiamond(screen, cx+half+12, y+1, 4, colGoldDim)
}

// drawDiamond 用四条线段（实心三角）绘制一个菱形。
// 这里用较粗的填充矩形旋转近似不太好，改用四个小三角形堆叠：
// 简化做法是绘制一个边长近似的填充矩形再加四个小三角，但 Ebiten 没有
// 直接 fill triangle 的方法；这里用 DrawFilledCircle 的小尺寸近似。
func drawDiamond(screen *ebiten.Image, cx, cy, r float32, clr color.RGBA) {
	// 用堆叠的水平细线绘制菱形：从顶到底，宽度先增后减。
	for d := -r; d <= r; d++ {
		w := r - float32(math.Abs(float64(d)))
		vector.DrawFilledRect(screen, cx-w, cy+d, w*2, 1, clr, false)
	}
}

// drawCenteredText 把缓存文字图像水平居中绘制于 cx 处。
func drawCenteredText(screen *ebiten.Image, cache *assetfont.TextCache,
	text string, clr color.Color, cx, y float32) int {
	img := cache.GetImage(text, clr)
	w := img.Bounds().Dx()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(cx)-float64(w)/2, float64(y))
	screen.DrawImage(img, op)
	return w
}

// drawText 在 (x, y) 绘制一段文字，返回宽度，便于排版。
func drawText(screen *ebiten.Image, cache *assetfont.TextCache,
	text string, clr color.Color, x, y float32) int {
	img := cache.GetImage(text, clr)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	screen.DrawImage(img, op)
	return img.Bounds().Dx()
}

// measureText 测量文字宽度（不绘制）。
func measureText(cache *assetfont.TextCache, text string) int {
	if text == "" {
		return 0
	}
	return cache.GetImage(text, colCream).Bounds().Dx()
}

// drawKeyCap 绘制 [Key] 描述 形式的提示，并返回光标移动后的 x 坐标。
func drawKeyCap(screen *ebiten.Image, cache *assetfont.TextCache,
	key, desc string, x, y float32) float32 {
	keyImg := cache.GetImage(key, colGoldHi)
	kw := float32(keyImg.Bounds().Dx())
	kh := float32(keyImg.Bounds().Dy())
	padX := float32(8)
	padY := float32(3)
	bw := kw + padX*2
	bh := kh + padY*2
	// 键帽底
	vector.DrawFilledRect(screen, x, y, bw, bh,
		color.RGBA{30, 28, 60, 220}, false)
	vector.StrokeRect(screen, x, y, bw, bh, 1, colGold, false)
	// 字
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x+padX), float64(y+padY))
	screen.DrawImage(keyImg, op)
	// 描述
	dx := x + bw + 6
	descImg := cache.GetImage(desc, colCream)
	dop := &ebiten.DrawImageOptions{}
	dop.GeoM.Translate(float64(dx), float64(y+padY))
	screen.DrawImage(descImg, dop)
	return dx + float32(descImg.Bounds().Dx())
}

// keyHintRowWidth 估算一行键位提示的总宽度，用于居中排版。
func keyHintRowWidth(cache *assetfont.TextCache,
	hints []struct{ key, desc string }) float32 {
	total := float32(0)
	for i, h := range hints {
		kImg := cache.GetImage(h.key, colGoldHi)
		dImg := cache.GetImage(h.desc, colCream)
		total += float32(kImg.Bounds().Dx()) + 16 // 键帽内边距 8*2
		total += 6 + float32(dImg.Bounds().Dx())
		if i < len(hints)-1 {
			total += 18
		}
	}
	return total
}

// drawStatusBadge 绘制连接状态 / 状态徽章（带圆点）。
func drawStatusBadge(screen *ebiten.Image, cache *assetfont.TextCache,
	text string, clr color.RGBA, cx, y float32) {
	img := cache.GetImage(text, clr)
	tw := float32(img.Bounds().Dx())
	th := float32(img.Bounds().Dy())
	padX := float32(16)
	bw := tw + padX*2 + 18 // 包括左侧圆点
	bh := th + 10
	bx := cx - bw/2
	by := y
	vector.DrawFilledRect(screen, bx, by, bw, bh,
		color.RGBA{20, 24, 50, 220}, false)
	vector.StrokeRect(screen, bx, by, bw, bh, 1, clr, false)
	// 状态点
	vector.DrawFilledCircle(screen, bx+padX, by+bh/2, 5, clr, false)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(bx+padX+12), float64(by+5))
	screen.DrawImage(img, op)
}

// drawRoomBadge 在房间号区域绘制大号的房间 ID 卡片。
func drawRoomBadge(screen *ebiten.Image, hint, head *assetfont.TextCache,
	roomID string, cx, y float32, _ int) {
	if roomID == "" {
		roomID = "------"
	}
	w := float32(380)
	h := float32(110)
	x := cx - w/2
	vector.DrawFilledRect(screen, x, y, w, h, colPanelDeep, false)
	vector.StrokeRect(screen, x, y, w, h, 2, colGold, false)
	vector.StrokeRect(screen, x+5, y+5, w-10, h-10, 1, colGoldDim, false)

	drawCenteredText(screen, hint, "ROOM ID", colGoldDim, cx, y+12)
	drawCenteredText(screen, head, roomID, colGoldHi, cx, y+38)
}

// connectionStatus 把网络层状态映射为可读文字与颜色。
func (g *Game) connectionStatus() (string, color.RGBA) {
	if g.netClient == nil {
		return "未连接", colMute
	}
	switch g.netClient.State() {
	case network.StateConnected:
		return "● 已连接", colTeal
	case network.StateConnecting:
		return "○ 连接中…", colGoldHi
	case network.StateReconnecting:
		return "○ 重连中…", colCrimson
	default:
		return "未连接", colMute
	}
}

// lobbyMessageColor 根据消息内容选择展示色。
func lobbyMessageColor(msg string) color.RGBA {
	if strings.Contains(msg, "失败") || strings.Contains(msg, "错误") ||
		strings.Contains(msg, "请填写") || strings.Contains(msg, "需要") ||
		strings.Contains(msg, "请选择") {
		return colCrimson
	}
	if strings.Contains(msg, "等待") {
		return colMute
	}
	if strings.Contains(msg, "开始") || strings.Contains(msg, "成功") ||
		strings.Contains(msg, "已选择") || strings.Contains(msg, "已点亮") {
		return colTeal
	}
	return colCream
}

// parseWaitingLine 解析 applyRoomInfo 拼装的字符串：
//   "<name> [<char>] (<status>)"
// 解析失败时返回原始字符串作为名字。
func parseWaitingLine(line string) (name, char, status string) {
	name = line
	if i := strings.Index(line, "["); i > 0 {
		name = strings.TrimSpace(line[:i])
		rest := line[i+1:]
		if j := strings.Index(rest, "]"); j >= 0 {
			char = rest[:j]
			rest = rest[j+1:]
		}
		if k := strings.Index(rest, "("); k >= 0 {
			rest = rest[k+1:]
			if m := strings.Index(rest, ")"); m >= 0 {
				status = rest[:m]
			}
		}
	}
	return
}

// lerpColor 在两种 RGBA 之间线性插值。
func lerpColor(a, b color.RGBA, t float64) color.RGBA {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return color.RGBA{
		R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t),
		A: 255,
	}
}

// indexOf 返回 s 在 list 中的下标，未找到返回 -1。
func indexOf(list []string, s string) int {
	for i, v := range list {
		if v == s {
			return i
		}
	}
	return -1
}

// drawSpectatorOverlay 在观战状态下覆盖顶部提示条与右下角水印。
func drawSpectatorOverlay(screen *ebiten.Image,
	head, hint *assetfont.TextCache, tick int) {
	w := float32(render.ScreenWidth)

	// 顶部提示条：半透明深色背景 + 金色边框
	barH := float32(46)
	vector.DrawFilledRect(screen, 0, 0, w, barH, color.RGBA{8, 11, 32, 210}, false)
	vector.DrawFilledRect(screen, 0, barH, w, 1, colGold, false)

	// 左侧红圆点（呼吸）
	pulse := 0.5 + 0.5*math.Sin(float64(tick)*0.12)
	dotClr := color.RGBA{212, 82, 78, uint8(140 + int(115*pulse))}
	vector.DrawFilledCircle(screen, 32, barH/2, 7, dotClr, false)

	// 主提示文字
	drawText(screen, head, "观战中", colGoldHi, 52, 6)

	// 右侧提示
	hintText := "仅可查看 · T 聊天 · Esc 退出观战"
	hintImg := hint.GetImage(hintText, colMute)
	hw := float32(hintImg.Bounds().Dx())
	drawText(screen, hint, hintText, colMute, w-hw-24, 16)

	// 右下角水印
	watermark := "SPECTATOR MODE"
	wmImg := head.GetImage(watermark, color.RGBA{212, 168, 67, 70})
	wmW := float32(wmImg.Bounds().Dx())
	wmH := float32(wmImg.Bounds().Dy())
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(w-wmW-32), float64(float32(render.ScreenHeight)-wmH-32))
	screen.DrawImage(wmImg, op)
}
