package game

import (
	"math/rand"

	"github.com/PlasmolysisMango/Gonopoly/internal/model"
)

// AIThinkTicks 表示 AI 决策延迟（约 0.5 秒 @ 60FPS）
const AIThinkTicks = 30

// aiShouldBuyLand 决定 AI 是否应该购买当前所在的可买地块
func (g *Game) aiShouldBuyLand(p *model.Player, s *model.Space) bool {
	if p == nil || s == nil {
		return false
	}
	if !s.IsBuyable() || p.Money < s.Price {
		return false
	}

	// 同色组分析（仅对 SpaceLand 有意义）
	if s.Type == model.SpaceLand && s.Color != "" && s.Color != "NONE" {
		group := g.board.ColorSets[s.Color]
		ownedInGroup := 0
		unownedInGroup := 0
		for _, sp := range group {
			if sp == s {
				continue
			}
			if sp.Owner == p {
				ownedInGroup++
			}
			if sp.Owner == nil {
				unownedInGroup++
			}
		}
		// 买下后即可完成同色组 → 必买
		if unownedInGroup == 0 && ownedInGroup == len(group)-1 {
			return true
		}
		// 已经拥有同色组中的至少一块
		if ownedInGroup > 0 {
			if p.Money > s.Price*3/2 {
				return true
			}
		}
	}

	// 资金充裕直接买
	if p.Money > s.Price*2 {
		return true
	}
	// 资金紧张不买
	if p.Money < s.Price*12/10 {
		return false
	}
	// 默认偏向购买
	return true
}

// aiChooseBuild 决定 AI 要在哪一块自有地产上建造（nil 表示不建造）
func (g *Game) aiChooseBuild(p *model.Player, board *model.Board) *model.Space {
	if p == nil || board == nil {
		return nil
	}

	var best *model.Space
	for _, s := range p.OwnedLands {
		if !p.CanBuild(s, board) {
			continue
		}
		cost := s.BuildCost()
		if p.Money <= cost*3 {
			continue
		}
		// 选择房屋数最少的地块（均匀升级）
		if best == nil || s.Houses < best.Houses {
			best = s
		}
	}
	return best
}

// aiShouldUseSkill 决定 AI 是否应该使用技能
func (g *Game) aiShouldUseSkill(p *model.Player, players []*model.Player) bool {
	if p == nil {
		return false
	}
	skill, ok := CharacterSkills[p.Name]
	if !ok {
		return false
	}
	if p.SkillPoints < skill.Cost {
		return false
	}

	switch p.Name {
	case "食蜂":
		// 心理掌握：是否有对手拥有未开发地块
		for _, other := range players {
			if other == p || other.Bankrupt {
				continue
			}
			for _, s := range other.OwnedLands {
				if s.Houses == 0 && !s.HasHotel && !s.Mortgaged {
					return true
				}
			}
		}
		return false
	case "黑子":
		return rand.Float64() < 0.5
	case "泪子":
		return rand.Float64() < 0.7
	case "警策":
		return rand.Float64() < 0.6
	}
	return false
}

// aiChooseMortgage 决定 AI 要抵押哪一块地产（nil 表示不抵押）
func (g *Game) aiChooseMortgage(p *model.Player, board *model.Board) *model.Space {
	if p == nil || board == nil {
		return nil
	}
	if p.Money >= 100 {
		return nil
	}

	// 优先抵押：未抵押 + 没有房屋 + 不属于完整组的单块地
	for _, s := range p.OwnedLands {
		if s.Mortgaged || s.Houses > 0 || s.HasHotel {
			continue
		}
		if board.OwnsFullColorSet(p, s.Color) {
			continue
		}
		return s
	}
	// 其次考虑公共/交通设施（同样未抵押）
	for _, s := range p.OwnedUtilities {
		if !s.Mortgaged {
			return s
		}
	}
	for _, s := range p.OwnedTransports {
		if !s.Mortgaged {
			return s
		}
	}
	return nil
}

// aiDecide 是 AI 的总决策入口，返回下一步要执行的动作
// action 取值: "buy" / "build" / "skill" / "mortgage" / "end_turn"
func (g *Game) aiDecide() (string, *model.Space) {
	p := g.ActivePlayer()
	if p == nil {
		return "end_turn", nil
	}

	// 1. 买地（仅当玩家所在地块是可买地块）
	if p.Position >= 0 && p.Position < len(g.board.Spaces) {
		cur := g.board.Spaces[p.Position]
		if cur.IsBuyable() && p.Money >= cur.Price && g.aiShouldBuyLand(p, cur) {
			return "buy", cur
		}
	}

	// 2. 建造
	if target := g.aiChooseBuild(p, g.board); target != nil {
		return "build", target
	}

	// 3. 技能
	if g.aiShouldUseSkill(p, g.players) {
		return "skill", nil
	}

	// 4. 抵押
	if target := g.aiChooseMortgage(p, g.board); target != nil {
		return "mortgage", target
	}

	// 5. 结束回合
	return "end_turn", nil
}
