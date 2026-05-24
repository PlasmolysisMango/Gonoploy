package game

import (
	"fmt"

	"github.com/PlasmolysisMango/Gonopoly/internal/model"
	"github.com/hajimehoshi/ebiten/v2"
)

func (g *Game) updateOperatingFull() {
	p := g.ActivePlayer()

	// AI 分支：跳过菜单交互，走自动决策
	if p != nil && p.IsAI {
		g.updateOperatingAI(p)
		return
	}

	switch g.menuState {
	case MenuMain:
		g.handleMenuMain(p)
	case MenuCharacter:
		g.handleMenuCharacter(p)
	case MenuSkill:
		g.handleMenuSkill(p)
	case MenuSetting:
		g.handleMenuSetting(p)
	case MenuHandle:
		g.handleMenuHandle(p)
	case MenuBuild:
		g.handleMenuBuild(p)
	case MenuMortgage:
		g.handleMenuMortgage(p)
	case MenuBuyback:
		g.handleMenuBuyback(p)
	case MenuDeal:
		g.handleMenuDeal(p)
	}
}

// ===== MAIN MENU =====
func (g *Game) enterMenuMain() {
	g.menuState = MenuMain
	s := g.board.Spaces[g.ActivePlayer().Position]
	if s.IsBuyable() {
		g.uiMgr.SetButtons("买地", "角色", "设置", "操作")
	} else {
		g.uiMgr.SetButtons("买地", "角色", "设置", "操作")
		g.uiMgr.LockBtn1(true)
	}
	if g.ActivePlayer().InJail > 0 {
		g.uiMgr.LockBtn4(true)
	}
}

func (g *Game) handleMenuMain(p *model.Player) {
	s := g.board.Spaces[p.Position]

	if g.uiMgr.Btn1Clicked() {
		if s.IsBuyable() && p.Money >= s.Price {
			g.buyProperty(p, s)
			g.uiMgr.LockBtn1(true)
		}
	}
	if g.uiMgr.Btn2Clicked() {
		g.enterMenuCharacter(p)
	}
	if g.uiMgr.Btn3Clicked() {
		g.enterMenuSetting()
	}
	if g.uiMgr.Btn4Clicked() {
		g.enterMenuHandle()
	}
	if g.uiMgr.EndTurnClicked() {
		g.uiMgr.SetEventText("")
		g.turnPhase = TurnEndTurn
	}
}

// ===== CHARACTER MENU =====
func (g *Game) enterMenuCharacter(p *model.Player) {
	g.menuState = MenuCharacter
	g.uiMgr.SetButtons("技能", "祝福", "", "返回")
	g.showCharacterInfo(p)
}

func (g *Game) handleMenuCharacter(p *model.Player) {
	if g.uiMgr.Btn1Clicked() {
		g.enterMenuSkill(p)
	}
	if g.uiMgr.Btn2Clicked() {
		g.showBlessingInfo(p)
	}
	if g.uiMgr.Btn4Clicked() {
		g.uiMgr.SetEventText("")
		g.enterMenuMain()
	}
}

func (g *Game) showCharacterInfo(p *model.Player) {
	info := fmt.Sprintf("角色：%s", p.Name)
	info += fmt.Sprintf("\n金钱：%d元", p.Money)
	info += fmt.Sprintf("\n技能点：%d", p.SkillPoints)
	info += fmt.Sprintf("\n地产：%d 公共：%d 交通：%d",
		len(p.OwnedLands), len(p.OwnedUtilities), len(p.OwnedTransports))
	skill, ok := CharacterSkills[p.Name]
	if ok {
		info += fmt.Sprintf("\n技能：%s（消耗%d点）", skill.Name, skill.Cost)
		info += "\n" + skill.Describe
	}
	g.uiMgr.SetEventText(info)
}

func (g *Game) showBlessingInfo(p *model.Player) {
	if len(p.Blessings) == 0 {
		g.uiMgr.SetEventText("当前没有祝福效果")
		return
	}
	info := "当前祝福：\n"
	for _, b := range p.Blessings {
		effect := "减少50%"
		if b.Modifier == "增加" {
			effect = "增加50%"
		}
		info += fmt.Sprintf("  %s：%s\n", b.Category, effect)
	}
	g.uiMgr.SetEventText(info)
}

// ===== SKILL MENU =====
func (g *Game) enterMenuSkill(p *model.Player) {
	g.menuState = MenuSkill
	g.uiMgr.SetButtons("确认使用", "取消", "", "返回")
	skill, ok := CharacterSkills[p.Name]
	if !ok || p.SkillPoints < skill.Cost {
		g.uiMgr.LockBtn1(true)
	}
	info := ""
	if ok {
		info = fmt.Sprintf("技能：%s\n消耗：%d点\n效果：%s\n当前技能点：%d",
			skill.Name, skill.Cost, skill.Describe, p.SkillPoints)
	} else {
		info = "该角色没有技能"
	}
	g.uiMgr.SetEventText(info)
}

func (g *Game) handleMenuSkill(p *model.Player) {
	if g.uiMgr.Btn1Clicked() {
		g.useSkill(p)
		g.enterMenuCharacter(p)
	}
	if g.uiMgr.Btn2Clicked() || g.uiMgr.Btn4Clicked() {
		g.enterMenuCharacter(p)
	}
}

// ===== SETTING MENU =====
func (g *Game) enterMenuSetting() {
	g.menuState = MenuSetting
	musicLabel := "音乐：开"
	if g.audioMgr != nil && g.audioMgr.IsMuted() {
		musicLabel = "音乐：关"
	}
	g.uiMgr.SetButtons(musicLabel, "保存", "读取", "返回")
	g.uiMgr.SetEventText("游戏设置")
}

func (g *Game) handleMenuSetting(p *model.Player) {
	if g.uiMgr.Btn1Clicked() {
		if g.audioMgr != nil {
			g.audioMgr.ToggleMute()
			if g.audioMgr.IsMuted() {
				g.uiMgr.AddMessage("音乐已关闭")
				g.uiMgr.SetButtons("音乐：关", "保存", "读取", "返回")
			} else {
				g.uiMgr.AddMessage("音乐已开启")
				g.uiMgr.SetButtons("音乐：开", "保存", "读取", "返回")
			}
		}
	}
	if g.uiMgr.Btn2Clicked() {
		if err := g.Save("saves/manual.json"); err == nil {
			g.uiMgr.AddMessage("游戏已保存")
		}
	}
	if g.uiMgr.Btn3Clicked() {
		if err := g.Load("saves/manual.json"); err == nil {
			g.uiMgr.AddMessage("游戏已加载")
		}
	}
	if g.uiMgr.Btn4Clicked() {
		g.uiMgr.SetEventText("")
		g.enterMenuMain()
	}
}

// ===== HANDLE MENU (操作) =====
func (g *Game) enterMenuHandle() {
	g.menuState = MenuHandle
	g.uiMgr.SetButtons("加盖", "交易", "抵押/赎回", "返回")
	g.uiMgr.SetEventText("选择操作")
}

func (g *Game) handleMenuHandle(p *model.Player) {
	if g.uiMgr.Btn1Clicked() {
		g.enterMenuBuild(p)
	}
	if g.uiMgr.Btn2Clicked() {
		g.enterMenuDeal(p)
	}
	if g.uiMgr.Btn3Clicked() {
		g.enterMenuMortgage(p)
	}
	if g.uiMgr.Btn4Clicked() {
		g.uiMgr.SetEventText("")
		g.enterMenuMain()
	}
}

// ===== BUILD MENU =====
func (g *Game) enterMenuBuild(p *model.Player) {
	g.menuState = MenuBuild
	g.buildCursor = 0
	g.updateBuildableList()
	g.uiMgr.SetButtons("确定", "取消", "", "返回")
	if len(g.buildableSpaces) == 0 {
		g.uiMgr.LockBtn1(true)
		g.uiMgr.SetEventText("没有可建造的地产")
	} else {
		g.showBuildInfo()
	}
}

func (g *Game) handleMenuBuild(p *model.Player) {
	if g.uiMgr.Btn1Clicked() && len(g.buildableSpaces) > 0 {
		target := g.buildableSpaces[g.buildCursor]
		cost := target.BuildCost()
		if p.Money >= cost {
			p.Money -= cost
			if target.Houses < 4 {
				target.Houses++
				g.uiMgr.AddMessage(fmt.Sprintf("%s 在 %s 建造房屋(Lv%d) $%d",
					p.Name, target.Name, target.Houses, cost))
			} else {
				target.HasHotel = true
				target.Houses = 0
				g.uiMgr.AddMessage(fmt.Sprintf("%s 在 %s 建造酒店! $%d",
					p.Name, target.Name, cost))
			}
			g.updateBuildableList()
			if len(g.buildableSpaces) == 0 {
				g.uiMgr.LockBtn1(true)
				g.uiMgr.SetEventText("没有更多可建造的地产")
			} else {
				g.showBuildInfo()
			}
		} else {
			g.uiMgr.AddMessage("资金不足!")
		}
	}
	if g.uiMgr.Btn2Clicked() {
		g.enterMenuHandle()
	}
	if g.uiMgr.Btn4Clicked() {
		g.enterMenuHandle()
	}
	if inpututil_isKeyJustPressed(ebiten.KeyLeft) && len(g.buildableSpaces) > 0 {
		g.buildCursor = (g.buildCursor - 1 + len(g.buildableSpaces)) % len(g.buildableSpaces)
		g.showBuildInfo()
	}
	if inpututil_isKeyJustPressed(ebiten.KeyRight) && len(g.buildableSpaces) > 0 {
		g.buildCursor = (g.buildCursor + 1) % len(g.buildableSpaces)
		g.showBuildInfo()
	}
}

func (g *Game) showBuildInfo() {
	if g.buildCursor >= len(g.buildableSpaces) {
		g.buildCursor = 0
	}
	target := g.buildableSpaces[g.buildCursor]
	level := target.Houses
	if target.HasHotel {
		level = 5
	}
	g.uiMgr.SetEventText(fmt.Sprintf("建造 [←→选择地产]\n地产：%s\n当前等级：%d\n建造费用：%d元\n(%d/%d)",
		target.Name, level, target.BuildCost(), g.buildCursor+1, len(g.buildableSpaces)))
}

// ===== MORTGAGE MENU =====
func (g *Game) enterMenuMortgage(p *model.Player) {
	g.menuState = MenuMortgage
	g.buildCursor = 0
	g.updateMortgageList()
	g.uiMgr.SetButtons("确定", "取消", "", "返回")
	if len(g.mortgageSpaces) == 0 {
		g.uiMgr.LockBtn1(true)
		g.uiMgr.SetEventText("没有可抵押的地产")
	} else {
		g.showMortgageInfo()
	}
}

func (g *Game) handleMenuMortgage(p *model.Player) {
	if g.uiMgr.Btn1Clicked() && len(g.mortgageSpaces) > 0 {
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
		if len(g.mortgageSpaces) == 0 {
			g.uiMgr.LockBtn1(true)
			g.uiMgr.SetEventText("没有更多可抵押的地产")
		} else {
			g.showMortgageInfo()
		}
	}
	if g.uiMgr.Btn2Clicked() || g.uiMgr.Btn4Clicked() {
		g.enterMenuHandle()
	}
	if inpututil_isKeyJustPressed(ebiten.KeyLeft) && len(g.mortgageSpaces) > 0 {
		g.buildCursor = (g.buildCursor - 1 + len(g.mortgageSpaces)) % len(g.mortgageSpaces)
		g.showMortgageInfo()
	}
	if inpututil_isKeyJustPressed(ebiten.KeyRight) && len(g.mortgageSpaces) > 0 {
		g.buildCursor = (g.buildCursor + 1) % len(g.mortgageSpaces)
		g.showMortgageInfo()
	}
}

func (g *Game) showMortgageInfo() {
	if g.buildCursor >= len(g.mortgageSpaces) {
		g.buildCursor = 0
	}
	target := g.mortgageSpaces[g.buildCursor]
	val := target.Price / 2
	if target.Houses > 0 {
		val = target.Houses * target.BuildCost() / 2
	}
	g.uiMgr.SetEventText(fmt.Sprintf("抵押 [←→选择地产]\n地产：%s\n可获得：%d元\n(%d/%d)",
		target.Name, val, g.buildCursor+1, len(g.mortgageSpaces)))
}

// ===== BUYBACK MENU =====
func (g *Game) enterMenuBuyback(p *model.Player) {
	g.menuState = MenuBuyback
	g.buildCursor = 0
	g.updateRedeemList()
	g.uiMgr.SetButtons("确定", "取消", "", "返回")
	if len(g.redeemSpaces) == 0 {
		g.uiMgr.LockBtn1(true)
		g.uiMgr.SetEventText("没有可赎回的地产")
	} else {
		g.showBuybackInfo()
	}
}

func (g *Game) handleMenuBuyback(p *model.Player) {
	if g.uiMgr.Btn1Clicked() && len(g.redeemSpaces) > 0 {
		target := g.redeemSpaces[g.buildCursor]
		cost := target.Price * 3 / 5
		if p.Money >= cost {
			p.Money -= cost
			target.Mortgaged = false
			g.uiMgr.AddMessage(fmt.Sprintf("%s 赎回 %s $%d", p.Name, target.Name, cost))
			g.updateRedeemList()
			if len(g.redeemSpaces) == 0 {
				g.uiMgr.LockBtn1(true)
				g.uiMgr.SetEventText("没有更多可赎回的地产")
			} else {
				g.showBuybackInfo()
			}
		} else {
			g.uiMgr.AddMessage("资金不足!")
		}
	}
	if g.uiMgr.Btn2Clicked() || g.uiMgr.Btn4Clicked() {
		g.enterMenuHandle()
	}
	if inpututil_isKeyJustPressed(ebiten.KeyLeft) && len(g.redeemSpaces) > 0 {
		g.buildCursor = (g.buildCursor - 1 + len(g.redeemSpaces)) % len(g.redeemSpaces)
		g.showBuybackInfo()
	}
	if inpututil_isKeyJustPressed(ebiten.KeyRight) && len(g.redeemSpaces) > 0 {
		g.buildCursor = (g.buildCursor + 1) % len(g.redeemSpaces)
		g.showBuybackInfo()
	}
}

func (g *Game) showBuybackInfo() {
	if g.buildCursor >= len(g.redeemSpaces) {
		g.buildCursor = 0
	}
	target := g.redeemSpaces[g.buildCursor]
	cost := target.Price * 3 / 5
	g.uiMgr.SetEventText(fmt.Sprintf("赎回 [←→选择地产]\n地产：%s\n赎回费用：%d元\n(%d/%d)",
		target.Name, cost, g.buildCursor+1, len(g.redeemSpaces)))
}

// ===== DEAL MENU =====
func (g *Game) enterMenuDeal(p *model.Player) {
	g.menuState = MenuDeal
	g.buildCursor = 0
	g.dealTarget = -1
	for i, other := range g.players {
		if i != g.activeIdx && !other.Bankrupt {
			g.dealTarget = i
			break
		}
	}
	g.updateDealList()
	g.uiMgr.SetButtons("确定", "切换对象", "", "返回")
	if len(g.dealSpaces) == 0 || g.dealTarget < 0 {
		g.uiMgr.LockBtn1(true)
	}
	g.showDealInfo()
}

func (g *Game) handleMenuDeal(p *model.Player) {
	if g.uiMgr.Btn1Clicked() && len(g.dealSpaces) > 0 && g.dealTarget >= 0 {
		target := g.dealSpaces[g.buildCursor]
		buyer := g.players[g.dealTarget]
		price := target.Price
		// 若交易对象为 AI，则由 AI 自动判断是否接受
		if buyer.IsAI {
			if buyer.Money-price >= 200 {
				buyer.Money -= price
				p.Money += price
				p.RemoveProperty(target)
				buyer.AddProperty(target)
				g.uiMgr.AddMessage(fmt.Sprintf("%s 将 %s 卖给 %s ($%d)",
					p.Name, target.Name, buyer.Name, price))
			} else {
				g.uiMgr.AddMessage(fmt.Sprintf("[AI]%s 拒绝了交易", buyer.Name))
			}
			g.updateDealList()
			if len(g.dealSpaces) == 0 {
				g.uiMgr.LockBtn1(true)
			}
			g.showDealInfo()
		} else if buyer.Money >= price {
			buyer.Money -= price
			p.Money += price
			p.RemoveProperty(target)
			buyer.AddProperty(target)
			g.uiMgr.AddMessage(fmt.Sprintf("%s 将 %s 卖给 %s ($%d)",
				p.Name, target.Name, buyer.Name, price))
			g.updateDealList()
			if len(g.dealSpaces) == 0 {
				g.uiMgr.LockBtn1(true)
			}
			g.showDealInfo()
		} else {
			g.uiMgr.AddMessage(fmt.Sprintf("%s 资金不足!", buyer.Name))
		}
	}
	if g.uiMgr.Btn2Clicked() {
		for {
			g.dealTarget = (g.dealTarget + 1) % len(g.players)
			if g.dealTarget != g.activeIdx && !g.players[g.dealTarget].Bankrupt {
				break
			}
		}
		g.buildCursor = 0
		g.updateDealList()
		if len(g.dealSpaces) == 0 {
			g.uiMgr.LockBtn1(true)
		} else {
			g.uiMgr.LockBtn1(false)
		}
		g.showDealInfo()
	}
	if g.uiMgr.Btn4Clicked() {
		g.enterMenuHandle()
	}
	if inpututil_isKeyJustPressed(ebiten.KeyLeft) && len(g.dealSpaces) > 0 {
		g.buildCursor = (g.buildCursor - 1 + len(g.dealSpaces)) % len(g.dealSpaces)
		g.showDealInfo()
	}
	if inpututil_isKeyJustPressed(ebiten.KeyRight) && len(g.dealSpaces) > 0 {
		g.buildCursor = (g.buildCursor + 1) % len(g.dealSpaces)
		g.showDealInfo()
	}
}

func (g *Game) showDealInfo() {
	if g.dealTarget < 0 {
		g.uiMgr.SetEventText("没有可交易的对象")
		return
	}
	buyer := g.players[g.dealTarget]
	if len(g.dealSpaces) == 0 {
		g.uiMgr.SetEventText(fmt.Sprintf("交易对象：%s\n没有可交易的地产", buyer.Name))
		return
	}
	if g.buildCursor >= len(g.dealSpaces) {
		g.buildCursor = 0
	}
	target := g.dealSpaces[g.buildCursor]
	g.uiMgr.SetEventText(fmt.Sprintf("交易 [←→选择地产]\n对象：%s\n地产：%s\n价格：%d元\n(%d/%d)",
		buyer.Name, target.Name, target.Price, g.buildCursor+1, len(g.dealSpaces)))
}

// ===== HELPER FUNCTIONS =====
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

// ===== AI OPERATING =====
func (g *Game) updateOperatingAI(p *model.Player) {
	g.aiThinkTimer++
	if g.aiThinkTimer < AIThinkTicks {
		return
	}
	g.aiThinkTimer = 0

	action, target := g.aiDecide()
	switch action {
	case "buy":
		if target != nil && target.IsBuyable() && p.Money >= target.Price {
			g.uiMgr.AddMessage(fmt.Sprintf("[AI]%s 购买 %s", p.Name, target.Name))
			g.buyProperty(p, target)
		}
	case "build":
		if target != nil {
			g.aiBuild(p, target)
		}
	case "skill":
		g.uiMgr.AddMessage(fmt.Sprintf("[AI]%s 使用技能", p.Name))
		g.useSkill(p)
	case "mortgage":
		if target != nil {
			g.aiMortgage(p, target)
		}
	case "end_turn":
		fallthrough
	default:
		g.uiMgr.SetEventText("")
		g.uiMgr.AddMessage(fmt.Sprintf("[AI]%s 结束回合", p.Name))
		g.turnPhase = TurnEndTurn
	}
}

func (g *Game) aiBuild(p *model.Player, target *model.Space) {
	cost := target.BuildCost()
	if p.Money < cost {
		return
	}
	p.Money -= cost
	if target.Houses < 4 {
		target.Houses++
		g.uiMgr.AddMessage(fmt.Sprintf("[AI]%s 在 %s 建造房屋(Lv%d) $%d",
			p.Name, target.Name, target.Houses, cost))
	} else {
		target.HasHotel = true
		target.Houses = 0
		g.uiMgr.AddMessage(fmt.Sprintf("[AI]%s 在 %s 建造酒店! $%d",
			p.Name, target.Name, cost))
	}
}

func (g *Game) aiMortgage(p *model.Player, target *model.Space) {
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
	g.uiMgr.AddMessage(fmt.Sprintf("[AI]%s 抵押 %s 获得$%d", p.Name, target.Name, refund))
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
