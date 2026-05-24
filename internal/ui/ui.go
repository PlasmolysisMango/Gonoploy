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

type Button struct {
	Rect    image.Rectangle
	Label   string
	Color   color.RGBA
	Visible bool
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
	if !b.Visible {
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
	if b.pressed {
		fillColor.R = fillColor.R / 2
		fillColor.G = fillColor.G / 2
		fillColor.B = fillColor.B / 2
	}
	vector.DrawFilledRect(screen, x, y, w, h, fillColor, false)
	vector.StrokeRect(screen, x, y, w, h, 2, color.Black, false)

	labelImg := tc.GetImage(b.Label, color.Black)
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

	BtnDice     *Button
	BtnBuy      *Button
	BtnChar     *Button
	BtnSetting  *Button
	BtnOperate  *Button
}

func NewManager(tc *assetfont.TextCache) *Manager {
	m := &Manager{
		textCache: tc,
	}
	// Match Python layout exactly:
	// dice_button: [700, 550, 200, 100]
	m.BtnDice = NewButton(700, 550, 200, 100, "骰子!", color.RGBA{230, 230, 230, 255})
	// button1 (买地): [320, 517, 100, 50]
	m.BtnBuy = NewButton(320, 517, 100, 50, "买地", color.RGBA{230, 230, 230, 255})
	// button2 (角色): [480, 517, 100, 50]
	m.BtnChar = NewButton(480, 517, 100, 50, "角色", color.RGBA{230, 230, 230, 255})
	// button3 (设置): [320, 583, 100, 50]
	m.BtnSetting = NewButton(320, 583, 100, 50, "设置", color.RGBA{230, 230, 230, 255})
	// button4 (操作): [480, 583, 100, 50]
	m.BtnOperate = NewButton(480, 583, 100, 50, "操作", color.RGBA{230, 230, 230, 255})
	return m
}

func (m *Manager) Update() {
	m.BtnDice.Update()
	m.BtnBuy.Update()
	m.BtnChar.Update()
	m.BtnSetting.Update()
	m.BtnOperate.Update()

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
	// moneybox: [700, 250, 200, 100]
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
	// messagebox: [950, 250, 400, 400]
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
	// eventbox: [250, 250, 400, 250]
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
	// dice display area: [700, 410, 200, 80]
	vector.DrawFilledRect(screen, 700, 410, 200, 80, color.White, false)
	vector.StrokeRect(screen, 700, 410, 200, 80, 2, color.Black, false)
}

func (m *Manager) drawButtons(screen *ebiten.Image) {
	m.BtnDice.Draw(screen, m.textCache)
	m.BtnBuy.Draw(screen, m.textCache)
	m.BtnChar.Draw(screen, m.textCache)
	m.BtnSetting.Draw(screen, m.textCache)
	m.BtnOperate.Draw(screen, m.textCache)
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
	return m.BtnDice.Clicked() || inpututil.IsKeyJustPressed(ebiten.KeyD)
}

func (m *Manager) BuyButtonClicked() bool {
	return m.BtnBuy.Clicked() || inpututil.IsKeyJustPressed(ebiten.KeyB)
}

func (m *Manager) OperateButtonClicked() bool {
	return m.BtnOperate.Clicked()
}

func (m *Manager) SetPlayerInfo(name string, money int, skillPts int) {
	m.playerName = name
	m.playerMoney = fmt.Sprintf("%d元", money)
	m.playerSkill = fmt.Sprintf("技能点:%d", skillPts)
}

func (m *Manager) SetBuyLabel(label string) {
	m.BtnBuy.Label = label
}

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
