package game

import (
	"fmt"
	"math/rand"

	"github.com/PlasmolysisMango/Gonopoly/internal/model"
)

func (g *Game) resolveChance(p *model.Player) {
	weights := []int{5, 5, 3, 3, 2, 2, 2, 1, 3, 2, 3, 3, 2, 2}
	totalWeight := 0
	for _, w := range weights {
		totalWeight += w
	}

	roll := rand.Intn(totalWeight)
	cumulative := 0
	eventType := 0
	for i, w := range weights {
		cumulative += w
		if roll < cumulative {
			eventType = i + 1
			break
		}
	}

	switch eventType {
	case 1: // 获得金钱
		amount := (rand.Intn(4) + 1) * 50
		p.Money += amount
		g.uiMgr.AddMessage(fmt.Sprintf("机会: %s 获得$%d", p.Name, amount))
	case 2: // 损失金钱
		amount := (rand.Intn(4) + 1) * 50
		p.Money -= amount
		g.uiMgr.AddMessage(fmt.Sprintf("机会: %s 损失$%d", p.Name, amount))
		if p.Money < 0 {
			g.handleBankruptcy(p, nil)
		}
	case 3: // 退税
		assets := p.TotalAssets() - p.Money
		refund := assets / 10
		p.Money += refund
		g.uiMgr.AddMessage(fmt.Sprintf("机会: 退税! %s 获得$%d", p.Name, refund))
	case 4: // 财产税
		assets := p.TotalAssets() - p.Money
		tax := assets / 10
		p.Money -= tax
		g.uiMgr.AddMessage(fmt.Sprintf("机会: 财产税! %s 缴纳$%d", p.Name, tax))
		if p.Money < 0 {
			g.handleBankruptcy(p, nil)
		}
	case 5: // 免费建造
		g.chanceFreeeBuild(p)
	case 6: // 强制拆迁
		g.chanceDemolish(p)
	case 7: // 全员建造
		g.chanceGlobalBuild()
	case 8: // 全员拆迁
		g.chanceGlobalDemolish()
	case 9: // 获得地产
		g.chanceFreeLand(p)
	case 10: // 失去地产
		g.chanceLoseLand(p)
	case 11: // 慈善
		amount := (rand.Intn(3) + 1) * 60
		p.Money -= amount
		share := amount / (g.alivePlayers() - 1)
		for _, other := range g.players {
			if other != p && !other.Bankrupt {
				other.Money += share
			}
		}
		g.uiMgr.AddMessage(fmt.Sprintf("机会: %s 慈善捐款$%d", p.Name, amount))
		if p.Money < 0 {
			g.handleBankruptcy(p, nil)
		}
	case 12: // 出狱卡
		p.JailPassports++
		g.uiMgr.AddMessage(fmt.Sprintf("机会: %s 获得出狱卡!", p.Name))
	case 13: // 入狱
		g.sendToJail(p)
		g.uiMgr.AddMessage(fmt.Sprintf("机会: %s 被送入监狱!", p.Name))
	case 14: // 随机传送
		newPos := rand.Intn(g.board.SpaceCount())
		p.Position = newPos
		g.uiMgr.AddMessage(fmt.Sprintf("机会: %s 被传送到 %s", p.Name, g.board.Spaces[newPos].Name))
	}
}

func (g *Game) chanceFreeeBuild(p *model.Player) {
	for _, s := range p.OwnedLands {
		if p.CanBuild(s, g.board) && !s.HasHotel {
			if s.Houses < 4 {
				s.Houses++
			} else {
				s.HasHotel = true
				s.Houses = 0
			}
			g.uiMgr.AddMessage(fmt.Sprintf("机会: %s 免费建造于 %s", p.Name, s.Name))
			return
		}
	}
	g.uiMgr.AddMessage(fmt.Sprintf("机会: %s 没有可建造的地产", p.Name))
}

func (g *Game) chanceDemolish(p *model.Player) {
	for _, s := range p.OwnedLands {
		if s.HasHotel {
			s.HasHotel = false
			s.Houses = 4
			g.uiMgr.AddMessage(fmt.Sprintf("机会: %s 的 %s 酒店被拆除", p.Name, s.Name))
			return
		}
		if s.Houses > 0 {
			s.Houses--
			g.uiMgr.AddMessage(fmt.Sprintf("机会: %s 的 %s 被拆除一栋房屋", p.Name, s.Name))
			return
		}
	}
	g.uiMgr.AddMessage(fmt.Sprintf("机会: %s 没有可拆除的建筑", p.Name))
}

func (g *Game) chanceGlobalBuild() {
	for _, p := range g.players {
		if p.Bankrupt {
			continue
		}
		for _, s := range p.OwnedLands {
			if p.CanBuild(s, g.board) && !s.HasHotel && s.Houses < 4 {
				s.Houses++
				g.uiMgr.AddMessage(fmt.Sprintf("全员建造: %s 的 %s 新增房屋", p.Name, s.Name))
				break
			}
		}
	}
}

func (g *Game) chanceGlobalDemolish() {
	for _, p := range g.players {
		if p.Bankrupt {
			continue
		}
		for _, s := range p.OwnedLands {
			if s.Houses > 0 {
				s.Houses--
				g.uiMgr.AddMessage(fmt.Sprintf("全员拆迁: %s 的 %s 房屋被拆", p.Name, s.Name))
				break
			}
		}
	}
}

func (g *Game) chanceFreeLand(p *model.Player) {
	var unowned []*model.Space
	for _, s := range g.board.Spaces {
		if s.IsBuyable() {
			unowned = append(unowned, s)
		}
	}
	if len(unowned) > 0 {
		s := unowned[rand.Intn(len(unowned))]
		p.AddProperty(s)
		g.uiMgr.AddMessage(fmt.Sprintf("机会: %s 免费获得 %s", p.Name, s.Name))
	} else {
		g.uiMgr.AddMessage("机会: 没有空地可分配")
	}
}

func (g *Game) chanceLoseLand(p *model.Player) {
	all := append(append(p.OwnedLands, p.OwnedUtilities...), p.OwnedTransports...)
	if len(all) > 0 {
		s := all[rand.Intn(len(all))]
		p.RemoveProperty(s)
		s.Houses = 0
		s.HasHotel = false
		s.Mortgaged = false
		g.uiMgr.AddMessage(fmt.Sprintf("机会: %s 失去了 %s", p.Name, s.Name))
	} else {
		g.uiMgr.AddMessage(fmt.Sprintf("机会: %s 没有地产可失去", p.Name))
	}
}

func (g *Game) resolveBlessing(p *model.Player) {
	categories := []string{"买地", "过路", "抵押", "加盖"}
	modifiers := []string{"增加", "减少"}
	cat := categories[rand.Intn(len(categories))]
	mod := modifiers[rand.Intn(len(modifiers))]

	blessing := model.Blessing{Category: cat, Modifier: mod}
	p.Blessings = append(p.Blessings, blessing)

	effect := "减少50%"
	if mod == "增加" {
		effect = "增加50%"
	}
	g.uiMgr.AddMessage(fmt.Sprintf("祝福: %s 下次%s费用%s", p.Name, cat, effect))
}
