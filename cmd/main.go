package main

import (
	"image"
	_ "image/jpeg"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type Game struct {
	posX, posY int
	img        *ebiten.Image
}

func (g *Game) Update() error {
	if g.posX < 320 {
		g.posX += 1
	}
	if g.posY < 240 {
		g.posY += 1
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	ebitenutil.DebugPrintAt(screen, "Hello World", g.posX, g.posY)
	geom := ebiten.GeoM{}
	geom.Scale(0.2, 0.2)
	geom.Rotate(0.5)
	op := &ebiten.DrawImageOptions{
		GeoM:   geom,
		Filter: ebiten.FilterNearest,
	}
	// 可以在这里设置op.GeoM进行位置、旋转等变换
	screen.DrawImage(g.img, op)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 320, 240
}

var _ ebiten.Game = &Game{}

func loadJpegImage(path string) (*ebiten.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	return ebiten.NewImageFromImage(img), nil
}

func main() {
	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowTitle("Hello, World!")
	ebiten.SetTPS(30)
	img, err := loadJpegImage("asset/pics/hotal.jpg")
	if err != nil {
		log.Fatal(err)
	}
	if err := ebiten.RunGame(&Game{img: img}); err != nil {
		log.Fatal(err)
	}
}
