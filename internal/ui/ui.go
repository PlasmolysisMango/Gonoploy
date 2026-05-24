package ui

import (
	"fmt"
	"image"
	"image/color"

	assetfont "github.com/PlasmolysisMango/Gonopoly/asset/font"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// ChatDisplayMsg 聊天消息显示条目。
type ChatDisplayMsg struct {
	From    string
	Content string
	Time    string
}

type Button struct {
	Rect    image.Rectangle
	Label   string
	Color   color.RGBA
	Visible bool
	Locked  bool
	pressed bool
	clicked bool
}

func NewButton(x, y, w, h int, label string, clr color.RGBA) *Button {
	return &Button{
		Rect:    image.Rect(x, y, x+w, y+h),
		Label:   label,
		Color:   clr,
		Visible: true,
	}
}

func (b *Button) Update() {
	b.clicked = false
	if !b.Visible || b.Locked {
		return
	}
	mx, my := ebiten.CursorPosition()
	inside := image.Pt(mx, my).In(b.Rect)

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && inside {
		b.pressed = true
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		if b.pressed && inside {
			b.clicked = true
		}
		b.pressed = false
	}
}

func (b *Button) Draw(screen *ebiten.Image, tc *assetfont.TextCache) {
	if !b.Visible {
		return
	}
	x := float32(b.Rect.Min.X)
	y := float32(b.Rect.Min.Y)
	w := float32(b.Rect.Dx())
	h := float32(b.Rect.Dy())

	fillColor := b.Color
	if b.Locked {
		fillColor = color.RGBA{180, 180, 180, 255}
	} else if b.pressed {
		fillColor.R = fillColor.R / 2
		fillColor.G = fillColor.G / 2
		fillColor.B = fillColor.B / 2
	}
	vector.DrawFilledRect(screen, x, y, w, h, fillColor, false)
	vector.StrokeRect(screen, x, y, w, h, 2, color.Black, false)

	textColor := color.RGBA{0, 0, 0, 255}
	if b.Locked {
		textColor = color.RGBA{120, 120, 120, 255}
	}
	labelImg := tc.GetImage(b.Label, textColor)
	lw := labelImg.Bounds().Dx()
	lh := labelImg.Bounds().Dy()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(b.Rect.Min.X)+(float64(w)-float64(lw))/2,
		float64(b.Rect.Min.Y)+(float64(h)-float64(lh))/2)
	screen.DrawImage(labelImg, op)
}

func (b *Button) Clicked() bool {
	return b.clicked
}

// Manager owns all UI widgets
type Manager struct {
	textCache *assetfont.TextCache
	messages  []string
	eventText string
	scrollOff int

	playerName  string
	playerMoney string
	playerSkill string

	BtnDice *Button
	Btn1    *Button
	Btn2    *Button
	Btn3    *Button
	Btn4    *Button

	diceMode bool

	// 聊天相关
	chatMessages []ChatDisplayMsg
	chatInput    string
	chatActive   bool
	chatScroll   int
}

func NewManager(tc *assetfont.TextCache) *Manager {
	m := &Manager{
		textCache: tc,
		diceMode:  true,
	}
	m.BtnDice = NewButton(700, 550, 200, 100, "骰子!", color.RGBA{230, 230, 230, 255})
	m.Btn1 = NewButton(320, 517, 100, 50, "买地", color.RGBA{230, 230, 230, 255})
	m.Btn2 = NewButton(480, 517, 100, 50, "角色", color.RGBA{230, 230, 230, 255})
	m.Btn3 = NewButton(320, 583, 100, 50, "设置", color.RGBA{230, 230, 230, 255})
	m.Btn4 = NewButton(480, 583, 100, 50, "操作", color.RGBA{230, 230, 230, 255})
	return m
}

func (m *Manager) Update() {
	m.BtnDice.Update()
	m.Btn1.Update()
	m.Btn2.Update()
	m.Btn3.Update()
	m.Btn4.Update()

	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		if m.scrollOff > 0 {
			m.scrollOff--
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		if m.scrollOff < len(m.messages)-10 {
			m.scrollOff++
		}
	}
}

func (m *Manager) Draw(screen *ebiten.Image) {
	m.drawMoneyBox(screen)
	m.drawMessageBox(screen)
	m.drawEventBox(screen)
	m.drawDiceArea(screen)
	m.drawButtons(screen)
}

func (m *Manager) drawMoneyBox(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, 700, 250, 200, 100, color.White, false)
	vector.StrokeRect(screen, 700, 250, 200, 100, 2, color.Black, false)

	if m.playerName != "" {
		label := "当前玩家："
		labelImg := m.textCache.GetImage(label, color.Black)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(720, 260)
		screen.DrawImage(labelImg, op)

		nameImg := m.textCache.GetImage(m.playerName, color.Black)
		nop := &ebiten.DrawImageOptions{}
		nop.GeoM.Translate(750, 285)
		screen.DrawImage(nameImg, nop)
	}
	if m.playerMoney != "" {
		moneyImg := m.textCache.GetImage("金钱："+m.playerMoney, color.Black)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(720, 310)
		screen.DrawImage(moneyImg, op)
	}
}

func (m *Manager) drawMessageBox(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, 950, 250, 400, 400, color.White, false)
	vector.StrokeRect(screen, 950, 250, 400, 400, 2, color.Black, false)

	startIdx := m.scrollOff
	endIdx := startIdx + 18
	if endIdx > len(m.messages) {
		endIdx = len(m.messages)
	}

	y := 260
	for i := startIdx; i < endIdx; i++ {
		prefix := fmt.Sprintf("【%d】", i+1)
		msgImg := m.textCache.GetImage(prefix+m.messages[i], color.Black)
		mop := &ebiten.DrawImageOptions{}
		mop.GeoM.Translate(960, float64(y))
		screen.DrawImage(msgImg, mop)
		y += 20
	}
}

func (m *Manager) drawEventBox(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, 250, 250, 400, 250, color.White, false)
	vector.StrokeRect(screen, 250, 250, 400, 250, 2, color.Black, false)

	if m.eventText != "" {
		lines := splitLines(m.eventText)
		y := 270
		for _, line := range lines {
			if line == "" {
				y += 20
				continue
			}
			img := m.textCache.GetImage(line, color.Black)
			eop := &ebiten.DrawImageOptions{}
			eop.GeoM.Translate(270, float64(y))
			screen.DrawImage(img, eop)
			y += 22
		}
	}
}

func (m *Manager) drawDiceArea(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, 700, 410, 200, 80, color.White, false)
	vector.StrokeRect(screen, 700, 410, 200, 80, 2, color.Black, false)
}

func (m *Manager) drawButtons(screen *ebiten.Image) {
	m.BtnDice.Draw(screen, m.textCache)
	m.Btn1.Draw(screen, m.textCache)
	m.Btn2.Draw(screen, m.textCache)
	m.Btn3.Draw(screen, m.textCache)
	m.Btn4.Draw(screen, m.textCache)
}

func (m *Manager) AddMessage(msg string) {
	m.messages = append(m.messages, msg)
	if len(m.messages) > 100 {
		m.messages = m.messages[1:]
	}
	if len(m.messages) > 18 {
		m.scrollOff = len(m.messages) - 18
	}
}

func (m *Manager) SetEventText(text string) {
	m.eventText = text
}

func (m *Manager) DiceButtonClicked() bool {
	if !m.diceMode {
		return false
	}
	return m.BtnDice.Clicked() || inpututil.IsKeyJustPressed(ebiten.KeyD)
}

func (m *Manager) EndTurnClicked() bool {
	if m.diceMode {
		return false
	}
	return m.BtnDice.Clicked() || inpututil.IsKeyJustPressed(ebiten.KeyN)
}

func (m *Manager) SetDiceMode(isDice bool) {
	m.diceMode = isDice
	if isDice {
		m.BtnDice.Label = "骰子!"
	} else {
		m.BtnDice.Label = "结束回合"
	}
}

func (m *Manager) SetPlayerInfo(name string, money int, skillPts int) {
	m.playerName = name
	m.playerMoney = fmt.Sprintf("%d元", money)
	m.playerSkill = fmt.Sprintf("技能点:%d", skillPts)
}

// SetButtons sets labels and visibility of the 4 action buttons
func (m *Manager) SetButtons(b1, b2, b3, b4 string) {
	m.Btn1.Label = b1
	m.Btn2.Label = b2
	m.Btn3.Label = b3
	m.Btn4.Label = b4
	m.Btn1.Visible = b1 != ""
	m.Btn2.Visible = b2 != ""
	m.Btn3.Visible = b3 != ""
	m.Btn4.Visible = b4 != ""
	m.Btn1.Locked = false
	m.Btn2.Locked = false
	m.Btn3.Locked = false
	m.Btn4.Locked = false
}

func (m *Manager) LockBtn1(locked bool) { m.Btn1.Locked = locked }
func (m *Manager) LockBtn2(locked bool) { m.Btn2.Locked = locked }
func (m *Manager) LockBtn3(locked bool) { m.Btn3.Locked = locked }
func (m *Manager) LockBtn4(locked bool) { m.Btn4.Locked = locked }

func (m *Manager) Btn1Clicked() bool { return m.Btn1.Clicked() }
func (m *Manager) Btn2Clicked() bool { return m.Btn2.Clicked() }
func (m *Manager) Btn3Clicked() bool { return m.Btn3.Clicked() }
func (m *Manager) Btn4Clicked() bool { return m.Btn4.Clicked() }

func splitLines(s string) []string {
	var lines []string
	current := ""
	for _, c := range s {
		if c == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

// ----- 聊天系统方法 -----

// AddChatMessage 添加一条聊天消息到显示列表。
func (m *Manager) AddChatMessage(from, content, timeStr string) {
	m.chatMessages = append(m.chatMessages, ChatDisplayMsg{
		From:    from,
		Content: content,
		Time:    timeStr,
	})
	if len(m.chatMessages) > 100 {
		m.chatMessages = m.chatMessages[1:]
	}
	// 自动滚动到底部
	if len(m.chatMessages) > 6 {
		m.chatScroll = len(m.chatMessages) - 6
	}
}

// SetChatActive 设置聊天输入框是否激活。
func (m *Manager) SetChatActive(active bool) {
	m.chatActive = active
}

// IsChatActive 返回聊天输入框是否激活。
func (m *Manager) IsChatActive() bool {
	return m.chatActive
}

// GetChatInput 获取当前聊天输入内容。
func (m *Manager) GetChatInput() string {
	return m.chatInput
}

// SetChatInput 设置聊天输入内容。
func (m *Manager) SetChatInput(s string) {
	m.chatInput = s
}

// ClearChatInput 清空聊天输入框。
func (m *Manager) ClearChatInput() {
	m.chatInput = ""
}

// DrawChat 绘制聊天区域（屏幕左下角）。
func (m *Manager) DrawChat(screen *ebiten.Image) {
	const (
		chatX      float32 = 10
		chatY      float32 = 680
		chatW      float32 = 380
		chatH      float32 = 200
		lineHeight         = 22
		maxVisible         = 6
		inputH     float32 = 28
	)

	// 半透明背景
	vector.DrawFilledRect(screen, chatX, chatY, chatW, chatH,
		color.RGBA{0, 0, 0, 140}, false)
	vector.StrokeRect(screen, chatX, chatY, chatW, chatH, 1,
		color.RGBA{200, 200, 200, 180}, false)

	// 绘制聊天消息
	startIdx := m.chatScroll
	endIdx := startIdx + maxVisible
	if endIdx > len(m.chatMessages) {
		endIdx = len(m.chatMessages)
	}

	y := int(chatY) + 6
	for i := startIdx; i < endIdx; i++ {
		cm := m.chatMessages[i]
		line := "[" + cm.From + "] " + cm.Content
		img := m.textCache.GetImage(line, color.RGBA{230, 230, 230, 255})
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(chatX)+8, float64(y))
		screen.DrawImage(img, op)
		y += lineHeight
	}

	// 绘制输入框
	inputY := chatY + chatH - inputH - 4
	if m.chatActive {
		vector.DrawFilledRect(screen, chatX+4, inputY, chatW-8, inputH,
			color.RGBA{40, 40, 60, 220}, false)
		vector.StrokeRect(screen, chatX+4, inputY, chatW-8, inputH, 2,
			color.RGBA{100, 200, 255, 255}, false)
		// 显示输入文字 + 光标
		display := m.chatInput + "_"
		inputImg := m.textCache.GetImage(display, color.White)
		iop := &ebiten.DrawImageOptions{}
		iop.GeoM.Translate(float64(chatX)+10, float64(inputY)+4)
		screen.DrawImage(inputImg, iop)
	} else {
		vector.DrawFilledRect(screen, chatX+4, inputY, chatW-8, inputH,
			color.RGBA{30, 30, 30, 160}, false)
		hintImg := m.textCache.GetImage("按T键聊天", color.RGBA{150, 150, 150, 200})
		hop := &ebiten.DrawImageOptions{}
		hop.GeoM.Translate(float64(chatX)+10, float64(inputY)+4)
		screen.DrawImage(hintImg, hop)
	}
}
