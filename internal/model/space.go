package model

import "image"

type SpaceType int

const (
	SpaceEvent     SpaceType = 0
	SpaceLand      SpaceType = 1
	SpaceUtility   SpaceType = 2
	SpaceTransport SpaceType = 3
)

type Orientation int

const (
	OrientLeft Orientation = iota
	OrientRight
	OrientUp
	OrientDown
	OrientLeftUp
	OrientLeftDown
	OrientRightUp
	OrientRightDown
)

type Space struct {
	ID        int
	Name      string
	Rect      image.Rectangle
	Type      SpaceType
	BasePrice int
	Price     int
	Color     string
	Orient    Orientation

	Owner     *Player
	Houses    int
	HasHotel  bool
	Mortgaged bool
}

func (s *Space) BuildCost() int {
	return s.Price / 2
}

func (s *Space) IsBuyable() bool {
	return s.Type != SpaceEvent && s.Owner == nil
}

func (s *Space) Center() image.Point {
	return image.Point{
		X: s.Rect.Min.X + s.Rect.Dx()/2,
		Y: s.Rect.Min.Y + s.Rect.Dy()/2,
	}
}
