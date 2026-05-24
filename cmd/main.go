package main

import (
	"log"

	"github.com/PlasmolysisMango/Gonopoly/internal/game"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	ebiten.SetWindowSize(1600, 900)
	ebiten.SetWindowTitle("Gonoploy - 大富翁")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	g := game.New()

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
