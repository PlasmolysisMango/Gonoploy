package font

import (
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// TextCache 缓存渲染好的文本图像
type TextCache struct {
	cache map[string]*ebiten.Image
	face  font.Face
}

//go:embed font.ttf
var fontBytes []byte

func NewTextCache(fontSize float64) *TextCache {
	tt, err := opentype.Parse(fontBytes)
	if err != nil {
		log.Fatal(err)
	}

	face, err := opentype.NewFace(tt, &opentype.FaceOptions{
		Size: fontSize,
		DPI:  72,
	})
	if err != nil {
		log.Fatal(err)
	}

	return &TextCache{
		cache: make(map[string]*ebiten.Image),
		face:  face,
	}
}

// GetImage 获取或创建文本图像
func (tc *TextCache) GetImage(text string, clr color.Color) *ebiten.Image {
	key := text + "|" + fmt.Sprintf("%#v", clr)

	if img, exists := tc.cache[key]; exists {
		return img
	}

	// 测量文本尺寸
	d := font.Drawer{
		Face: tc.face,
	}
	bounds, _ := d.BoundString(text)
	width := (bounds.Max.X - bounds.Min.X).Ceil()
	height := (bounds.Max.Y - bounds.Min.Y).Ceil()

	if width <= 0 || height <= 0 {
		width, height = 1, 1
	}

	// 创建 RGBA 图像
	rgba := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(rgba, rgba.Bounds(), &image.Uniform{color.RGBA{0, 0, 0, 0}}, image.Point{}, draw.Src)

	// 渲染文本到 RGBA 图像
	d.Src = image.NewUniform(clr)
	d.Dst = rgba
	d.Dot = fixed.P(-bounds.Min.X.Ceil(), -bounds.Min.Y.Ceil())
	d.DrawString(text)

	// 转换为 Ebiten 图像
	ebitenImg := ebiten.NewImageFromImage(rgba)
	tc.cache[key] = ebitenImg

	return ebitenImg
}
