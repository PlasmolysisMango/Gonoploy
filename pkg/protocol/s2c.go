package protocol

// 服务端到客户端消息类型常量
const (
	TypeRoomInfo     = "room_info"
	TypeGameState    = "game_state"
	TypeTurnUpdate   = "turn_update"
	TypeDiceResult   = "dice_result"
	TypePlayerUpdate = "player_update"
	TypeSpaceUpdate  = "space_update"
	TypeEventMsg     = "event_msg"
	TypeChatMsg      = "chat_msg"
	TypeError        = "error"
	TypePlayerJoined = "player_joined"
	TypePlayerLeft   = "player_left"
	TypeGameOver     = "game_over"
	TypeReconnectOK  = "reconnect_ok"
	TypeSpectateOK   = "spectate_ok"
	TypeRoomList     = "room_list"
)

// RoomPlayer 房间中的玩家信息（房间大厅展示用）
type RoomPlayer struct {
	Name     string `json:"name"`
	CharName string `json:"char_name"`
	Ready    bool   `json:"ready"`
	Online   bool   `json:"online"`
}

// S2CRoomInfo 房间信息广播
type S2CRoomInfo struct {
	RoomID  string       `json:"room_id"`
	Players []RoomPlayer `json:"players"`
	Started bool         `json:"started"`
}

// GameBlessing 玩家祝福/状态条目（对应 model.Blessing）
type GameBlessing struct {
	Category string `json:"category"`
	Modifier string `json:"modifier"`
}

// GamePlayer 完整玩家状态（用于全量同步）
type GamePlayer struct {
	Name          string         `json:"name"`
	Money         int            `json:"money"`
	Position      int            `json:"position"`
	Direction     int            `json:"direction"`
	InJail        int            `json:"in_jail"`
	JailPassports int            `json:"jail_passports"`
	SkillPoints   int            `json:"skill_points"`
	BonusCount    int            `json:"bonus_count"`
	Bankrupt      bool           `json:"bankrupt"`
	IsAI          bool           `json:"is_ai"`
	Blessings     []GameBlessing `json:"blessings,omitempty"`
	CharName      string         `json:"char_name"`
}

// GameSpace 完整地块状态
type GameSpace struct {
	ID        int    `json:"id"`
	OwnerName string `json:"owner_name,omitempty"`
	Houses    int    `json:"houses"`
	HasHotel  bool   `json:"has_hotel,omitempty"`
	Mortgaged bool   `json:"mortgaged,omitempty"`
	Price     int    `json:"price"`
}

// S2CGameState 完整游戏状态（加入/重连时全量同步）
type S2CGameState struct {
	Players    []GamePlayer `json:"players"`
	Spaces     []GameSpace  `json:"spaces"`
	ActiveIdx  int          `json:"active_idx"`
	Phase      int          `json:"phase"`
	DiceValues [2]int       `json:"dice_values"`
	DiceSum    int          `json:"dice_sum"`
}

// S2CTurnUpdate 回合切换通知
type S2CTurnUpdate struct {
	ActiveIdx int `json:"active_idx"`
	Phase     int `json:"phase"`
}

// S2CDiceResult 掷骰结果
type S2CDiceResult struct {
	Values   [2]int `json:"values"`
	Sum      int    `json:"sum"`
	IsDouble bool   `json:"is_double"`
}

// S2CPlayerUpdate 玩家增量更新
type S2CPlayerUpdate struct {
	Idx    int        `json:"idx"`
	Player GamePlayer `json:"player"`
}

// S2CSpaceUpdate 地块增量更新
type S2CSpaceUpdate struct {
	Space GameSpace `json:"space"`
}

// S2CEventMsg 游戏事件文本消息
type S2CEventMsg struct {
	Text string `json:"text"`
}

// S2CChatMsg 聊天消息广播
type S2CChatMsg struct {
	From      string `json:"from"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}

// S2CError 错误消息
type S2CError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// S2CPlayerJoined 玩家加入通知
type S2CPlayerJoined struct {
	Name     string `json:"name"`
	CharName string `json:"char_name"`
}

// S2CPlayerLeft 玩家离开通知
type S2CPlayerLeft struct {
	Name          string `json:"name"`
	Reconnectable bool   `json:"reconnectable"`
}

// S2CGameOver 游戏结束通知
type S2CGameOver struct {
	WinnerName string `json:"winner_name"`
}

// S2CReconnectOK 重连成功，附带令牌
type S2CReconnectOK struct {
	Token string `json:"token"`
}

// S2CSpectateOK 观战加入成功确认
type S2CSpectateOK struct {
	RoomID string `json:"room_id"`
}

// RoomListItem 房间列表中的单个房间摘要
type RoomListItem struct {
	RoomID      string   `json:"room_id"`
	PlayerCount int      `json:"player_count"`
	MaxPlayers  int      `json:"max_players"`
	Started     bool     `json:"started"`
	Players     []string `json:"players"` // 玩家名列表
}

// S2CRoomList 服务器广播的可用房间列表
type S2CRoomList struct {
	Rooms []RoomListItem `json:"rooms"`
}
