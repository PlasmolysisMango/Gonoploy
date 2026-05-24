// Package server - game_logic.go 服务端独立游戏逻辑驱动器。
//
// ServerGame 不依赖 Ebiten，纯粹按消息驱动游戏状态：
//   - HandleAction: 接收玩家操作消息，执行逻辑并返回需要广播的消息列表。
//   - GetFullState: 用于加入/重连时下发完整状态。
//
// 所有相位/状态常量值与 internal/game 包中的 TurnPhase / GameState 保持一致。
package server

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math/rand"

	"github.com/PlasmolysisMango/Gonopoly/internal/model"
	"github.com/PlasmolysisMango/Gonopoly/pkg/protocol"
)

//go:embed buildings.config
var serverBuildingsConfig string

// Phase* 与 game.TurnPhase 枚举值保持一致。
const (
	PhaseWaitRoll  = 0
	PhaseRolling   = 1
	PhaseJumping   = 2
	PhaseMoving    = 3
	PhaseLanded    = 4
	PhaseOperating = 5
	PhaseEndTurn   = 6
)

// State* 与 game.GameState 中的对局相关枚举值保持一致。
const (
	StateCharSelect = 0
	StatePlaying    = 1
	StateGameOver   = 2
)

// skillDef 服务端角色技能定义（与 game.CharacterSkills 等价，避免循环依赖）。
type skillDef struct {
	Name string
	Cost int
}

var serverCharSkills = map[string]skillDef{
	"食蜂": {Name: "心理掌握", Cost: 5},
	"黑子": {Name: "空间移动", Cost: 2},
	"泪子": {Name: "预知脚本", Cost: 1},
	"警策": {Name: "液化暗影", Cost: 2},
}

// ServerGame 服务端独立运行的游戏逻辑驱动器。
type ServerGame struct {
	players   []*model.Player
	charNames []string       // 与 players 平行，存放角色名（用于技能查表/客户端图标）
	nameIndex map[string]int // 房间玩家名 -> 下标

	board *model.Board
	dice  *model.Dice

	state     int
	turnPhase int
	activeIdx int

	messages [][]byte // 当前动作产生的待广播消息
}

// NewServerGame 用房间玩家列表创建一个新的服务端对局。
func NewServerGame(players []protocol.RoomPlayer) *ServerGame {
	sg := &ServerGame{
		board:     model.ParseBoard(serverBuildingsConfig),
		dice:      model.NewDice(),
		state:     StatePlaying,
		turnPhase: PhaseWaitRoll,
		activeIdx: 0,
		nameIndex: make(map[string]int, len(players)),
	}
	for i, rp := range players {
		char := rp.CharName
		if char == "" {
			char = rp.Name
		}
		// 服务端不需要图标，使用 nil；Direction 用下标即可。
		p := model.NewPlayer(rp.Name, nil, i)
		sg.players = append(sg.players, p)
		sg.charNames = append(sg.charNames, char)
		sg.nameIndex[rp.Name] = i
	}
	return sg
}

// Phase 返回当前回合阶段（用于外部诊断）。
func (sg *ServerGame) Phase() int { return sg.turnPhase }

// State 返回当前游戏状态（playing/over）。
func (sg *ServerGame) State() int { return sg.state }

// ActiveIdx 返回当前活动玩家下标。
func (sg *ServerGame) ActiveIdx() int { return sg.activeIdx }

// HandleAction 根据消息类型驱动游戏逻辑。返回需要广播给房间内所有客户端的消息列表。
// 验证失败时静默忽略（返回空切片）。
func (sg *ServerGame) HandleAction(playerName string, msgType string, payload []byte) [][]byte {
	sg.messages = nil
	if sg.state != StatePlaying {
		return nil
	}
	idx, ok := sg.nameIndex[playerName]
	if !ok {
		return nil
	}
	if idx != sg.activeIdx {
		return nil
	}
	p := sg.players[idx]
	if p == nil || p.Bankrupt {
		return nil
	}

	switch msgType {
	case protocol.TypeRollDice:
		if sg.turnPhase != PhaseWaitRoll {
			return sg.messages
		}
		sg.executeRoll()
		sg.processAITurns()

	case protocol.TypeBuyProperty:
		if sg.turnPhase != PhaseOperating {
			return sg.messages
		}
		sg.executeBuy()

	case protocol.TypeBuild:
		if sg.turnPhase != PhaseOperating {
			return sg.messages
		}
		var c protocol.C2SBuild
		_ = json.Unmarshal(payload, &c)
		sg.executeBuild(c.SpaceID)

	case protocol.TypeUseSkill:
		if sg.turnPhase != PhaseOperating {
			return sg.messages
		}
		sg.executeSkill()

	case protocol.TypeMortgage:
		if sg.turnPhase != PhaseOperating {
			return sg.messages
		}
		var c protocol.C2SMortgage
		_ = json.Unmarshal(payload, &c)
		sg.executeMortgage(c.SpaceID)

	case protocol.TypeRedeem:
		if sg.turnPhase != PhaseOperating {
			return sg.messages
		}
		var c protocol.C2SRedeem
		_ = json.Unmarshal(payload, &c)
		sg.executeRedeem(c.SpaceID)

	case protocol.TypeTrade:
		if sg.turnPhase != PhaseOperating {
			return sg.messages
		}
		var c protocol.C2STrade
		_ = json.Unmarshal(payload, &c)
		sg.executeTrade(c.SpaceID, c.TargetPlayer)

	case protocol.TypeEndTurn:
		if sg.turnPhase != PhaseOperating {
			return sg.messages
		}
		sg.executeEndTurn()
		sg.processAITurns()
	}

	return sg.messages
}

// GetFullState 返回当前游戏的完整状态快照。
func (sg *ServerGame) GetFullState() protocol.S2CGameState {
	st := protocol.S2CGameState{
		ActiveIdx:  sg.activeIdx,
		Phase:      sg.turnPhase,
		DiceValues: sg.dice.Values,
		DiceSum:    sg.dice.Sum,
	}
	st.Players = make([]protocol.GamePlayer, len(sg.players))
	for i := range sg.players {
		st.Players[i] = sg.toGamePlayer(i)
	}
	st.Spaces = make([]protocol.GameSpace, len(sg.board.Spaces))
	for i, s := range sg.board.Spaces {
		st.Spaces[i] = sg.toGameSpace(s)
	}
	return st
}

// FullStateMessage 将完整状态打包为可广播的消息。
func (sg *ServerGame) FullStateMessage() []byte {
	data, _ := protocol.NewMessage(protocol.TypeGameState, sg.GetFullState())
	return data
}

// ===== 动作执行 =====

func (sg *ServerGame) executeRoll() {
	p := sg.players[sg.activeIdx]

	// 出狱倒计时
	if p.InJail > 0 {
		p.InJail--
		if p.InJail > 0 {
			sg.emitEvent(fmt.Sprintf("%s 在监狱中，跳过回合", p.Name))
			sg.emitPlayerUpdate(sg.activeIdx)
			sg.executeEndTurn()
			return
		}
		sg.emitEvent(fmt.Sprintf("%s 出狱了", p.Name))
		sg.emitPlayerUpdate(sg.activeIdx)
	}

	// 掷骰
	v1 := rand.Intn(6) + 1
	v2 := rand.Intn(6) + 1
	sg.dice.Values = [2]int{v1, v2}
	sg.dice.Sum = v1 + v2
	sg.dice.IsDouble = v1 == v2
	sg.dice.Rolled = true
	sg.dice.Rolling = false
	sg.emit(protocol.TypeDiceResult, protocol.S2CDiceResult{
		Values:   sg.dice.Values,
		Sum:      sg.dice.Sum,
		IsDouble: sg.dice.IsDouble,
	})
	sg.emitEvent(fmt.Sprintf("%s 掷出了 %d+%d=%d", p.Name, v1, v2, sg.dice.Sum))

	// 三连双 → 入狱
	if sg.dice.IsDouble {
		p.BonusCount++
		if p.BonusCount >= 3 {
			sg.sendToJail(p)
			sg.emitPlayerUpdate(sg.activeIdx)
			sg.executeEndTurn()
			return
		}
	}

	// 移动 + 经过起点检测
	sg.turnPhase = PhaseMoving
	sg.emitTurnUpdate()
	spaceCount := sg.board.SpaceCount()
	for i := 0; i < sg.dice.Sum; i++ {
		oldPos := p.Position
		p.Position = (p.Position + 1) % spaceCount
		if p.Position == 0 && oldPos != 0 {
			p.Money += 150
			p.SkillPoints++
			sg.emitEvent(fmt.Sprintf("%s 经过起点，获得$150", p.Name))
		}
	}
	sg.emitPlayerUpdate(sg.activeIdx)

	// 着陆
	sg.turnPhase = PhaseLanded
	sg.emitTurnUpdate()
	sg.handleLanded()
	if sg.state != StatePlaying {
		return
	}
	if p.Bankrupt {
		sg.executeEndTurn()
		return
	}

	sg.turnPhase = PhaseOperating
	sg.emitTurnUpdate()
}

func (sg *ServerGame) handleLanded() {
	p := sg.players[sg.activeIdx]
	s := sg.board.Spaces[p.Position]
	sg.emitEvent(fmt.Sprintf("%s 到达 %s", p.Name, s.Name))

	switch s.Type {
	case model.SpaceEvent:
		sg.handleEventSpace(p, s)
	case model.SpaceLand, model.SpaceUtility, model.SpaceTransport:
		if s.Owner == nil {
			if s.IsBuyable() {
				sg.emitEvent(fmt.Sprintf("%s 无主，价格%d元", s.Name, s.Price))
			}
			// 等待玩家发送 buy_property 或 end_turn
		} else if s.Owner != p {
			charge := p.GetCharge(s, sg.board, sg.dice.Sum)
			if charge > 0 {
				p.Money -= charge
				s.Owner.Money += charge
				ownerIdx := sg.playerIdx(s.Owner)
				sg.emitEvent(fmt.Sprintf("%s 向 %s 支付租金$%d", p.Name, s.Owner.Name, charge))
				sg.emitPlayerUpdate(sg.activeIdx)
				if ownerIdx >= 0 {
					sg.emitPlayerUpdate(ownerIdx)
				}
				if p.Money < 0 {
					sg.handleBankruptcy(p)
				}
			}
		}
	}
}

func (sg *ServerGame) handleEventSpace(p *model.Player, s *model.Space) {
	switch s.ID {
	case 0:
		// 起点：移动时已处理
	case 4, 11, 20, 31:
		sg.resolveChance(p)
	case 5:
		p.SkillPoints += 2
		sg.emitEvent(fmt.Sprintf("%s 获得2点技能点", p.Name))
		sg.emitPlayerUpdate(sg.activeIdx)
	case 16:
		sg.sendToJail(p)
		sg.emitPlayerUpdate(sg.activeIdx)
	case 21:
		sg.resolveBlessing(p)
		sg.emitPlayerUpdate(sg.activeIdx)
	}
}

func (sg *ServerGame) executeBuy() {
	p := sg.players[sg.activeIdx]
	s := sg.board.Spaces[p.Position]
	if !s.IsBuyable() || p.Money < s.Price {
		return
	}
	p.Money -= s.Price
	p.AddProperty(s)
	sg.emitEvent(fmt.Sprintf("%s 购买了 %s ($%d)", p.Name, s.Name, s.Price))
	sg.emitPlayerUpdate(sg.activeIdx)
	sg.emitSpaceUpdate(s)
}

func (sg *ServerGame) executeBuild(spaceID int) {
	if spaceID < 0 || spaceID >= len(sg.board.Spaces) {
		return
	}
	p := sg.players[sg.activeIdx]
	s := sg.board.Spaces[spaceID]
	if !p.CanBuild(s, sg.board) {
		return
	}
	cost := s.BuildCost()
	if p.Money < cost {
		return
	}
	p.Money -= cost
	if s.Houses < 4 {
		s.Houses++
		sg.emitEvent(fmt.Sprintf("%s 在 %s 建造房屋(Lv%d) $%d", p.Name, s.Name, s.Houses, cost))
	} else {
		s.HasHotel = true
		s.Houses = 0
		sg.emitEvent(fmt.Sprintf("%s 在 %s 建造酒店! $%d", p.Name, s.Name, cost))
	}
	sg.emitPlayerUpdate(sg.activeIdx)
	sg.emitSpaceUpdate(s)
}

func (sg *ServerGame) executeMortgage(spaceID int) {
	if spaceID < 0 || spaceID >= len(sg.board.Spaces) {
		return
	}
	p := sg.players[sg.activeIdx]
	s := sg.board.Spaces[spaceID]
	if s.Owner != p || s.Mortgaged {
		return
	}
	refund := 0
	if s.Houses > 0 {
		refund = s.Houses * s.BuildCost() / 2
		s.Houses = 0
	} else if s.HasHotel {
		refund = s.BuildCost() * 5 / 2
		s.HasHotel = false
	} else {
		refund = s.Price / 2
		s.Mortgaged = true
	}
	p.Money += refund
	sg.emitEvent(fmt.Sprintf("%s 抵押 %s 获得$%d", p.Name, s.Name, refund))
	sg.emitPlayerUpdate(sg.activeIdx)
	sg.emitSpaceUpdate(s)
}

func (sg *ServerGame) executeRedeem(spaceID int) {
	if spaceID < 0 || spaceID >= len(sg.board.Spaces) {
		return
	}
	p := sg.players[sg.activeIdx]
	s := sg.board.Spaces[spaceID]
	if s.Owner != p || !s.Mortgaged {
		return
	}
	cost := s.Price * 3 / 5
	if p.Money < cost {
		return
	}
	p.Money -= cost
	s.Mortgaged = false
	sg.emitEvent(fmt.Sprintf("%s 赎回 %s $%d", p.Name, s.Name, cost))
	sg.emitPlayerUpdate(sg.activeIdx)
	sg.emitSpaceUpdate(s)
}

func (sg *ServerGame) executeTrade(spaceID int, targetName string) {
	targetIdx, ok := sg.nameIndex[targetName]
	if !ok || targetIdx == sg.activeIdx {
		return
	}
	if spaceID < 0 || spaceID >= len(sg.board.Spaces) {
		return
	}
	p := sg.players[sg.activeIdx]
	target := sg.players[targetIdx]
	if target == nil || target.Bankrupt {
		return
	}
	s := sg.board.Spaces[spaceID]
	if s.Owner != p || s.Houses > 0 || s.HasHotel || s.Mortgaged {
		return
	}
	// 已构成完整色组的地块不可拍卖（与本地版一致）
	if s.Type == model.SpaceLand && sg.board.OwnsFullColorSet(p, s.Color) {
		return
	}
	price := s.Price
	if target.Money < price {
		sg.emitEvent(fmt.Sprintf("%s 资金不足，交易失败", target.Name))
		return
	}
	target.Money -= price
	p.Money += price
	p.RemoveProperty(s)
	target.AddProperty(s)
	sg.emitEvent(fmt.Sprintf("%s 将 %s 卖给 %s ($%d)", p.Name, s.Name, target.Name, price))
	sg.emitPlayerUpdate(sg.activeIdx)
	sg.emitPlayerUpdate(targetIdx)
	sg.emitSpaceUpdate(s)
}

func (sg *ServerGame) executeSkill() {
	p := sg.players[sg.activeIdx]
	char := sg.charNames[sg.activeIdx]
	skill, ok := serverCharSkills[char]
	if !ok || p.SkillPoints < skill.Cost {
		return
	}
	p.SkillPoints -= skill.Cost
	sg.emitEvent(fmt.Sprintf("%s 使用技能: %s", p.Name, skill.Name))

	switch char {
	case "食蜂":
		sg.skillSteal(p, skill.Cost)
	case "黑子":
		sg.skillTeleport(p)
	case "泪子":
		sg.skillFavorableChance(p)
	case "警策":
		sg.skillExtraTurn(p, skill.Cost)
	}
	sg.emitPlayerUpdate(sg.activeIdx)
}

func (sg *ServerGame) executeEndTurn() {
	p := sg.players[sg.activeIdx]

	// 单人剩余 → 游戏结束
	if sg.alivePlayers() <= 1 {
		sg.state = StateGameOver
		var winner *model.Player
		for _, pl := range sg.players {
			if !pl.Bankrupt {
				winner = pl
				break
			}
		}
		winnerName := ""
		if winner != nil {
			winnerName = winner.Name
		}
		sg.emit(protocol.TypeGameOver, protocol.S2CGameOver{WinnerName: winnerName})
		return
	}

	// 双数 + 未三连 + 未入狱 → 当前玩家继续
	if sg.dice.IsDouble && p.BonusCount < 3 && p.InJail == 0 && !p.Bankrupt {
		p.HasOperated = false
		sg.dice.Reset()
		sg.dice.IsDouble = false
		sg.turnPhase = PhaseWaitRoll
		sg.emitEvent(fmt.Sprintf("%s 掷出双数，再来一次!", p.Name))
		sg.emitTurnUpdate()
		return
	}

	p.BonusCount = 0
	sg.dice.Reset()
	sg.dice.IsDouble = false
	sg.nextPlayer()
	sg.turnPhase = PhaseWaitRoll
	sg.emitEvent(fmt.Sprintf("轮到 %s", sg.players[sg.activeIdx].Name))
	sg.emitTurnUpdate()
}

// ===== 机会/祝福事件 =====

func (sg *ServerGame) resolveChance(p *model.Player) {
	weights := []int{5, 5, 3, 3, 2, 2, 2, 1, 3, 2, 3, 3, 2, 2}
	total := 0
	for _, w := range weights {
		total += w
	}
	roll := rand.Intn(total)
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
	case 1:
		amount := (rand.Intn(4) + 1) * 50
		p.Money += amount
		sg.emitEvent(fmt.Sprintf("机会: %s 获得$%d", p.Name, amount))
		sg.emitPlayerUpdate(sg.activeIdx)
	case 2:
		amount := (rand.Intn(4) + 1) * 50
		p.Money -= amount
		sg.emitEvent(fmt.Sprintf("机会: %s 损失$%d", p.Name, amount))
		sg.emitPlayerUpdate(sg.activeIdx)
		if p.Money < 0 {
			sg.handleBankruptcy(p)
		}
	case 3:
		assets := p.TotalAssets() - p.Money
		refund := assets / 10
		p.Money += refund
		sg.emitEvent(fmt.Sprintf("机会: 退税! %s 获得$%d", p.Name, refund))
		sg.emitPlayerUpdate(sg.activeIdx)
	case 4:
		assets := p.TotalAssets() - p.Money
		tax := assets / 10
		p.Money -= tax
		sg.emitEvent(fmt.Sprintf("机会: 财产税! %s 缴纳$%d", p.Name, tax))
		sg.emitPlayerUpdate(sg.activeIdx)
		if p.Money < 0 {
			sg.handleBankruptcy(p)
		}
	case 5:
		sg.chanceFreeBuild(p)
	case 6:
		sg.chanceDemolish(p)
	case 7:
		sg.chanceGlobalBuild()
	case 8:
		sg.chanceGlobalDemolish()
	case 9:
		sg.chanceFreeLand(p)
	case 10:
		sg.chanceLoseLand(p)
	case 11:
		amount := (rand.Intn(3) + 1) * 60
		p.Money -= amount
		alive := sg.alivePlayers()
		share := 0
		if alive > 1 {
			share = amount / (alive - 1)
		}
		for i, other := range sg.players {
			if other != p && !other.Bankrupt {
				other.Money += share
				sg.emitPlayerUpdate(i)
			}
		}
		sg.emitEvent(fmt.Sprintf("机会: %s 慈善捐款$%d", p.Name, amount))
		sg.emitPlayerUpdate(sg.activeIdx)
		if p.Money < 0 {
			sg.handleBankruptcy(p)
		}
	case 12:
		p.JailPassports++
		sg.emitEvent(fmt.Sprintf("机会: %s 获得出狱卡!", p.Name))
		sg.emitPlayerUpdate(sg.activeIdx)
	case 13:
		sg.sendToJail(p)
		sg.emitEvent(fmt.Sprintf("机会: %s 被送入监狱!", p.Name))
		sg.emitPlayerUpdate(sg.activeIdx)
	case 14:
		newPos := rand.Intn(sg.board.SpaceCount())
		p.Position = newPos
		sg.emitEvent(fmt.Sprintf("机会: %s 被传送到 %s", p.Name, sg.board.Spaces[newPos].Name))
		sg.emitPlayerUpdate(sg.activeIdx)
	}
}

func (sg *ServerGame) chanceFreeBuild(p *model.Player) {
	for _, s := range p.OwnedLands {
		if p.CanBuild(s, sg.board) && !s.HasHotel {
			if s.Houses < 4 {
				s.Houses++
			} else {
				s.HasHotel = true
				s.Houses = 0
			}
			sg.emitEvent(fmt.Sprintf("机会: %s 免费建造于 %s", p.Name, s.Name))
			sg.emitSpaceUpdate(s)
			return
		}
	}
	sg.emitEvent(fmt.Sprintf("机会: %s 没有可建造的地产", p.Name))
}

func (sg *ServerGame) chanceDemolish(p *model.Player) {
	for _, s := range p.OwnedLands {
		if s.HasHotel {
			s.HasHotel = false
			s.Houses = 4
			sg.emitEvent(fmt.Sprintf("机会: %s 的 %s 酒店被拆除", p.Name, s.Name))
			sg.emitSpaceUpdate(s)
			return
		}
		if s.Houses > 0 {
			s.Houses--
			sg.emitEvent(fmt.Sprintf("机会: %s 的 %s 被拆除一栋房屋", p.Name, s.Name))
			sg.emitSpaceUpdate(s)
			return
		}
	}
	sg.emitEvent(fmt.Sprintf("机会: %s 没有可拆除的建筑", p.Name))
}

func (sg *ServerGame) chanceGlobalBuild() {
	for _, p := range sg.players {
		if p.Bankrupt {
			continue
		}
		for _, s := range p.OwnedLands {
			if p.CanBuild(s, sg.board) && !s.HasHotel && s.Houses < 4 {
				s.Houses++
				sg.emitEvent(fmt.Sprintf("全员建造: %s 的 %s 新增房屋", p.Name, s.Name))
				sg.emitSpaceUpdate(s)
				break
			}
		}
	}
}

func (sg *ServerGame) chanceGlobalDemolish() {
	for _, p := range sg.players {
		if p.Bankrupt {
			continue
		}
		for _, s := range p.OwnedLands {
			if s.Houses > 0 {
				s.Houses--
				sg.emitEvent(fmt.Sprintf("全员拆迁: %s 的 %s 房屋被拆", p.Name, s.Name))
				sg.emitSpaceUpdate(s)
				break
			}
		}
	}
}

func (sg *ServerGame) chanceFreeLand(p *model.Player) {
	var unowned []*model.Space
	for _, s := range sg.board.Spaces {
		if s.IsBuyable() {
			unowned = append(unowned, s)
		}
	}
	if len(unowned) == 0 {
		sg.emitEvent("机会: 没有空地可分配")
		return
	}
	s := unowned[rand.Intn(len(unowned))]
	p.AddProperty(s)
	sg.emitEvent(fmt.Sprintf("机会: %s 免费获得 %s", p.Name, s.Name))
	sg.emitSpaceUpdate(s)
}

func (sg *ServerGame) chanceLoseLand(p *model.Player) {
	var all []*model.Space
	all = append(all, p.OwnedLands...)
	all = append(all, p.OwnedUtilities...)
	all = append(all, p.OwnedTransports...)
	if len(all) == 0 {
		sg.emitEvent(fmt.Sprintf("机会: %s 没有地产可失去", p.Name))
		return
	}
	s := all[rand.Intn(len(all))]
	p.RemoveProperty(s)
	s.Houses = 0
	s.HasHotel = false
	s.Mortgaged = false
	sg.emitEvent(fmt.Sprintf("机会: %s 失去了 %s", p.Name, s.Name))
	sg.emitSpaceUpdate(s)
}

func (sg *ServerGame) resolveBlessing(p *model.Player) {
	categories := []string{"买地", "过路", "抵押", "加盖"}
	modifiers := []string{"增加", "减少"}
	cat := categories[rand.Intn(len(categories))]
	mod := modifiers[rand.Intn(len(modifiers))]
	p.Blessings = append(p.Blessings, model.Blessing{Category: cat, Modifier: mod})
	effect := "减少50%"
	if mod == "增加" {
		effect = "增加50%"
	}
	sg.emitEvent(fmt.Sprintf("祝福: %s 下次%s费用%s", p.Name, cat, effect))
}

// ===== 技能 =====

func (sg *ServerGame) skillSteal(p *model.Player, cost int) {
	var candidates []*model.Space
	var owners []int
	for i, other := range sg.players {
		if other == p || other.Bankrupt {
			continue
		}
		for _, s := range other.OwnedLands {
			if s.Houses == 0 && !s.HasHotel && !s.Mortgaged {
				candidates = append(candidates, s)
				owners = append(owners, i)
			}
		}
	}
	if len(candidates) == 0 {
		p.SkillPoints += cost
		sg.emitEvent("没有可窃取的地产")
		return
	}
	pick := rand.Intn(len(candidates))
	target := candidates[pick]
	oldOwner := target.Owner
	oldOwnerIdx := owners[pick]
	oldOwner.RemoveProperty(target)
	p.AddProperty(target)
	sg.emitEvent(fmt.Sprintf("从 %s 窃取了 %s!", oldOwner.Name, target.Name))
	sg.emitPlayerUpdate(oldOwnerIdx)
	sg.emitSpaceUpdate(target)
}

func (sg *ServerGame) skillTeleport(p *model.Player) {
	var lands []int
	for i, s := range sg.board.Spaces {
		if s.Type == model.SpaceLand {
			lands = append(lands, i)
		}
	}
	if len(lands) == 0 {
		return
	}
	dest := lands[rand.Intn(len(lands))]
	p.Position = dest
	sg.emitEvent(fmt.Sprintf("传送到 %s!", sg.board.Spaces[dest].Name))
}

func (sg *ServerGame) skillFavorableChance(p *model.Player) {
	favorable := []int{1, 3, 5, 7, 9, 12}
	pick := favorable[rand.Intn(len(favorable))]
	switch pick {
	case 1:
		amount := (rand.Intn(4) + 1) * 50
		p.Money += amount
		sg.emitEvent(fmt.Sprintf("预知: %s 获得$%d", p.Name, amount))
	case 3:
		assets := p.TotalAssets() - p.Money
		refund := assets / 10
		p.Money += refund
		sg.emitEvent(fmt.Sprintf("预知: 退税! 获得$%d", refund))
	case 5:
		sg.chanceFreeBuild(p)
	case 7:
		sg.chanceGlobalBuild()
	case 9:
		sg.chanceFreeLand(p)
	case 12:
		p.JailPassports++
		sg.emitEvent(fmt.Sprintf("预知: %s 获得出狱卡!", p.Name))
	}
}

func (sg *ServerGame) skillExtraTurn(p *model.Player, cost int) {
	if p.BonusCount > 0 {
		p.SkillPoints += cost
		sg.emitEvent("已有额外回合，技能无效")
		return
	}
	sg.dice.IsDouble = true
	sg.emitEvent("获得额外一次行动!")
}

// ===== AI 自动决策 =====

// processAITurns 在当前活动玩家是 AI 时连续推进其回合，直到轮到人类。
func (sg *ServerGame) processAITurns() {
	const safetyLimit = 200
	for steps := 0; steps < safetyLimit; steps++ {
		if sg.state != StatePlaying {
			return
		}
		p := sg.players[sg.activeIdx]
		if !p.IsAI || p.Bankrupt {
			return
		}
		switch sg.turnPhase {
		case PhaseWaitRoll:
			sg.executeRoll()
		case PhaseOperating:
			sg.aiOperate()
		default:
			return
		}
	}
}

func (sg *ServerGame) aiOperate() {
	action, target := sg.aiDecide()
	switch action {
	case "buy":
		sg.executeBuy()
	case "build":
		if target != nil {
			sg.executeBuild(target.ID)
		} else {
			sg.executeEndTurn()
		}
	case "skill":
		sg.executeSkill()
	case "mortgage":
		if target != nil {
			sg.executeMortgage(target.ID)
		} else {
			sg.executeEndTurn()
		}
	default:
		sg.executeEndTurn()
	}
}

// aiDecide 返回 AI 的下一步动作。逻辑与 internal/game/ai.go 等价。
func (sg *ServerGame) aiDecide() (string, *model.Space) {
	p := sg.players[sg.activeIdx]
	if p == nil {
		return "end_turn", nil
	}

	// 1. 当前位置可买
	if p.Position >= 0 && p.Position < len(sg.board.Spaces) {
		cur := sg.board.Spaces[p.Position]
		if cur.IsBuyable() && p.Money >= cur.Price && sg.aiShouldBuyLand(p, cur) {
			return "buy", cur
		}
	}

	// 2. 建造
	if target := sg.aiChooseBuild(p); target != nil {
		return "build", target
	}

	// 3. 技能
	if sg.aiShouldUseSkill(p) {
		return "skill", nil
	}

	// 4. 抵押
	if target := sg.aiChooseMortgage(p); target != nil {
		return "mortgage", target
	}

	return "end_turn", nil
}

func (sg *ServerGame) aiShouldBuyLand(p *model.Player, s *model.Space) bool {
	if s == nil || !s.IsBuyable() || p.Money < s.Price {
		return false
	}
	if s.Type == model.SpaceLand && s.Color != "" && s.Color != "NONE" {
		group := sg.board.ColorSets[s.Color]
		owned, unowned := 0, 0
		for _, sp := range group {
			if sp == s {
				continue
			}
			if sp.Owner == p {
				owned++
			}
			if sp.Owner == nil {
				unowned++
			}
		}
		if unowned == 0 && owned == len(group)-1 {
			return true
		}
		if owned > 0 && p.Money > s.Price*3/2 {
			return true
		}
	}
	if p.Money > s.Price*2 {
		return true
	}
	if p.Money < s.Price*12/10 {
		return false
	}
	return true
}

func (sg *ServerGame) aiChooseBuild(p *model.Player) *model.Space {
	var best *model.Space
	for _, s := range p.OwnedLands {
		if !p.CanBuild(s, sg.board) {
			continue
		}
		cost := s.BuildCost()
		if p.Money <= cost*3 {
			continue
		}
		if best == nil || s.Houses < best.Houses {
			best = s
		}
	}
	return best
}

func (sg *ServerGame) aiShouldUseSkill(p *model.Player) bool {
	char := sg.charNames[sg.activeIdx]
	skill, ok := serverCharSkills[char]
	if !ok || p.SkillPoints < skill.Cost {
		return false
	}
	switch char {
	case "食蜂":
		for _, other := range sg.players {
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

func (sg *ServerGame) aiChooseMortgage(p *model.Player) *model.Space {
	if p.Money >= 100 {
		return nil
	}
	for _, s := range p.OwnedLands {
		if s.Mortgaged || s.Houses > 0 || s.HasHotel {
			continue
		}
		if sg.board.OwnsFullColorSet(p, s.Color) {
			continue
		}
		return s
	}
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

// ===== 辅助 =====

func (sg *ServerGame) sendToJail(p *model.Player) {
	if p.JailPassports > 0 {
		p.JailPassports--
		sg.emitEvent(fmt.Sprintf("%s 使用出狱卡!", p.Name))
		return
	}
	p.Position = 16
	p.InJail = 2
	p.BonusCount = 0
	sg.emitEvent(fmt.Sprintf("%s 入狱!", p.Name))
}

func (sg *ServerGame) handleBankruptcy(p *model.Player) {
	if p.TotalAssets() >= 0 {
		return
	}
	p.Bankrupt = true
	idx := sg.playerIdx(p)
	var changed []*model.Space
	for _, s := range p.OwnedLands {
		s.Owner = nil
		s.Houses = 0
		s.HasHotel = false
		s.Mortgaged = false
		changed = append(changed, s)
	}
	for _, s := range p.OwnedUtilities {
		s.Owner = nil
		s.Mortgaged = false
		changed = append(changed, s)
	}
	for _, s := range p.OwnedTransports {
		s.Owner = nil
		s.Mortgaged = false
		changed = append(changed, s)
	}
	p.OwnedLands = nil
	p.OwnedUtilities = nil
	p.OwnedTransports = nil
	sg.emitEvent(fmt.Sprintf("%s 破产了!", p.Name))
	if idx >= 0 {
		sg.emitPlayerUpdate(idx)
	}
	for _, s := range changed {
		sg.emitSpaceUpdate(s)
	}
}

func (sg *ServerGame) nextPlayer() {
	for i := 0; i < len(sg.players); i++ {
		sg.activeIdx = (sg.activeIdx + 1) % len(sg.players)
		if !sg.players[sg.activeIdx].Bankrupt {
			break
		}
	}
	sg.players[sg.activeIdx].HasOperated = false
}

func (sg *ServerGame) alivePlayers() int {
	count := 0
	for _, p := range sg.players {
		if !p.Bankrupt {
			count++
		}
	}
	return count
}

func (sg *ServerGame) playerIdx(p *model.Player) int {
	for i, pl := range sg.players {
		if pl == p {
			return i
		}
	}
	return -1
}

// ===== 协议映射 =====

func (sg *ServerGame) toGamePlayer(idx int) protocol.GamePlayer {
	p := sg.players[idx]
	gp := protocol.GamePlayer{
		Name:          p.Name,
		Money:         p.Money,
		Position:      p.Position,
		Direction:     p.Direction,
		InJail:        p.InJail,
		JailPassports: p.JailPassports,
		SkillPoints:   p.SkillPoints,
		BonusCount:    p.BonusCount,
		Bankrupt:      p.Bankrupt,
		IsAI:          p.IsAI,
		CharName:      sg.charNames[idx],
	}
	for _, b := range p.Blessings {
		gp.Blessings = append(gp.Blessings, protocol.GameBlessing{
			Category: b.Category,
			Modifier: b.Modifier,
		})
	}
	return gp
}

func (sg *ServerGame) toGameSpace(s *model.Space) protocol.GameSpace {
	gs := protocol.GameSpace{
		ID:        s.ID,
		Houses:    s.Houses,
		HasHotel:  s.HasHotel,
		Mortgaged: s.Mortgaged,
		Price:     s.Price,
	}
	if s.Owner != nil {
		gs.OwnerName = s.Owner.Name
	}
	return gs
}

// ===== 消息广播 =====

func (sg *ServerGame) emit(msgType string, payload interface{}) {
	data, err := protocol.NewMessage(msgType, payload)
	if err != nil {
		return
	}
	sg.messages = append(sg.messages, data)
}

func (sg *ServerGame) emitEvent(text string) {
	sg.emit(protocol.TypeEventMsg, protocol.S2CEventMsg{Text: text})
}

func (sg *ServerGame) emitTurnUpdate() {
	sg.emit(protocol.TypeTurnUpdate, protocol.S2CTurnUpdate{
		ActiveIdx: sg.activeIdx,
		Phase:     sg.turnPhase,
	})
}

func (sg *ServerGame) emitPlayerUpdate(idx int) {
	if idx < 0 || idx >= len(sg.players) {
		return
	}
	sg.emit(protocol.TypePlayerUpdate, protocol.S2CPlayerUpdate{
		Idx:    idx,
		Player: sg.toGamePlayer(idx),
	})
}

func (sg *ServerGame) emitSpaceUpdate(s *model.Space) {
	sg.emit(protocol.TypeSpaceUpdate, protocol.S2CSpaceUpdate{
		Space: sg.toGameSpace(s),
	})
}

// RestoreFromState 从持久化的游戏状态快照恢复 ServerGame。
// 用于服务端重启后从存储中恢复未完成的对局。
func RestoreFromState(state protocol.S2CGameState) *ServerGame {
	sg := &ServerGame{
		board:     model.ParseBoard(serverBuildingsConfig),
		dice:      model.NewDice(),
		state:     StatePlaying,
		turnPhase: state.Phase,
		activeIdx: state.ActiveIdx,
		nameIndex: make(map[string]int, len(state.Players)),
	}

	// 恢复骰子状态
	sg.dice.Values = state.DiceValues
	sg.dice.Sum = state.DiceSum
	sg.dice.Rolled = true

	// 恢复玩家
	for i, gp := range state.Players {
		p := model.NewPlayer(gp.Name, nil, gp.Direction)
		p.Money = gp.Money
		p.Position = gp.Position
		p.InJail = gp.InJail
		p.JailPassports = gp.JailPassports
		p.SkillPoints = gp.SkillPoints
		p.BonusCount = gp.BonusCount
		p.Bankrupt = gp.Bankrupt
		p.IsAI = gp.IsAI
		for _, b := range gp.Blessings {
			p.Blessings = append(p.Blessings, model.Blessing{
				Category: b.Category,
				Modifier: b.Modifier,
			})
		}
		sg.players = append(sg.players, p)
		sg.charNames = append(sg.charNames, gp.CharName)
		sg.nameIndex[gp.Name] = i
	}

	// 恢复地块所有权和建筑状态
	for _, gs := range state.Spaces {
		if gs.ID < 0 || gs.ID >= len(sg.board.Spaces) {
			continue
		}
		s := sg.board.Spaces[gs.ID]
		s.Houses = gs.Houses
		s.HasHotel = gs.HasHotel
		s.Mortgaged = gs.Mortgaged
		s.Price = gs.Price
		if gs.OwnerName != "" {
			if idx, ok := sg.nameIndex[gs.OwnerName]; ok {
				sg.players[idx].AddProperty(s)
			}
		}
	}

	return sg
}
