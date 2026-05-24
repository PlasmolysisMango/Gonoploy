package render

import (
	"fmt"
	"image/color"

	assetfont "github.com/PlasmolysisMango/Gonopoly/asset/font"
	"github.com/PlasmolysisMango/Gonopoly/internal/model"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	ScreenWidth  = 1600
	ScreenHeight = 900
)

var colorMap = map[string]color.RGBA{
	"RED":    {220, 50, 50, 255},
	"GREEN":  {50, 180, 50, 255},
	"BLUE":   {50, 100, 220, 255},
	"PURPLE": {150, 50, 200, 255},
	"YELLOW": {220, 200, 50, 255},
	"GREY":   {150, 150, 150, 255},
}

type Renderer struct {
	boardBG   *ebiten.Image
	textSmall *assetfont.TextCache
	textMed   *assetfont.TextCache
	textName  *assetfont.TextCache
	textPrice *assetfont.TextCache
}

func NewRenderer() *Renderer {
	return &Renderer{
		textSmall: assetfont.NewTextCache(12),
		textMed:   assetfont.NewTextCache(16),
		textName:  assetfont.NewTextCache(20),
		textPrice: assetfont.NewTextCache(15),
	}
}

func (r *Renderer) InitBoardBackground(board *model.Board) {
	r.boardBG = ebiten.NewImage(ScreenWidth, ScreenHeight)
	r.boardBG.Fill(color.RGBA{200, 200, 200, 255})

	for _, s := range board.Spaces {
		r.drawSpace(s)
	}
}

func (r *Renderer) drawSpace(s *model.Space) {
	x := float32(s.Rect.Min.X)
	y := float32(s.Rect.Min.Y)
	w := float32(s.Rect.Dx())
	h := float32(s.Rect.Dy())

	// Corner spaces get colored backgrounds
	bgColor := color.RGBA{255, 255, 255, 255}
	if s.Type == model.SpaceEvent && s.Rect.Dx() == 200 && s.Rect.Dy() == 200 {
		switch s.ID {
		case 0: // 起点
			bgColor = color.RGBA{240, 200, 130, 255}
		case 5: // 技能
			bgColor = color.RGBA{240, 200, 130, 255}
		case 16: // 监狱
			bgColor = color.RGBA{180, 180, 180, 255}
		case 21: // 祝福
			bgColor = color.RGBA{240, 200, 130, 255}
		}
	}

	vector.DrawFilledRect(r.boardBG, x, y, w, h, bgColor, false)
	vector.StrokeRect(r.boardBG, x, y, w, h, 2, color.Black, false)

	if s.Color != "" && s.Color != "NONE" {
		clr := colorMap[s.Color]
		r.drawColorBar(s, clr)
	}

	r.drawSpaceText(s)
}

func (r *Renderer) drawColorBar(s *model.Space, clr color.RGBA) {
	x := float32(s.Rect.Min.X)
	y := float32(s.Rect.Min.Y)
	w := float32(s.Rect.Dx())
	h := float32(s.Rect.Dy())

	barSize := float32(30)
	switch s.Orient {
	case model.OrientLeft:
		vector.DrawFilledRect(r.boardBG, x+w-barSize, y, barSize, h, clr, false)
	case model.OrientRight:
		vector.DrawFilledRect(r.boardBG, x, y, barSize, h, clr, false)
	case model.OrientUp:
		vector.DrawFilledRect(r.boardBG, x, y+h-barSize, w, barSize, clr, false)
	case model.OrientDown:
		vector.DrawFilledRect(r.boardBG, x, y, w, barSize, clr, false)
	}
}

func (r *Renderer) drawSpaceText(s *model.Space) {
	center := s.Center()
	h := s.Rect.Dy()

	nameImg := r.textName.GetImage(s.Name, color.Black)
	nw, nh := nameImg.Bounds().Dx(), nameImg.Bounds().Dy()

	op := &ebiten.DrawImageOptions{}
	nameX := float64(center.X - nw/2)
	nameY := float64(center.Y - nh/2)
	if s.BasePrice > 0 {
		nameY = float64(center.Y - nh/2 - h/10)
	}
	// Clamp to space bounds
	if nameX < float64(s.Rect.Min.X+2) {
		nameX = float64(s.Rect.Min.X + 2)
	}
	op.GeoM.Translate(nameX, nameY)
	r.boardBG.DrawImage(nameImg, op)

	if s.BasePrice > 0 {
		priceStr := fmt.Sprintf("价格：%d元", s.BasePrice)
		priceImg := r.textPrice.GetImage(priceStr, color.Black)
		pw := priceImg.Bounds().Dx()
		op2 := &ebiten.DrawImageOptions{}
		priceX := float64(center.X - pw/2)
		if priceX < float64(s.Rect.Min.X+2) {
			priceX = float64(s.Rect.Min.X + 2)
		}
		op2.GeoM.Translate(priceX, float64(center.Y+h/10))
		r.boardBG.DrawImage(priceImg, op2)
	}
}

func (r *Renderer) DrawBoard(screen *ebiten.Image) {
	if r.boardBG != nil {
		screen.DrawImage(r.boardBG, nil)
	}
}

func (r *Renderer) DrawPropertyStatus(screen *ebiten.Image, board *model.Board) {
	for _, s := range board.Spaces {
		if s.Owner == nil {
			continue
		}
		center := s.Center()

		if s.Mortgaged {
			mortImg := r.textSmall.GetImage("抵", color.RGBA{255, 50, 50, 255})
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(center.X-6), float64(s.Rect.Min.Y+2))
			screen.DrawImage(mortImg, op)
			continue
		}

		ownerImg := r.textSmall.GetImage(string([]rune(s.Owner.Name)[0]), color.RGBA{0, 100, 200, 255})
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(s.Rect.Min.X+2), float64(s.Rect.Min.Y+2))
		screen.DrawImage(ownerImg, op)

		if s.HasHotel {
			hotelImg := r.textSmall.GetImage("★", color.RGBA{255, 50, 50, 255})
			hop := &ebiten.DrawImageOptions{}
			hop.GeoM.Translate(float64(s.Rect.Max.X-14), float64(s.Rect.Min.Y+2))
			screen.DrawImage(hotelImg, hop)
		} else if s.Houses > 0 {
			houseStr := fmt.Sprintf("%d", s.Houses)
			houseImg := r.textSmall.GetImage(houseStr, color.RGBA{50, 180, 50, 255})
			hop := &ebiten.DrawImageOptions{}
			hop.GeoM.Translate(float64(s.Rect.Max.X-12), float64(s.Rect.Min.Y+2))
			screen.DrawImage(houseImg, hop)
		}
	}
}

func (r *Renderer) DrawPlayers(screen *ebiten.Image, players []*model.Player, board *model.Board) {
	for _, p := range players {
		if p.Bankrupt {
			continue
		}
		s := board.Spaces[p.Position]
		center := s.Center()

		offsetX, offsetY := playerOffset(p.Direction)
		ix, iy := center.X+offsetX-20, center.Y+offsetY-20

		op := &ebiten.DrawImageOptions{}
		iconW := p.Icon.Bounds().Dx()
		iconH := p.Icon.Bounds().Dy()
		scaleX := 40.0 / float64(iconW)
		scaleY := 40.0 / float64(iconH)
		op.GeoM.Scale(scaleX, scaleY)
		op.GeoM.Translate(float64(ix), float64(iy))
		screen.DrawImage(p.Icon, op)
	}
}

func playerOffset(direction int) (int, int) {
	switch direction {
	case 0:
		return -20, -20
	case 1:
		return 20, -20
	case 2:
		return -20, 20
	case 3:
		return 20, 20
	}
	return 0, 0
}

func (r *Renderer) DrawDice(screen *ebiten.Image, dice *model.Dice) {
	baseX, baseY := 700, 410
	for i := 0; i < 2; i++ {
		dx := baseX + i*100
		dy := baseY
		vector.DrawFilledRect(screen, float32(dx), float32(dy), 80, 80, color.White, false)
		vector.StrokeRect(screen, float32(dx), float32(dy), 80, 80, 2, color.Black, false)
		r.drawDiceFace(screen, dx, dy, dice.Values[i])
	}
}

func (r *Renderer) drawDiceFace(screen *ebiten.Image, x, y, val int) {
	cx := float32(x + 40)
	cy := float32(y + 40)
	dotR := float32(6)
	clr := color.Black

	positions := dicePositions(val, cx, cy)
	for _, pos := range positions {
		vector.DrawFilledCircle(screen, pos[0], pos[1], dotR, clr, false)
	}
}

func dicePositions(val int, cx, cy float32) [][2]float32 {
	off := float32(20)
	switch val {
	case 1:
		return [][2]float32{{cx, cy}}
	case 2:
		return [][2]float32{{cx - off, cy - off}, {cx + off, cy + off}}
	case 3:
		return [][2]float32{{cx - off, cy - off}, {cx, cy}, {cx + off, cy + off}}
	case 4:
		return [][2]float32{{cx - off, cy - off}, {cx + off, cy - off}, {cx - off, cy + off}, {cx + off, cy + off}}
	case 5:
		return [][2]float32{{cx - off, cy - off}, {cx + off, cy - off}, {cx, cy}, {cx - off, cy + off}, {cx + off, cy + off}}
	case 6:
		return [][2]float32{{cx - off, cy - off}, {cx + off, cy - off}, {cx - off, cy}, {cx + off, cy}, {cx - off, cy + off}, {cx + off, cy + off}}
	}
	return nil
}

func (r *Renderer) TextSmall() *assetfont.TextCache { return r.textSmall }
func (r *Renderer) TextMed() *assetfont.TextCache   { return r.textMed }
