package server

import (
	"log"
	"sync"
	"time"

	"github.com/PlasmolysisMango/Gonopoly/pkg/protocol"
)

const ReconnectTimeout = 5 * time.Minute // 5分钟重连窗口

// disconnectTimerManager 管理所有房间内玩家的断线超时计时器。
type disconnectTimerManager struct {
	timers map[string]*time.Timer // playerName -> timer
	mu     sync.Mutex
}

func newDisconnectTimerManager() *disconnectTimerManager {
	return &disconnectTimerManager{
		timers: make(map[string]*time.Timer),
	}
}

// StartDisconnectTimer 启动断线超时检测。
// 如果超时仍未重连，将玩家标记为永久离开并广播通知。
func (r *Room) StartDisconnectTimer(playerName string) {
	r.dtm.mu.Lock()
	defer r.dtm.mu.Unlock()

	// 取消已有计时器（如果存在）
	if t, ok := r.dtm.timers[playerName]; ok {
		t.Stop()
	}

	r.dtm.timers[playerName] = time.AfterFunc(ReconnectTimeout, func() {
		r.handleDisconnectTimeout(playerName)
	})

	log.Printf("room %s: started disconnect timer for %s (%v)", r.ID, playerName, ReconnectTimeout)
}

// CancelDisconnectTimer 取消断线计时器（重连成功时调用）。
func (r *Room) CancelDisconnectTimer(playerName string) {
	r.dtm.mu.Lock()
	defer r.dtm.mu.Unlock()

	if t, ok := r.dtm.timers[playerName]; ok {
		t.Stop()
		delete(r.dtm.timers, playerName)
		log.Printf("room %s: cancelled disconnect timer for %s", r.ID, playerName)
	}
}

// handleDisconnectTimeout 断线超时后的处理：标记玩家永久离开。
func (r *Room) handleDisconnectTimeout(playerName string) {
	r.dtm.mu.Lock()
	delete(r.dtm.timers, playerName)
	r.dtm.mu.Unlock()

	log.Printf("room %s: player %s disconnect timeout, marking as permanently left", r.ID, playerName)

	// 广播玩家永久离开（Reconnectable=false）
	left, _ := protocol.NewMessage(protocol.TypePlayerLeft, protocol.S2CPlayerLeft{
		Name:          playerName,
		Reconnectable: false,
	})
	r.Broadcast(left)

	// 如果当前轮到该玩家，自动结束回合
	r.HandleDisconnectedPlayerTurn(playerName)
}

// HandleDisconnectedPlayerTurn 处理断线玩家的回合：自动执行 end_turn。
// 如果当前活动玩家就是断线的玩家，自动结束其回合。
func (r *Room) HandleDisconnectedPlayerTurn(playerName string) {
	r.mu.RLock()
	game := r.Game
	r.mu.RUnlock()
	if game == nil {
		return
	}

	// 检查是否轮到该玩家
	idx, ok := game.nameIndex[playerName]
	if !ok {
		return
	}
	if game.ActiveIdx() != idx {
		return
	}

	// 根据当前阶段决定如何处理
	switch game.Phase() {
	case PhaseWaitRoll:
		// 自动掷骰+结束回合
		msgs := game.HandleAction(playerName, protocol.TypeRollDice, nil)
		for _, m := range msgs {
			r.Broadcast(m)
		}
		// 掷骰后如果还在操作阶段，自动结束
		if game.Phase() == PhaseOperating && game.ActiveIdx() == idx {
			msgs = game.HandleAction(playerName, protocol.TypeEndTurn, nil)
			for _, m := range msgs {
				r.Broadcast(m)
			}
		}
	case PhaseOperating:
		// 直接结束回合
		msgs := game.HandleAction(playerName, protocol.TypeEndTurn, nil)
		for _, m := range msgs {
			r.Broadcast(m)
		}
	}
}

// IsPlayerOnline 检查指定玩家是否在线。
func (r *Room) IsPlayerOnline(playerName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.Players {
		if p.Name == playerName {
			return p.Online
		}
	}
	return false
}

// CheckActivePlayerOnline 检查当前活动玩家是否在线，若不在线则自动处理其回合。
// 在每次游戏操作广播后调用，确保断线玩家的回合被自动处理。
func (r *Room) CheckActivePlayerOnline() {
	r.mu.RLock()
	game := r.Game
	r.mu.RUnlock()
	if game == nil {
		return
	}

	activeIdx := game.ActiveIdx()
	if activeIdx < 0 || activeIdx >= len(game.players) {
		return
	}
	activeName := game.players[activeIdx].Name

	if !r.IsPlayerOnline(activeName) {
		// 给断线玩家一个短暂延迟，避免连续快速处理
		time.AfterFunc(500*time.Millisecond, func() {
			// 再次验证该玩家仍然离线且仍是活动玩家
			if !r.IsPlayerOnline(activeName) && game.ActiveIdx() == activeIdx {
				r.HandleDisconnectedPlayerTurn(activeName)
			}
		})
	}
}
