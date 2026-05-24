package game

import (
	_ "embed"
	"image/color"

	assetfont "github.com/PlasmolysisMango/Gonopoly/asset/font"
	asset "github.com/PlasmolysisMango/Gonopoly/asset/pics"
	"github.com/PlasmolysisMango/Gonopoly/internal/model"
	"github.com/PlasmolysisMango/Gonopoly/internal/render"
	"github.com/PlasmolysisMango/Gonopoly/internal/ui"
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

	// Character selection state
	selectedChars []string
	charCursor    int
}

func New() *Game {
	assets := asset.Load()
	board := model.ParseBoard(buildingsConfig)
	renderer := render.NewRenderer()
	renderer.InitBoardBackground(board)

	g := &Game{
		state:    StateCharSelect,
		board:    board,
		dice:     model.NewDice(),
		renderer: renderer,
		assets:   assets,
		textLg:   assetfont.NewTextCache(24),
	}

	g.uiMgr = ui.NewManager(renderer.TextMed())
	return g
}

func (g *Game) Update() error {
	switch g.state {
	case StateCharSelect:
		g.updateCharSelect()
	case StatePlaying:
		g.updatePlaying()
	case StateGameOver:
		g.updateGameOver()
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	switch g.state {
	case StateCharSelect:
		g.drawCharSelect(screen)
	case StatePlaying:
		g.drawPlaying(screen)
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
	if inpututil_isKeyJustPressed(ebiten.KeyEnter) && len(g.selectedChars) >= 2 {
		g.startGame()
		return
	}

	names := asset.CharacterNames()

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

func (g *Game) startGame() {
	for i, name := range g.selectedChars {
		icon := g.assets.Icons[name]
		p := model.NewPlayer(name, icon, i)
		g.players = append(g.players, p)
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

	if len(g.selectedChars) >= 2 {
		hintImg := g.renderer.TextMed().GetImage("按Enter开始游戏", color.RGBA{200, 200, 200, 255})
		hop := &ebiten.DrawImageOptions{}
		hop.GeoM.Translate(650, 800)
		screen.DrawImage(hintImg, hop)
	}
}

func (g *Game) drawPlaying(screen *ebiten.Image) {
	g.renderer.DrawBoard(screen)
	g.renderer.DrawPropertyStatus(screen, g.board)
	g.renderer.DrawPlayers(screen, g.players, g.board)
	g.renderer.DrawDice(screen, g.dice)
	g.uiMgr.Draw(screen)
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
