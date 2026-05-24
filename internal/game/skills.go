package game

import (
	"fmt"
	"math/rand"

	"github.com/PlasmolysisMango/Gonopoly/internal/model"
)

type SkillDef struct {
	Name     string
	Cost     int
	Describe string
}

var CharacterSkills = map[string]SkillDef{
	"食蜂": {Name: "心理掌握", Cost: 5, Describe: "窃取其他玩家一块未开发地产"},
	"黑子": {Name: "空间移动", Cost: 2, Describe: "随机传送到一块地产"},
	"泪子": {Name: "预知脚本", Cost: 1, Describe: "触发一个有利的机会事件"},
	"警策": {Name: "液化暗影", Cost: 2, Describe: "获得额外一次行动机会"},
}

func (g *Game) useSkill(p *model.Player) {
	skill, ok := CharacterSkills[p.Name]
	if !ok {
		g.uiMgr.AddMessage(fmt.Sprintf("%s 没有技能", p.Name))
		return
	}
	if p.SkillPoints < skill.Cost {
		g.uiMgr.AddMessage(fmt.Sprintf("%s 技能点不足 (需要%d, 拥有%d)", p.Name, skill.Cost, p.SkillPoints))
		return
	}

	p.SkillPoints -= skill.Cost
	g.uiMgr.AddMessage(fmt.Sprintf("%s 使用技能: %s", p.Name, skill.Name))

	switch p.Name {
	case "食蜂":
		g.skillSteal(p)
	case "黑子":
		g.skillTeleport(p)
	case "泪子":
		g.skillFavorableChance(p)
	case "警策":
		g.skillExtraTurn(p)
	}
}

func (g *Game) skillSteal(p *model.Player) {
	var candidates []*model.Space
	for _, other := range g.players {
		if other == p || other.Bankrupt {
			continue
		}
		for _, s := range other.OwnedLands {
			if s.Houses == 0 && !s.HasHotel && !s.Mortgaged {
				candidates = append(candidates, s)
			}
		}
	}
	if len(candidates) == 0 {
		g.uiMgr.AddMessage("没有可窃取的地产")
		p.SkillPoints += CharacterSkills["食蜂"].Cost
		return
	}
	target := candidates[rand.Intn(len(candidates))]
	oldOwner := target.Owner
	oldOwner.RemoveProperty(target)
	p.AddProperty(target)
	g.uiMgr.AddMessage(fmt.Sprintf("从 %s 窃取了 %s!", oldOwner.Name, target.Name))
}

func (g *Game) skillTeleport(p *model.Player) {
	var lands []int
	for i, s := range g.board.Spaces {
		if s.Type == model.SpaceLand {
			lands = append(lands, i)
		}
	}
	if len(lands) == 0 {
		return
	}
	dest := lands[rand.Intn(len(lands))]
	p.Position = dest
	g.uiMgr.AddMessage(fmt.Sprintf("传送到 %s!", g.board.Spaces[dest].Name))
}

func (g *Game) skillFavorableChance(p *model.Player) {
	favorable := []int{1, 3, 5, 7, 9, 12}
	pick := favorable[rand.Intn(len(favorable))]
	g.resolveFavorableChance(p, pick)
}

func (g *Game) resolveFavorableChance(p *model.Player, eventType int) {
	switch eventType {
	case 1:
		amount := (rand.Intn(4) + 1) * 50
		p.Money += amount
		g.uiMgr.AddMessage(fmt.Sprintf("预知: %s 获得$%d", p.Name, amount))
	case 3:
		assets := p.TotalAssets() - p.Money
		refund := assets / 10
		p.Money += refund
		g.uiMgr.AddMessage(fmt.Sprintf("预知: 退税! 获得$%d", refund))
	case 5:
		g.chanceFreeeBuild(p)
	case 7:
		g.chanceGlobalBuild()
	case 9:
		g.chanceFreeLand(p)
	case 12:
		p.JailPassports++
		g.uiMgr.AddMessage(fmt.Sprintf("预知: %s 获得出狱卡!", p.Name))
	}
}

func (g *Game) skillExtraTurn(p *model.Player) {
	if p.BonusCount > 0 {
		g.uiMgr.AddMessage("已有额外回合，技能无效")
		p.SkillPoints += CharacterSkills["警策"].Cost
		return
	}
	g.dice.IsDouble = true
	g.uiMgr.AddMessage("获得额外一次行动!")
}
