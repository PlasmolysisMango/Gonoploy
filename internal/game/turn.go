package game

import (
	"fmt"

	"github.com/PlasmolysisMango/Gonopoly/internal/model"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func inpututil_isKeyJustPressed(key ebiten.Key) bool {
	return inpututil.IsKeyJustPressed(key)
}

func (g *Game) updatePlaying() {
	if g.alivePlayers() <= 1 {
		g.state = StateGameOver
		return
	}

	g.uiMgr.Update()
	g.updatePlayerInfoUI()
	g.tickAutoSave()

	if inpututil_isKeyJustPressed(ebiten.KeyF5) {
		if err := g.Save("saves/manual.json"); err == nil {
			g.uiMgr.AddMessage("游戏已保存")
		}
	}
	if inpututil_isKeyJustPressed(ebiten.KeyF9) {
		if err := g.Load("saves/manual.json"); err == nil {
			g.uiMgr.AddMessage("游戏已加载")
		}
	}
	if inpututil_isKeyJustPressed(ebiten.KeyF10) {
		if g.audioMgr != nil {
			g.audioMgr.ToggleMute()
		}
	}

	switch g.turnPhase {
	case TurnWaitRoll:
		g.updateWaitRoll()
	case TurnRolling:
		g.updateRolling()
	case TurnMoving:
		g.updateMoving()
	case TurnLanded:
		g.updateLanded()
	case TurnOperating:
		g.updateOperating()
	case TurnEndTurn:
		g.updateEndTurn()
	}
}

func (g *Game) updateWaitRoll() {
	g.uiMgr.SetDiceMode(true)
	p := g.ActivePlayer()
	if p.InJail > 0 {
		p.InJail--
		if p.InJail == 0 {
			g.uiMgr.AddMessage(fmt.Sprintf("%s 出狱了", p.Name))
		} else {
			g.uiMgr.AddMessage(fmt.Sprintf("%s 在监狱中，跳过回合", p.Name))
			g.turnPhase = TurnEndTurn
			return
		}
	}

	if g.uiMgr.DiceButtonClicked() {
		g.dice.StartRoll()
		g.turnPhase = TurnRolling
	}
}

func (g *Game) updateRolling() {
	g.dice.Tick()
	if g.dice.Rolled {
		p := g.ActivePlayer()
		g.uiMgr.AddMessage(fmt.Sprintf("%s 掷出了 %d+%d=%d",
			p.Name, g.dice.Values[0], g.dice.Values[1], g.dice.Sum))

		if g.dice.IsDouble {
			p.BonusCount++
			if p.BonusCount >= 3 {
				g.sendToJail(p)
				g.turnPhase = TurnEndTurn
				return
			}
		}

		g.moveSteps = g.dice.Sum
		g.moveCounter = 0
		g.moveTick = 0
		g.turnPhase = TurnMoving
		g.uiMgr.SetDiceMode(false)
	}
}

var _ = 0 // ensure no import cycle

func (g *Game) updateMoving() {
	g.moveTick++
	if g.moveTick >= 6 {
		g.moveTick = 0
		p := g.ActivePlayer()
		oldPos := p.Position
		p.Position = (p.Position + 1) % g.board.SpaceCount()
		g.moveCounter++

		if p.Position == 0 && oldPos != 0 {
			p.Money += 150
			p.SkillPoints++
			g.uiMgr.AddMessage(fmt.Sprintf("%s 经过起点，获得$150", p.Name))
		}

		if g.moveCounter >= g.moveSteps {
			g.turnPhase = TurnLanded
		}
	}
}

func (g *Game) updateLanded() {
	p := g.ActivePlayer()
	s := g.board.Spaces[p.Position]
	g.uiMgr.AddMessage(fmt.Sprintf("%s 到达 %s", p.Name, s.Name))

	switch s.Type {
	case model.SpaceEvent:
		g.handleEventSpace(p, s)
	case model.SpaceLand, model.SpaceUtility, model.SpaceTransport:
		if s.Owner == nil {
			g.uiMgr.SetEventText(fmt.Sprintf("%s 无主，价格$%d\n按B购买", s.Name, s.Price))
			g.turnPhase = TurnOperating
			g.menuState = MenuMain
			return
		} else if s.Owner != p {
			charge := p.GetCharge(s, g.board, g.dice.Sum)
			if charge > 0 {
				p.Money -= charge
				s.Owner.Money += charge
				g.uiMgr.AddMessage(fmt.Sprintf("%s 向 %s 支付租金$%d", p.Name, s.Owner.Name, charge))
				if p.Money < 0 {
					g.handleBankruptcy(p, s.Owner)
					return
				}
			}
		}
	}

	g.turnPhase = TurnOperating
	g.menuState = MenuMain
}

func (g *Game) handleEventSpace(p *model.Player, s *model.Space) {
	switch s.ID {
	case 0: // 起点 - already handled in movement
	case 4, 11, 20, 31: // 机会
		g.resolveChance(p)
	case 5: // 技能
		p.SkillPoints += 2
		g.uiMgr.AddMessage(fmt.Sprintf("%s 获得2点技能点", p.Name))
	case 16: // 监狱
		g.sendToJail(p)
	case 21: // 祝福
		g.resolveBlessing(p)
	}
}

func (g *Game) updateOperating() {
	g.updateOperatingFull()
}

func (g *Game) updateEndTurn() {
	p := g.ActivePlayer()
	if g.dice.IsDouble && p.BonusCount < 3 && p.InJail == 0 {
		g.uiMgr.AddMessage(fmt.Sprintf("%s 掷出双数，再来一次!", p.Name))
		p.HasOperated = false
		g.dice.Reset()
		g.uiMgr.SetDiceMode(true)
		g.turnPhase = TurnWaitRoll
	} else {
		p.BonusCount = 0
		g.dice.Reset()
		g.nextPlayer()
		g.uiMgr.SetDiceMode(true)
		g.uiMgr.AddMessage(fmt.Sprintf("轮到 %s", g.ActivePlayer().Name))
		g.turnPhase = TurnWaitRoll
	}
}

func (g *Game) buyProperty(p *model.Player, s *model.Space) {
	p.Money -= s.Price
	p.AddProperty(s)
	g.uiMgr.AddMessage(fmt.Sprintf("%s 购买了 %s ($%d)", p.Name, s.Name, s.Price))
	g.uiMgr.SetEventText("")
}

func (g *Game) sendToJail(p *model.Player) {
	if p.JailPassports > 0 {
		p.JailPassports--
		g.uiMgr.AddMessage(fmt.Sprintf("%s 使用出狱卡!", p.Name))
		return
	}
	p.Position = 16
	p.InJail = 2
	p.BonusCount = 0
	g.uiMgr.AddMessage(fmt.Sprintf("%s 入狱!", p.Name))
}

func (g *Game) handleBankruptcy(p *model.Player, creditor *model.Player) {
	if p.TotalAssets() < 0 {
		p.Bankrupt = true
		for _, s := range p.OwnedLands {
			s.Owner = nil
			s.Houses = 0
			s.HasHotel = false
			s.Mortgaged = false
		}
		for _, s := range p.OwnedUtilities {
			s.Owner = nil
		}
		for _, s := range p.OwnedTransports {
			s.Owner = nil
		}
		g.uiMgr.AddMessage(fmt.Sprintf("%s 破产了!", p.Name))
		g.turnPhase = TurnEndTurn
	}
}
