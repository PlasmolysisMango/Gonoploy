package protocol

// 客户端到服务端消息类型常量
const (
	TypeCreateRoom  = "create_room"
	TypeJoinRoom    = "join_room"
	TypeLeaveRoom   = "leave_room"
	TypeReady       = "ready"
	TypeRollDice    = "roll_dice"
	TypeBuyProperty = "buy_property"
	TypeBuild       = "build"
	TypeUseSkill    = "use_skill"
	TypeMortgage    = "mortgage"
	TypeRedeem      = "redeem"
	TypeTrade       = "trade"
	TypeEndTurn     = "end_turn"
	TypeChat        = "chat"
	TypeReconnect   = "reconnect"
	TypeListRooms   = "list_rooms"
	TypeSpectate    = "spectate"
)

// C2SCreateRoom 创建房间请求
type C2SCreateRoom struct {
	PlayerName string `json:"player_name"`
	CharName   string `json:"char_name"`
}

// C2SJoinRoom 加入房间请求
type C2SJoinRoom struct {
	RoomID     string `json:"room_id"`
	PlayerName string `json:"player_name"`
	CharName   string `json:"char_name"`
}

// C2SLeaveRoom 离开房间请求
type C2SLeaveRoom struct{}

// C2SReady 准备就绪请求
type C2SReady struct{}

// C2SRollDice 掷骰请求
type C2SRollDice struct{}

// C2SBuyProperty 购买当前位置物业
type C2SBuyProperty struct{}

// C2SBuild 在指定地块建造
type C2SBuild struct {
	SpaceID int `json:"space_id"`
}

// C2SUseSkill 使用技能
type C2SUseSkill struct{}

// C2SMortgage 抵押地产
type C2SMortgage struct {
	SpaceID int `json:"space_id"`
}

// C2SRedeem 赎回地产
type C2SRedeem struct {
	SpaceID int `json:"space_id"`
}

// C2STrade 交易请求
type C2STrade struct {
	SpaceID      int    `json:"space_id"`
	TargetPlayer string `json:"target_player"`
}

// C2SEndTurn 结束回合
type C2SEndTurn struct{}

// C2SChat 聊天消息
type C2SChat struct {
	Content string `json:"content"`
}

// C2SReconnect 重连请求
type C2SReconnect struct {
	RoomID     string `json:"room_id"`
	PlayerName string `json:"player_name"`
	Token      string `json:"token"`
}

// C2SListRooms 房间列表查询请求（无负载）
type C2SListRooms struct{}

// C2SSpectate 观战请求
type C2SSpectate struct {
	RoomID     string `json:"room_id"`
	PlayerName string `json:"player_name"` // 观众昵称
}
