package res

import (
	"bytes"
	_ "embed"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

//go:embed icons/黑子.png
var iconKuroko []byte

//go:embed icons/泪子.png
var iconRuiko []byte

//go:embed icons/警策.png
var iconKouzaku []byte

//go:embed icons/食蜂.png
var iconShokuhou []byte

//go:embed icons/当麻.png
var iconTouma []byte

//go:embed icons/美琴.png
var iconMikoto []byte

//go:embed icons/初春.png
var iconUiharu []byte

//go:embed icons/削板.png
var iconSogiita []byte

//go:embed icons/婚后.png
var iconKongou []byte

//go:embed res/flag.png
var resFlag []byte

//go:embed res/house.png
var resHouse []byte

//go:embed res/hotal.png
var resHotel []byte

//go:embed icon.png
var appIcon []byte

type Assets struct {
	Icons   map[string]*ebiten.Image
	Flag    *ebiten.Image
	House   *ebiten.Image
	Hotel   *ebiten.Image
	AppIcon image.Image
}

func decodeImage(data []byte) *ebiten.Image {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		log.Fatalf("decode image: %v", err)
	}
	return ebiten.NewImageFromImage(img)
}

func decodeRawImage(data []byte) image.Image {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		log.Fatalf("decode image: %v", err)
	}
	return img
}

func Load() *Assets {
	a := &Assets{
		Icons: make(map[string]*ebiten.Image),
	}

	a.Icons["黑子"] = decodeImage(iconKuroko)
	a.Icons["泪子"] = decodeImage(iconRuiko)
	a.Icons["警策"] = decodeImage(iconKouzaku)
	a.Icons["食蜂"] = decodeImage(iconShokuhou)
	a.Icons["当麻"] = decodeImage(iconTouma)
	a.Icons["美琴"] = decodeImage(iconMikoto)
	a.Icons["初春"] = decodeImage(iconUiharu)
	a.Icons["削板"] = decodeImage(iconSogiita)
	a.Icons["婚后"] = decodeImage(iconKongou)

	a.Flag = decodeImage(resFlag)
	a.House = decodeImage(resHouse)
	a.Hotel = decodeImage(resHotel)
	a.AppIcon = decodeRawImage(appIcon)

	return a
}

func CharacterNames() []string {
	return []string{"黑子", "泪子", "警策", "食蜂", "当麻", "美琴", "初春", "削板", "婚后"}
}
