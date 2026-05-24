package game

import (
	_ "embed"
	"image/color"
	"math/rand"

	assetfont "github.com/PlasmolysisMango/Gonopoly/asset/font"
	asset "github.com/PlasmolysisMango/Gonopoly/asset/pics"
	"github.com/PlasmolysisMango/Gonopoly/internal/model"
	"github.com/PlasmolysisMango/Gonopoly/internal/network"
	"github.com/PlasmolysisMango/Gonopoly/internal/render"
	"github.com/PlasmolysisMango/Gonopoly/internal/ui"
	"github.com/PlasmolysisMango/Gonopoly/pkg/protocol"
	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed buildings.config
var buildingsConfig string

type Game struct {
	state     GameState
	turnPhase TurnPhase
	menuState MenuState

	board     *model.Board
	players   []*model.Player
	activeIdx int
	dice      *model.Dice

	renderer *render.Renderer
	assets   *asset.Assets
	uiMgr    *ui.Manager
	textLg   *assetfont.TextCache

	// 大厅/菜单专用字体（更具仪式感的层级）
	textTitle *assetfont.TextCache // 56pt - GONOPLOY 标题
	textHead  *assetfont.TextCache // 28pt - 段落标题
	textBody  *assetfont.TextCache // 20pt - 正文
	textHint  *assetfont.TextCache // 14pt - 键帽 / 副信息

	// menuTick 驱动菜单/大厅页面的轻微动画（光标闪烁、闪光等）
	menuTick int
	// lobbyReady 仅用于本地展示当前是否已点过准备
	lobbyReady bool

	charge    int
	animating bool

	// Movement state
	moveSteps   int
	moveCounter int
	moveTick    int

	// Build/Mortgage/Deal state
	buildCursor    int
	buildableSpaces []*model.Space
	mortgageSpaces  []*model.Space
	redeemSpaces    []*model.Space
	dealSpaces      []*model.Space
	dealTarget      int

	// Audio & Save
	audioMgr      *AudioManager
	autoSaveTick  int

	// AI 思考延迟计数器
	aiThinkTimer int

	// Character selection state
	selectedChars []string
	charCursor    int

	// 联机模式相关
	mode      GameMode            // 当前游戏模式
	netClient *network.NetClient  // 网络客户端（仅联机模式使用）
	localIdx  int                 // 本机玩家在 players 数组中的下标
	myName    string              // 本机玩家名字
	roomID    string              // 当前联机房间 ID
	isSpectator bool              // 当前是否以观众身份连接

	// 菜单/大厅并不复用 turnPhase 菜单；使用独立的模式字段。
	menuCursor   int    // 主菜单光标（0=本地，1=联机）
	lobbyCursor  int    // 大厅光标（0=创建房间，1=加入房间）
	lobbyServer  string // 服务器地址输入
	lobbyRoomID  string // 输入的房间 ID
	lobbyName    string // 输入的玩家名
	lobbyCharName string // 选中的角色
	lobbyField   int    // 当前输入焦点（0=server，1=name，2=room，3=char）
	lobbyAction  int    // 0=创建、1=加入
	lobbyMessage string // 大厅状态/错误提示
	waitingPlayers []string // 等待页说明玩家列表

	// 联机大厅中的可用房间列表缓存
	roomList    []protocol.RoomListItem // 服务器返回的房间列表
	roomListIdx int                     // 当前选中的房间下标
	roomListAt  int64                   // 上次刷新时间戳（menuTick）用于提示、闪烁
	lobbyFocus  int                     // 0=表单区，1=房间列表区
}

func New() *Game {
	assets := asset.Load()
	board := model.ParseBoard(buildingsConfig)
	renderer := render.NewRenderer()
	renderer.InitBoardBackground(board)

	g := &Game{
		state:       StateMainMenu,
		mode:        ModeLocal,
		board:       board,
		dice:        model.NewDice(),
		renderer:    renderer,
		assets:      assets,
		textLg:      assetfont.NewTextCache(24),
		textTitle:   assetfont.NewTextCache(56),
		textHead:    assetfont.NewTextCache(28),
		textBody:    assetfont.NewTextCache(20),
		textHint:    assetfont.NewTextCache(14),
		lobbyServer: "ws://127.0.0.1:8080/ws",
		lobbyName:   "Player",
	}

	g.uiMgr = ui.NewManager(renderer.TextMed())
	return g
}

func (g *Game) Update() error {
	switch g.state {
	case StateMainMenu:
		g.updateMainMenu()
	case StateLobby:
		g.updateLobby()
	case StateWaiting:
		g.updateWaiting()
	case StateCharSelect:
		g.updateCharSelect()
	case StatePlaying:
		if g.mode == ModeOnline {
			g.updatePlayingOnline()
		} else {
			g.updatePlaying()
		}
	case StateSpectating:
		g.updateSpectating()
	case StateGameOver:
		g.updateGameOver()
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	switch g.state {
	case StateMainMenu:
		g.drawMainMenu(screen)
	case StateLobby:
		g.drawLobby(screen)
	case StateWaiting:
		g.drawWaiting(screen)
	case StateCharSelect:
		g.drawCharSelect(screen)
	case StatePlaying:
		g.drawPlaying(screen)
	case StateSpectating:
		g.drawSpectating(screen)
	case StateGameOver:
		g.drawGameOver(screen)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return render.ScreenWidth, render.ScreenHeight
}

func (g *Game) ActivePlayer() *model.Player {
	if g.activeIdx < len(g.players) {
		return g.players[g.activeIdx]
	}
	return nil
}

func (g *Game) nextPlayer() {
	for {
		g.activeIdx = (g.activeIdx + 1) % len(g.players)
		if !g.players[g.activeIdx].Bankrupt {
			break
		}
	}
	g.players[g.activeIdx].HasOperated = false
}

func (g *Game) alivePlayers() int {
	count := 0
	for _, p := range g.players {
		if !p.Bankrupt {
			count++
		}
	}
	return count
}

func (g *Game) updateCharSelect() {
	names := asset.CharacterNames()

	if inpututil_isKeyJustPressed(ebiten.KeyEnter) && len(g.selectedChars) >= 1 {
		// 从未选择的角色中随机挑选 AI 填充至 4 人
		var remaining []string
		for _, n := range names {
			if !contains(g.selectedChars, n) {
				remaining = append(remaining, n)
			}
		}
		rand.Shuffle(len(remaining), func(i, j int) {
			remaining[i], remaining[j] = remaining[j], remaining[i]
		})

		aiNames := []string{}
		need := 4 - len(g.selectedChars)
		if need > len(remaining) {
			need = len(remaining)
		}
		for i := 0; i < need; i++ {
			aiNames = append(aiNames, remaining[i])
		}
		g.startGame(aiNames)
		return
	}

	if inpututil_isKeyJustPressed(ebiten.KeyRight) {
		g.charCursor = (g.charCursor + 1) % len(names)
	}
	if inpututil_isKeyJustPressed(ebiten.KeyLeft) {
		g.charCursor = (g.charCursor - 1 + len(names)) % len(names)
	}
	if inpututil_isKeyJustPressed(ebiten.KeySpace) {
		name := names[g.charCursor]
		if !contains(g.selectedChars, name) && len(g.selectedChars) < 4 {
			g.selectedChars = append(g.selectedChars, name)
		} else if contains(g.selectedChars, name) {
			g.selectedChars = removeStr(g.selectedChars, name)
		}
	}
}

func (g *Game) startGame(aiNames []string) {
	dir := 0
	for _, name := range g.selectedChars {
		icon := g.assets.Icons[name]
		p := model.NewPlayer(name, icon, dir)
		g.players = append(g.players, p)
		dir++
	}
	for _, name := range aiNames {
		icon := g.assets.Icons[name]
		p := model.NewPlayer(name, icon, dir)
		p.IsAI = true
		g.players = append(g.players, p)
		dir++
	}
	g.state = StatePlaying
	g.turnPhase = TurnWaitRoll
	g.audioMgr = NewAudioManager()
	g.uiMgr.AddMessage("游戏开始!")
	g.uiMgr.AddMessage("轮到 " + g.players[0].Name)
	g.updatePlayerInfoUI()
}

func (g *Game) drawCharSelect(screen *ebiten.Image) {
	screen.Fill(color.RGBA{30, 30, 50, 255})

	titleImg := g.textLg.GetImage("选择角色 (空格选择, Enter开始)", color.White)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(400, 50)
	screen.DrawImage(titleImg, op)

	tipImg := g.renderer.TextMed().GetImage("选择1-4个角色，剩余位置由AI填充", color.RGBA{200, 200, 200, 255})
	top := &ebiten.DrawImageOptions{}
	top.GeoM.Translate(500, 100)
	screen.DrawImage(tipImg, top)

	names := asset.CharacterNames()
	for i, name := range names {
		x := 100 + (i%5)*280
		y := 150 + (i/5)*300

		icon := g.assets.Icons[name]
		iop := &ebiten.DrawImageOptions{}
		iw, ih := icon.Bounds().Dx(), icon.Bounds().Dy()
		scale := 120.0 / float64(iw)
		if float64(ih)*scale > 120 {
			scale = 120.0 / float64(ih)
		}
		iop.GeoM.Scale(scale, scale)
		iop.GeoM.Translate(float64(x), float64(y))
		screen.DrawImage(icon, iop)

		nameImg := g.renderer.TextMed().GetImage(name, color.White)
		nop := &ebiten.DrawImageOptions{}
		nop.GeoM.Translate(float64(x), float64(y+130))
		screen.DrawImage(nameImg, nop)

		if i == g.charCursor {
			clr := color.RGBA{255, 255, 0, 255}
			cursorImg := g.renderer.TextMed().GetImage("▶", clr)
			cop := &ebiten.DrawImageOptions{}
			cop.GeoM.Translate(float64(x-25), float64(y+60))
			screen.DrawImage(cursorImg, cop)
		}

		if contains(g.selectedChars, name) {
			selImg := g.renderer.TextMed().GetImage("✓", color.RGBA{0, 255, 0, 255})
			sop := &ebiten.DrawImageOptions{}
			sop.GeoM.Translate(float64(x+100), float64(y))
			screen.DrawImage(selImg, sop)
		}
	}

	if len(g.selectedChars) >= 1 {
		hintImg := g.renderer.TextMed().GetImage("按Enter开始游戏", color.RGBA{200, 200, 200, 255})
		hop := &ebiten.DrawImageOptions{}
		hop.GeoM.Translate(650, 800)
		screen.DrawImage(hintImg, hop)
	}
}

func (g *Game) drawPlaying(screen *ebiten.Image) {
	g.renderer.DrawBoard(screen)
	g.renderer.DrawPropertyStatus(screen, g.board)
	g.uiMgr.Draw(screen)
	g.renderer.DrawDice(screen, g.dice)
	g.renderer.DrawPlayers(screen, g.players, g.board)
	// 联机模式绘制聊天 UI
	if g.mode == ModeOnline {
		g.uiMgr.DrawChat(screen)
	}
}

func (g *Game) drawGameOver(screen *ebiten.Image) {
	screen.Fill(color.RGBA{20, 20, 40, 255})
	var winner *model.Player
	for _, p := range g.players {
		if !p.Bankrupt {
			winner = p
			break
		}
	}
	if winner != nil {
		msg := winner.Name + " 获胜!"
		img := g.textLg.GetImage(msg, color.RGBA{255, 215, 0, 255})
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(600, 400)
		screen.DrawImage(img, op)
	}
}

func (g *Game) updateGameOver() {
	if inpututil_isKeyJustPressed(ebiten.KeyEscape) {
		// Could restart, for now just do nothing
	}
}

func (g *Game) updatePlayerInfoUI() {
	p := g.ActivePlayer()
	if p != nil {
		g.uiMgr.SetPlayerInfo(p.Name, p.Money, p.SkillPoints)
	}
}

func (g *Game) tickAutoSave() {
	g.autoSaveTick++
	if g.autoSaveTick >= 3600 { // 60 seconds at 60 TPS
		g.autoSaveTick = 0
		g.Save("saves/auto.json")
	}
}

func contains(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}

func removeStr(list []string, s string) []string {
	for i, item := range list {
		if item == s {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}
