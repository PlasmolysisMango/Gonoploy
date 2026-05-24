package game

// GameMode 标识当前游戏运行模式：本地或联机。
type GameMode int

const (
	// ModeLocal 本地模式（单机+AI），原有逻辑不变。
	ModeLocal GameMode = iota
	// ModeOnline 联机模式，状态由服务端同步。
	ModeOnline
)

type GameState int

const (
	StateCharSelect GameState = iota // 保持原值 0
	StatePlaying                      // 保持原值 1
	StateGameOver                     // 保持原值 2
	StateMainMenu                     // 新增：主菜单
	StateLobby                        // 新增：联机大厅
	StateWaiting                      // 新增：等待房间内其他玩家
	StateSpectating                   // 新增：观战中
)

type TurnPhase int

const (
	TurnWaitRoll TurnPhase = iota
	TurnRolling
	TurnJumping
	TurnMoving
	TurnLanded
	TurnOperating
	TurnEndTurn
)

type MenuState int

const (
	MenuMain MenuState = iota
	MenuCharacter
	MenuSkill
	MenuSetting
	MenuHandle
	MenuBuild
	MenuMortgage
	MenuBuyback
	MenuDeal
	MenuBankrupt
)
