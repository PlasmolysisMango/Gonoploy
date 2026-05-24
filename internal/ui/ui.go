package ui

import (
	"fmt"
	"image/color"

	assetfont "github.com/PlasmolysisMango/Gonopoly/asset/font"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Manager struct {
	textCache *assetfont.TextCache
	messages  []string
	eventText string
	scrollOff int

	diceClicked bool
	playerName  string
	playerMoney string
	playerSkill string
}

func NewManager(tc *assetfont.TextCache) *Manager {
	return &Manager{
		textCache: tc,
	}
}

func (m *Manager) Update() {
	m.diceClicked = false
	if inpututil.IsKeyJustPressed(ebiten.KeyD) {
		m.diceClicked = true
	}

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
	m.drawControls(screen)
}

func (m *Manager) drawMoneyBox(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, 700, 250, 200, 80, color.RGBA{50, 50, 70, 230}, false)
	vector.StrokeRect(screen, 700, 250, 200, 80, 2, color.White, false)

	if m.playerName != "" {
		nameImg := m.textCache.GetImage(m.playerName, color.RGBA{255, 220, 100, 255})
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(710, 258)
		screen.DrawImage(nameImg, op)
	}
	if m.playerMoney != "" {
		moneyImg := m.textCache.GetImage(m.playerMoney, color.White)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(710, 280)
		screen.DrawImage(moneyImg, op)
	}
	if m.playerSkill != "" {
		skillImg := m.textCache.GetImage(m.playerSkill, color.RGBA{150, 200, 255, 255})
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(710, 302)
		screen.DrawImage(skillImg, op)
	}
}

func (m *Manager) drawMessageBox(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, 950, 250, 400, 400, color.RGBA{30, 30, 50, 230}, false)
	vector.StrokeRect(screen, 950, 250, 400, 400, 2, color.RGBA{100, 100, 120, 255}, false)

	titleImg := m.textCache.GetImage("消息记录", color.RGBA{200, 200, 200, 255})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(960, 255)
	screen.DrawImage(titleImg, op)

	startIdx := m.scrollOff
	endIdx := startIdx + 18
	if endIdx > len(m.messages) {
		endIdx = len(m.messages)
	}

	y := 280
	for i := startIdx; i < endIdx; i++ {
		msgImg := m.textCache.GetImage(m.messages[i], color.RGBA{220, 220, 220, 255})
		mop := &ebiten.DrawImageOptions{}
		mop.GeoM.Translate(960, float64(y))
		screen.DrawImage(msgImg, mop)
		y += 20
	}
}

func (m *Manager) drawEventBox(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, 250, 250, 400, 400, color.RGBA{30, 30, 50, 230}, false)
	vector.StrokeRect(screen, 250, 250, 400, 400, 2, color.RGBA{100, 100, 120, 255}, false)

	if m.eventText != "" {
		img := m.textCache.GetImage(m.eventText, color.RGBA{255, 255, 200, 255})
		eop := &ebiten.DrawImageOptions{}
		eop.GeoM.Translate(270, 280)
		screen.DrawImage(img, eop)
	}
}

func (m *Manager) drawControls(screen *ebiten.Image) {
	// Dice button
	vector.DrawFilledRect(screen, 700, 550, 200, 60, color.RGBA{60, 120, 60, 255}, false)
	vector.StrokeRect(screen, 700, 550, 200, 60, 2, color.White, false)
	diceLabel := m.textCache.GetImage("[D] 掷骰子", color.White)
	dop := &ebiten.DrawImageOptions{}
	dop.GeoM.Translate(740, 570)
	screen.DrawImage(diceLabel, dop)

	// Row 1: Buy + End
	vector.DrawFilledRect(screen, 700, 630, 95, 35, color.RGBA{60, 60, 120, 255}, false)
	buyLabel := m.textCache.GetImage("[B]买地", color.White)
	bop := &ebiten.DrawImageOptions{}
	bop.GeoM.Translate(712, 638)
	screen.DrawImage(buyLabel, bop)

	vector.DrawFilledRect(screen, 805, 630, 95, 35, color.RGBA{120, 60, 60, 255}, false)
	nextLabel := m.textCache.GetImage("[N]结束", color.White)
	nop := &ebiten.DrawImageOptions{}
	nop.GeoM.Translate(817, 638)
	screen.DrawImage(nextLabel, nop)

	// Row 2: Build + Mortgage + Redeem
	vector.DrawFilledRect(screen, 700, 672, 63, 35, color.RGBA{50, 90, 50, 255}, false)
	cLabel := m.textCache.GetImage("[C]建", color.White)
	cop := &ebiten.DrawImageOptions{}
	cop.GeoM.Translate(706, 680)
	screen.DrawImage(cLabel, cop)

	vector.DrawFilledRect(screen, 768, 672, 63, 35, color.RGBA{90, 60, 50, 255}, false)
	mLabel := m.textCache.GetImage("[M]押", color.White)
	mop := &ebiten.DrawImageOptions{}
	mop.GeoM.Translate(774, 680)
	screen.DrawImage(mLabel, mop)

	vector.DrawFilledRect(screen, 836, 672, 63, 35, color.RGBA{50, 60, 90, 255}, false)
	rLabel := m.textCache.GetImage("[R]赎", color.White)
	rop := &ebiten.DrawImageOptions{}
	rop.GeoM.Translate(842, 680)
	screen.DrawImage(rLabel, rop)

	// Row 3: Trade + Skill
	vector.DrawFilledRect(screen, 700, 714, 95, 35, color.RGBA{80, 50, 100, 255}, false)
	tLabel := m.textCache.GetImage("[T]交易", color.White)
	top := &ebiten.DrawImageOptions{}
	top.GeoM.Translate(712, 722)
	screen.DrawImage(tLabel, top)

	vector.DrawFilledRect(screen, 805, 714, 95, 35, color.RGBA{100, 80, 30, 255}, false)
	sLabel := m.textCache.GetImage("[S]技能", color.White)
	sop := &ebiten.DrawImageOptions{}
	sop.GeoM.Translate(817, 722)
	screen.DrawImage(sLabel, sop)
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
	return m.diceClicked
}

func (m *Manager) SetPlayerInfo(name string, money int, skillPts int) {
	m.playerName = name
	m.playerMoney = fmt.Sprintf("$%d", money)
	m.playerSkill = fmt.Sprintf("技能点:%d", skillPts)
}
