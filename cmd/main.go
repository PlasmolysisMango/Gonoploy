package main

import (
	_ "embed"
	"image/color"
	"log"

	assetfont "github.com/PlasmolysisMango/Gonopoly/asset/font"
	"github.com/hajimehoshi/ebiten/v2"
)

type Game struct {
	textMgr *assetfont.TextCache
}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{0, 0, 0, 255})

	// 显示中文
	screen.DrawImage(g.textMgr.GetImage("中文", color.White), &ebiten.DrawImageOptions{})
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 800, 600
}

func main() {
	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("中文显示示例")
	textMgr := assetfont.NewTextCache(16)

	if err := ebiten.RunGame(&Game{
		textMgr: textMgr,
	}); err != nil {
		log.Fatal(err)
	}
}
