package game

import (
	"fmt"

	"github.com/PlasmolysisMango/Gonopoly/internal/model"
	"github.com/hajimehoshi/ebiten/v2"
)

func (g *Game) updateOperatingFull() {
	p := g.ActivePlayer()
	s := g.board.Spaces[p.Position]

	switch g.menuState {
	case MenuMain:
		if g.uiMgr.BuyButtonClicked() {
			if s.IsBuyable() && p.Money >= s.Price {
				g.buyProperty(p, s)
			}
		}
		if inpututil_isKeyJustPressed(ebiten.KeyC) || g.uiMgr.OperateButtonClicked() {
			g.menuState = MenuBuild
			g.buildCursor = 0
			g.updateBuildableList()
		}
		if inpututil_isKeyJustPressed(ebiten.KeyM) {
			g.menuState = MenuMortgage
			g.buildCursor = 0
			g.updateMortgageList()
		}
		if inpututil_isKeyJustPressed(ebiten.KeyR) {
			g.menuState = MenuBuyback
			g.buildCursor = 0
			g.updateRedeemList()
		}
		if inpututil_isKeyJustPressed(ebiten.KeyT) {
			g.menuState = MenuDeal
			g.buildCursor = 0
			g.dealTarget = (g.activeIdx + 1) % len(g.players)
			g.updateDealList()
		}
		if inpututil_isKeyJustPressed(ebiten.KeyS) {
			g.useSkill(p)
		}
		if inpututil_isKeyJustPressed(ebiten.KeyN) {
			g.turnPhase = TurnEndTurn
		}

	case MenuBuild:
		g.handleBuildMenu(p)
	case MenuMortgage:
		g.handleMortgageMenu(p)
	case MenuBuyback:
		g.handleBuybackMenu(p)
	case MenuDeal:
		g.handleDealMenu(p)
	}
}

func (g *Game) handleBuildMenu(p *model.Player) {
	if inpututil_isKeyJustPressed(ebiten.KeyEscape) {
		g.menuState = MenuMain
		return
	}
	if len(g.buildableSpaces) == 0 {
		g.uiMgr.SetEventText("没有可建造的地产")
		g.menuState = MenuMain
		return
	}
	if inpututil_isKeyJustPressed(ebiten.KeyLeft) {
		g.buildCursor = (g.buildCursor - 1 + len(g.buildableSpaces)) % len(g.buildableSpaces)
	}
	if inpututil_isKeyJustPressed(ebiten.KeyRight) {
		g.buildCursor = (g.buildCursor + 1) % len(g.buildableSpaces)
	}
	if inpututil_isKeyJustPressed(ebiten.KeyEnter) {
		target := g.buildableSpaces[g.buildCursor]
		cost := target.BuildCost()
		if p.Money >= cost {
			p.Money -= cost
			if target.Houses < 4 {
				target.Houses++
				g.uiMgr.AddMessage(fmt.Sprintf("%s 在 %s 建造房屋 (Lv%d) $%d",
					p.Name, target.Name, target.Houses, cost))
			} else {
				target.HasHotel = true
				target.Houses = 0
				g.uiMgr.AddMessage(fmt.Sprintf("%s 在 %s 建造酒店! $%d",
					p.Name, target.Name, cost))
			}
			g.updateBuildableList()
		} else {
			g.uiMgr.AddMessage("资金不足!")
		}
	}

	if len(g.buildableSpaces) > 0 {
		target := g.buildableSpaces[g.buildCursor]
		g.uiMgr.SetEventText(fmt.Sprintf("建造 [←→选择 Enter确认 Esc返回]\n%s (Lv%d) 费用:$%d",
			target.Name, target.Houses, target.BuildCost()))
	}
}

func (g *Game) handleMortgageMenu(p *model.Player) {
	if inpututil_isKeyJustPressed(ebiten.KeyEscape) {
		g.menuState = MenuMain
		g.uiMgr.SetEventText("")
		return
	}
	if len(g.mortgageSpaces) == 0 {
		g.uiMgr.SetEventText("没有可抵押的地产")
		g.menuState = MenuMain
		return
	}
	if inpututil_isKeyJustPressed(ebiten.KeyLeft) {
		g.buildCursor = (g.buildCursor - 1 + len(g.mortgageSpaces)) % len(g.mortgageSpaces)
	}
	if inpututil_isKeyJustPressed(ebiten.KeyRight) {
		g.buildCursor = (g.buildCursor + 1) % len(g.mortgageSpaces)
	}
	if inpututil_isKeyJustPressed(ebiten.KeyEnter) {
		target := g.mortgageSpaces[g.buildCursor]
		refund := 0
		if target.Houses > 0 {
			refund = target.Houses * target.BuildCost() / 2
			target.Houses = 0
		} else if target.HasHotel {
			refund = target.BuildCost() * 5 / 2
			target.HasHotel = false
		} else {
			refund = target.Price / 2
			target.Mortgaged = true
		}
		p.Money += refund
		g.uiMgr.AddMessage(fmt.Sprintf("%s 抵押 %s 获得$%d", p.Name, target.Name, refund))
		g.updateMortgageList()
	}

	if len(g.mortgageSpaces) > 0 && g.buildCursor < len(g.mortgageSpaces) {
		target := g.mortgageSpaces[g.buildCursor]
		val := target.Price / 2
		if target.Houses > 0 {
			val = target.Houses * target.BuildCost() / 2
		}
		g.uiMgr.SetEventText(fmt.Sprintf("抵押 [←→选择 Enter确认 Esc返回]\n%s 可得:$%d", target.Name, val))
	}
}

func (g *Game) handleBuybackMenu(p *model.Player) {
	if inpututil_isKeyJustPressed(ebiten.KeyEscape) {
		g.menuState = MenuMain
		g.uiMgr.SetEventText("")
		return
	}
	if len(g.redeemSpaces) == 0 {
		g.uiMgr.SetEventText("没有可赎回的地产")
		g.menuState = MenuMain
		return
	}
	if inpututil_isKeyJustPressed(ebiten.KeyLeft) {
		g.buildCursor = (g.buildCursor - 1 + len(g.redeemSpaces)) % len(g.redeemSpaces)
	}
	if inpututil_isKeyJustPressed(ebiten.KeyRight) {
		g.buildCursor = (g.buildCursor + 1) % len(g.redeemSpaces)
	}
	if inpututil_isKeyJustPressed(ebiten.KeyEnter) {
		target := g.redeemSpaces[g.buildCursor]
		cost := target.Price * 3 / 5
		if p.Money >= cost {
			p.Money -= cost
			target.Mortgaged = false
			g.uiMgr.AddMessage(fmt.Sprintf("%s 赎回 %s $%d", p.Name, target.Name, cost))
			g.updateRedeemList()
		} else {
			g.uiMgr.AddMessage("资金不足!")
		}
	}

	if len(g.redeemSpaces) > 0 && g.buildCursor < len(g.redeemSpaces) {
		target := g.redeemSpaces[g.buildCursor]
		cost := target.Price * 3 / 5
		g.uiMgr.SetEventText(fmt.Sprintf("赎回 [←→选择 Enter确认 Esc返回]\n%s 费用:$%d", target.Name, cost))
	}
}

func (g *Game) handleDealMenu(p *model.Player) {
	if inpututil_isKeyJustPressed(ebiten.KeyEscape) {
		g.menuState = MenuMain
		g.uiMgr.SetEventText("")
		return
	}
	if inpututil_isKeyJustPressed(ebiten.KeyTab) {
		for {
			g.dealTarget = (g.dealTarget + 1) % len(g.players)
			if g.dealTarget != g.activeIdx && !g.players[g.dealTarget].Bankrupt {
				break
			}
		}
		g.updateDealList()
		g.buildCursor = 0
	}
	if len(g.dealSpaces) == 0 {
		target := g.players[g.dealTarget]
		g.uiMgr.SetEventText(fmt.Sprintf("交易对象: %s\n没有可交易的地产 [Tab切换 Esc返回]", target.Name))
		return
	}
	if inpututil_isKeyJustPressed(ebiten.KeyLeft) {
		g.buildCursor = (g.buildCursor - 1 + len(g.dealSpaces)) % len(g.dealSpaces)
	}
	if inpututil_isKeyJustPressed(ebiten.KeyRight) {
		g.buildCursor = (g.buildCursor + 1) % len(g.dealSpaces)
	}
	if inpututil_isKeyJustPressed(ebiten.KeyEnter) {
		target := g.dealSpaces[g.buildCursor]
		buyer := g.players[g.dealTarget]
		price := target.Price
		if buyer.Money >= price {
			buyer.Money -= price
			p.Money += price
			p.RemoveProperty(target)
			buyer.AddProperty(target)
			g.uiMgr.AddMessage(fmt.Sprintf("%s 将 %s 卖给 %s ($%d)",
				p.Name, target.Name, buyer.Name, price))
			g.updateDealList()
		} else {
			g.uiMgr.AddMessage(fmt.Sprintf("%s 资金不足!", buyer.Name))
		}
	}

	if len(g.dealSpaces) > 0 && g.buildCursor < len(g.dealSpaces) {
		target := g.dealSpaces[g.buildCursor]
		buyer := g.players[g.dealTarget]
		g.uiMgr.SetEventText(fmt.Sprintf("交易 [←→选 Tab切换对象 Enter确认 Esc返回]\n卖给:%s 地产:%s 价格:$%d",
			buyer.Name, target.Name, target.Price))
	}
}

func (g *Game) updateBuildableList() {
	p := g.ActivePlayer()
	g.buildableSpaces = nil
	for _, s := range p.OwnedLands {
		if p.CanBuild(s, g.board) {
			g.buildableSpaces = append(g.buildableSpaces, s)
		}
	}
	if g.buildCursor >= len(g.buildableSpaces) {
		g.buildCursor = 0
	}
}

func (g *Game) updateMortgageList() {
	p := g.ActivePlayer()
	g.mortgageSpaces = nil
	for _, s := range p.OwnedLands {
		if !s.Mortgaged {
			g.mortgageSpaces = append(g.mortgageSpaces, s)
		}
	}
	for _, s := range p.OwnedUtilities {
		if !s.Mortgaged {
			g.mortgageSpaces = append(g.mortgageSpaces, s)
		}
	}
	for _, s := range p.OwnedTransports {
		if !s.Mortgaged {
			g.mortgageSpaces = append(g.mortgageSpaces, s)
		}
	}
	if g.buildCursor >= len(g.mortgageSpaces) {
		g.buildCursor = 0
	}
}

func (g *Game) updateRedeemList() {
	p := g.ActivePlayer()
	g.redeemSpaces = nil
	all := append(append(p.OwnedLands, p.OwnedUtilities...), p.OwnedTransports...)
	for _, s := range all {
		if s.Mortgaged {
			g.redeemSpaces = append(g.redeemSpaces, s)
		}
	}
	if g.buildCursor >= len(g.redeemSpaces) {
		g.buildCursor = 0
	}
}

func (g *Game) updateDealList() {
	p := g.ActivePlayer()
	g.dealSpaces = nil
	all := append(append(p.OwnedLands, p.OwnedUtilities...), p.OwnedTransports...)
	for _, s := range all {
		if s.Houses == 0 && !s.HasHotel && !s.Mortgaged {
			if s.Type != model.SpaceLand || !g.board.OwnsFullColorSet(p, s.Color) {
				g.dealSpaces = append(g.dealSpaces, s)
			}
		}
	}
	if g.buildCursor >= len(g.dealSpaces) {
		g.buildCursor = 0
	}
}
