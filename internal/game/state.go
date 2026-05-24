package game

type GameState int

const (
	StateCharSelect GameState = iota
	StatePlaying
	StateGameOver
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
