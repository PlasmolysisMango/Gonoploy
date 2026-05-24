// Package game - online.go 联机模式下的客户端逻辑。
//
// 联机模式与本地模式的核心差异：
//  1. 状态权威在服务端，客户端按收到的 S2C 消息更新本地缓存。
//  2. 仅在自己回合时才允许向服务端发送操作消息（C2S）。
//  3. 移动/掷骰等动画仍在本地播放（由 dice/render 层驱动）。
//
// 本文件提供的 apply* 系列函数只完成基本的状态映射，
// 复杂细节（如完整地块参数、祝福计算等）将在后续任务中扩展。
package game

import (
	"fmt"
	"time"

	asset "github.com/PlasmolysisMango/Gonopoly/asset/pics"
	"github.com/PlasmolysisMango/Gonopoly/internal/model"
	"github.com/PlasmolysisMango/Gonopoly/internal/network"
	"github.com/PlasmolysisMango/Gonopoly/pkg/protocol"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// updatePlayingOnline 在联机模式下逐帧执行：
//   - 检查连接状态（断线/重连中时显示提示）
//   - 轮询并处理服务器消息（应用状态）
//   - 推进本地动画（骰子/移动）
//   - 处理聊天输入
//   - 仅在自己回合时接受 UI 输入并发送操作
func (g *Game) updatePlayingOnline() {
	// 0. 检查连接状态
	if g.netClient != nil {
		switch g.netClient.State() {
		case network.StateReconnecting:
			// 断线重连中：显示提示，不处理输入
			g.uiMgr.Update()
			g.lobbyMessage = "连接中断，正在重连..."
			return
		case network.StateDisconnected:
			// 重连失败：显示提示，允许返回主菜单
			g.uiMgr.Update()
			g.lobbyMessage = "连接已断开，重连失败"
			if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
				g.netClient.Close()
				g.netClient = nil
				g.state = StateMainMenu
			}
			return
		}
	}

	// 1. 轮询并处理服务器推送
	if g.netClient != nil {
		for {
			msg, ok := g.netClient.Poll()
			if !ok {
				break
			}
			g.handleServerMessage(*msg)
		}
	}

	// 2. 推进 UI/动画
	g.uiMgr.Update()
	g.updatePlayerInfoUI()
	g.updateOnlineAnimations()

	// 3. 处理聊天输入（聊天激活时拦截其他按键）
	if g.handleChatInput() {
		return
	}

	// 4. 自己回合允许发送操作
	if g.isMyTurn() {
		g.handleOnlineLocalInput()
	}
}

// updateOnlineAnimations 在联机模式下推进本地视觉动画。
// 服务端只通知最终结果（骰子值/移动到位），客户端用动画过渡呈现。
func (g *Game) updateOnlineAnimations() {
	if g.dice != nil && g.dice.Rolling {
		g.dice.Tick()
	}
}

// handleOnlineLocalInput 在自己回合接收按键/按钮并转换为 C2S 消息。
// 当前实现保持简洁：仅把最关键的几个操作映射出去，详细 UI 由 Task 14 完成。
func (g *Game) handleOnlineLocalInput() {
	// 掷骰 / 结束回合（沿用本地 UI 按钮）
	if g.uiMgr.DiceButtonClicked() {
		g.sendAction(protocol.TypeRollDice, &protocol.C2SRollDice{})
		return
	}
	if g.uiMgr.EndTurnClicked() {
		g.sendAction(protocol.TypeEndTurn, &protocol.C2SEndTurn{})
		return
	}

	// 快捷键：B 购买当前地块
	if inpututil.IsKeyJustPressed(ebiten.KeyB) {
		g.sendAction(protocol.TypeBuyProperty, &protocol.C2SBuyProperty{})
		return
	}
	// 快捷键：S 使用技能
	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		g.sendAction(protocol.TypeUseSkill, &protocol.C2SUseSkill{})
		return
	}
}

// isMyTurn 判断是否轮到本机玩家操作。
// 仅当处于联机模式且 activeIdx == localIdx 时返回 true。
func (g *Game) isMyTurn() bool {
	return g.mode == ModeOnline && g.activeIdx == g.localIdx
}

// sendAction 向服务端发送一条操作消息，错误仅记录到消息框。
func (g *Game) sendAction(msgType string, payload interface{}) {
	if g.netClient == nil {
		return
	}
	if err := g.netClient.Send(msgType, payload); err != nil {
		g.uiMgr.AddMessage("发送失败: " + err.Error())
	}
}

// handleServerMessage 根据消息类型路由到具体的 apply* 函数。
func (g *Game) handleServerMessage(msg protocol.Message) {
	switch msg.Type {
	case protocol.TypeRoomInfo:
		g.applyRoomInfo(msg)
	case protocol.TypeGameState:
		g.applyFullState(msg)
	case protocol.TypeTurnUpdate:
		g.applyTurnUpdate(msg)
	case protocol.TypeDiceResult:
		g.applyDiceResult(msg)
	case protocol.TypePlayerUpdate:
		g.applyPlayerUpdate(msg)
	case protocol.TypeSpaceUpdate:
		g.applySpaceUpdate(msg)
	case protocol.TypeEventMsg:
		g.applyEventMsg(msg)
	case protocol.TypeChatMsg:
		g.applyChatMsg(msg)
	case protocol.TypePlayerJoined:
		g.applyPlayerJoined(msg)
	case protocol.TypePlayerLeft:
		g.applyPlayerLeft(msg)
	case protocol.TypeGameOver:
		g.applyGameOver(msg)
	case protocol.TypeReconnectOK:
		g.applyReconnectOK(msg)
	case protocol.TypeSpectateOK:
		g.applySpectateOK(msg)
	case protocol.TypeError:
		g.applyError(msg)
	}
}

// ----- apply* 系列：从服务端消息更新本地状态 -----

// applyRoomInfo 处理房间信息：在等待页展示玩家列表，并在游戏开始时切到 Playing。
func (g *Game) applyRoomInfo(msg protocol.Message) {
	info, err := protocol.ParsePayload[protocol.S2CRoomInfo](msg)
	if err != nil {
		return
	}
	g.roomID = info.RoomID
	g.waitingPlayers = nil
	for _, rp := range info.Players {
		status := "未就绪"
		if rp.Ready {
			status = "已就绪"
		}
		if !rp.Online {
			status = "离线"
		}
		g.waitingPlayers = append(g.waitingPlayers,
			fmt.Sprintf("%s [%s] (%s)", rp.Name, rp.CharName, status))
	}
	if info.Started {
		// 服务器宣布游戏开始（实际玩家/地块状态会随后由 game_state 同步）
		if g.state == StateWaiting {
			g.lobbyMessage = "游戏开始!"
		}
	} else {
		g.lobbyMessage = "等待其他玩家..."
	}
}

// applyFullState 全量同步玩家与地块。加入/重连时由服务端主动下发。
func (g *Game) applyFullState(msg protocol.Message) {
	st, err := protocol.ParsePayload[protocol.S2CGameState](msg)
	if err != nil {
		return
	}

	// 1. 重建玩家列表（保留对头像的解析）
	g.players = g.players[:0]
	for _, gp := range st.Players {
		icon := g.assets.Icons[gp.CharName]
		if icon == nil {
			// 兜底：用名字尝试匹配
			icon = g.assets.Icons[gp.Name]
		}
		p := model.NewPlayer(gp.Name, icon, 0)
		applyGamePlayer(p, gp)
		g.players = append(g.players, p)
	}

	// 2. 找到本机玩家下标
	g.localIdx = 0
	for i, p := range g.players {
		if p.Name == g.myName {
			g.localIdx = i
			break
		}
	}

	// 3. 同步地块（仅价格/所有权/建筑/抵押四个字段，名称等仍由本地配置维持）
	for _, gs := range st.Spaces {
		if gs.ID < 0 || gs.ID >= len(g.board.Spaces) {
			continue
		}
		applyGameSpace(g.board.Spaces[gs.ID], gs, g.players)
	}

	// 4. 回合 / 骰子
	g.activeIdx = st.ActiveIdx
	g.turnPhase = TurnPhase(st.Phase)
	if g.dice != nil {
		g.dice.Values = st.DiceValues
		g.dice.Sum = st.DiceSum
	}

	// 5. 切换状态：从等待页过渡到对局界面
	if g.state == StateWaiting {
		if g.isSpectator {
			g.state = StateSpectating
		} else {
			g.state = StatePlaying
			g.uiMgr.SetDiceMode(true)
		}
	}
	g.updatePlayerInfoUI()
}

// applyTurnUpdate 增量更新当前活跃玩家与回合阶段。
func (g *Game) applyTurnUpdate(msg protocol.Message) {
	upd, err := protocol.ParsePayload[protocol.S2CTurnUpdate](msg)
	if err != nil {
		return
	}
	g.activeIdx = upd.ActiveIdx
	g.turnPhase = TurnPhase(upd.Phase)

	if !g.isSpectator {
		if g.turnPhase == TurnWaitRoll {
			g.uiMgr.SetDiceMode(true)
		} else if g.turnPhase == TurnEndTurn || g.turnPhase == TurnOperating {
			g.uiMgr.SetDiceMode(false)
		}
	} else {
		g.uiMgr.SetDiceMode(false)
	}

	if p := g.ActivePlayer(); p != nil {
		g.uiMgr.AddMessage("轮到 " + p.Name)
	}
	g.updatePlayerInfoUI()
}

// applyDiceResult 触发本地骰子动画并显示结果。
func (g *Game) applyDiceResult(msg protocol.Message) {
	res, err := protocol.ParsePayload[protocol.S2CDiceResult](msg)
	if err != nil || g.dice == nil {
		return
	}
	g.dice.Values = res.Values
	g.dice.Sum = res.Sum
	g.dice.IsDouble = res.IsDouble
	g.dice.Rolled = true
	g.dice.Rolling = false

	if p := g.ActivePlayer(); p != nil {
		g.uiMgr.AddMessage(fmt.Sprintf("%s 掷出了 %d+%d=%d",
			p.Name, res.Values[0], res.Values[1], res.Sum))
	}
}

// applyPlayerUpdate 应用单个玩家的增量变化。
func (g *Game) applyPlayerUpdate(msg protocol.Message) {
	upd, err := protocol.ParsePayload[protocol.S2CPlayerUpdate](msg)
	if err != nil {
		return
	}
	if upd.Idx < 0 || upd.Idx >= len(g.players) {
		return
	}
	applyGamePlayer(g.players[upd.Idx], upd.Player)
	g.updatePlayerInfoUI()
}

// applySpaceUpdate 应用单个地块的增量变化（所有权/建筑/抵押）。
func (g *Game) applySpaceUpdate(msg protocol.Message) {
	upd, err := protocol.ParsePayload[protocol.S2CSpaceUpdate](msg)
	if err != nil {
		return
	}
	if upd.Space.ID < 0 || upd.Space.ID >= len(g.board.Spaces) {
		return
	}
	applyGameSpace(g.board.Spaces[upd.Space.ID], upd.Space, g.players)
}

// applyEventMsg 把事件文字转发到 UI 消息框。
func (g *Game) applyEventMsg(msg protocol.Message) {
	ev, err := protocol.ParsePayload[protocol.S2CEventMsg](msg)
	if err != nil {
		return
	}
	g.uiMgr.AddMessage(ev.Text)
}

// applyChatMsg 把聊天消息转发到聊天 UI。
func (g *Game) applyChatMsg(msg protocol.Message) {
	chat, err := protocol.ParsePayload[protocol.S2CChatMsg](msg)
	if err != nil {
		return
	}
	timeStr := time.Unix(chat.Timestamp, 0).Format("15:04")
	g.uiMgr.AddChatMessage(chat.From, chat.Content, timeStr)
}

// applyPlayerJoined 在等待页面更新玩家列表，对局中作为提示信息。
func (g *Game) applyPlayerJoined(msg protocol.Message) {
	pj, err := protocol.ParsePayload[protocol.S2CPlayerJoined](msg)
	if err != nil {
		return
	}
	g.uiMgr.AddMessage(pj.Name + " 加入了房间")
}

// applyPlayerLeft 提示玩家离开。
func (g *Game) applyPlayerLeft(msg protocol.Message) {
	pl, err := protocol.ParsePayload[protocol.S2CPlayerLeft](msg)
	if err != nil {
		return
	}
	suffix := ""
	if pl.Reconnectable {
		suffix = "（可重连）"
	}
	g.uiMgr.AddMessage(pl.Name + " 离开了房间" + suffix)
}

// applyGameOver 切到游戏结束界面。
func (g *Game) applyGameOver(msg protocol.Message) {
	over, err := protocol.ParsePayload[protocol.S2CGameOver](msg)
	if err == nil && over.WinnerName != "" {
		g.uiMgr.AddMessage(over.WinnerName + " 获胜!")
	}
	g.state = StateGameOver
}

// applySpectateOK 接收服务端观战加入成功确认，切换到观战状态。
func (g *Game) applySpectateOK(msg protocol.Message) {
	ok, err := protocol.ParsePayload[protocol.S2CSpectateOK](msg)
	if err != nil {
		return
	}
	g.roomID = ok.RoomID
	g.isSpectator = true
	g.state = StateSpectating
	g.lobbyMessage = "观战中"
	g.uiMgr.AddMessage("已以观战身份加入房间")
}

// applyReconnectOK 保存令牌并提示重连成功。
func (g *Game) applyReconnectOK(msg protocol.Message) {
	rec, err := protocol.ParsePayload[protocol.S2CReconnectOK](msg)
	if err != nil {
		return
	}
	if g.netClient != nil {
		g.netClient.SetReconnectInfo(g.roomID, g.myName, rec.Token)
	}
	g.uiMgr.AddMessage("重连成功")
}

// applyError 在 UI 上提示错误。
func (g *Game) applyError(msg protocol.Message) {
	ev, err := protocol.ParsePayload[protocol.S2CError](msg)
	if err != nil {
		return
	}
	text := "错误: " + ev.Message
	g.uiMgr.AddMessage(text)
	if g.state == StateWaiting || g.state == StateLobby {
		g.lobbyMessage = text
	}
}

// ===== 字段映射工具 =====

// applyGamePlayer 将协议结构 GamePlayer 同步到 model.Player。
// 仅同步标量字段；地产关系由 applyGameSpace 修复。
func applyGamePlayer(p *model.Player, gp protocol.GamePlayer) {
	p.Name = gp.Name
	p.Money = gp.Money
	p.Position = gp.Position
	p.Direction = gp.Direction
	p.InJail = gp.InJail
	p.JailPassports = gp.JailPassports
	p.SkillPoints = gp.SkillPoints
	p.BonusCount = gp.BonusCount
	p.Bankrupt = gp.Bankrupt
	p.IsAI = gp.IsAI

	p.Blessings = p.Blessings[:0]
	for _, b := range gp.Blessings {
		p.Blessings = append(p.Blessings, model.Blessing{
			Category: b.Category,
			Modifier: b.Modifier,
		})
	}
}

// applyGameSpace 将协议地块同步到本地 Space，并按 OwnerName 重建所有权链接。
func applyGameSpace(s *model.Space, gs protocol.GameSpace, players []*model.Player) {
	s.Houses = gs.Houses
	s.HasHotel = gs.HasHotel
	s.Mortgaged = gs.Mortgaged
	if gs.Price > 0 {
		s.Price = gs.Price
	}

	// 先清掉旧的所有权关系
	if s.Owner != nil {
		removeFromOwner(s.Owner, s)
		s.Owner = nil
	}
	if gs.OwnerName == "" {
		return
	}
	for _, p := range players {
		if p.Name == gs.OwnerName {
			p.AddProperty(s)
			return
		}
	}
}

// removeFromOwner 从原所有者的对应分类列表中移除地块（与 Player.RemoveProperty 等价但不清空 Owner，
// 因为调用方紧接着会重新设置）。
func removeFromOwner(p *model.Player, s *model.Space) {
	switch s.Type {
	case model.SpaceLand:
		p.OwnedLands = removeSpaceFromList(p.OwnedLands, s)
	case model.SpaceUtility:
		p.OwnedUtilities = removeSpaceFromList(p.OwnedUtilities, s)
	case model.SpaceTransport:
		p.OwnedTransports = removeSpaceFromList(p.OwnedTransports, s)
	}
}

func removeSpaceFromList(list []*model.Space, s *model.Space) []*model.Space {
	for i, sp := range list {
		if sp == s {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}

// 占位，避免未来 import 被优化掉时的编辑负担。
var _ = asset.CharacterNames

// handleChatInput 处理聊天输入。返回 true 表示聊天处于激活状态，应拦截其他游戏输入。
func (g *Game) handleChatInput() bool {
	if g.uiMgr.IsChatActive() {
		// 处理字符输入
		chars := ebiten.AppendInputChars(nil)
		if len(chars) > 0 {
			g.uiMgr.SetChatInput(g.uiMgr.GetChatInput() + string(chars))
		}

		// Enter 发送
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			content := g.uiMgr.GetChatInput()
			if content != "" {
				g.sendAction(protocol.TypeChat, &protocol.C2SChat{Content: content})
				g.uiMgr.ClearChatInput()
			}
			g.uiMgr.SetChatActive(false)
			return true
		}

		// Escape 取消
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.uiMgr.ClearChatInput()
			g.uiMgr.SetChatActive(false)
			return true
		}

		// Backspace 删除
		if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
			input := g.uiMgr.GetChatInput()
			if len(input) > 0 {
				// 处理 UTF-8 字符
				runes := []rune(input)
				g.uiMgr.SetChatInput(string(runes[:len(runes)-1]))
			}
			return true
		}

		return true
	}

	// T 键打开聊天
	if inpututil.IsKeyJustPressed(ebiten.KeyT) {
		g.uiMgr.SetChatActive(true)
		return true
	}

	return false
}

// updateSpectating 观战模式下的主循环：
//   - 轮询并应用所有服务端推送
//   - 不接受任何游戏操作输入
//   - 仅允许聊天与 Esc 退出观战
func (g *Game) updateSpectating() {
	// 连接状态检查
	if g.netClient != nil {
		switch g.netClient.State() {
		case network.StateReconnecting:
			g.uiMgr.Update()
			g.lobbyMessage = "连接中断，正在重连..."
			return
		case network.StateDisconnected:
			g.uiMgr.Update()
			g.lobbyMessage = "连接已断开"
			if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
				g.exitSpectate()
			}
			return
		}
	}

	// 1. 轮询服务端消息
	if g.netClient != nil {
		for {
			msg, ok := g.netClient.Poll()
			if !ok {
				break
			}
			g.handleServerMessage(*msg)
		}
	}

	// 2. 推进 UI/动画
	g.uiMgr.Update()
	g.updatePlayerInfoUI()
	g.updateOnlineAnimations()

	// 3. 聊天输入（观众仍可以发送聊天）
	if g.handleChatInput() {
		return
	}

	// 4. Esc 退出观战，返回主菜单
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.exitSpectate()
	}
}

// exitSpectate 退出观战，断开连接并返回主菜单。
func (g *Game) exitSpectate() {
	if g.netClient != nil {
		// 主动发送 leave 让服务端及时清理观众记录
		_ = g.netClient.Send(protocol.TypeLeaveRoom, &protocol.C2SLeaveRoom{})
		g.netClient.Close()
		g.netClient = nil
	}
	g.isSpectator = false
	g.state = StateMainMenu
}

// drawSpectating 观战模式的绘制：在正常棋盘上覆盖顶部提示条与水印。
func (g *Game) drawSpectating(screen *ebiten.Image) {
	g.drawPlaying(screen)
	drawSpectatorOverlay(screen, g.textHead, g.textHint, g.menuTick)
	g.menuTick++
}
