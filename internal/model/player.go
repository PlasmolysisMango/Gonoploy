package model

import "github.com/hajimehoshi/ebiten/v2"

type Player struct {
	Name      string
	Money     int
	Position  int
	Icon      *ebiten.Image
	Direction int

	OwnedLands      []*Space
	OwnedUtilities  []*Space
	OwnedTransports []*Space

	InJail        int
	JailPassports int
	SkillPoints   int
	Blessings     []Blessing
	BonusCount    int
	Bankrupt      bool
	HasOperated   bool
	IsAI          bool `json:"is_ai"`
}

type Blessing struct {
	Category string
	Modifier string
}

func NewPlayer(name string, icon *ebiten.Image, direction int) *Player {
	return &Player{
		Name:      name,
		Money:     1500,
		Position:  0,
		Icon:      icon,
		Direction: direction,
		SkillPoints: 1,
	}
}

func (p *Player) AddProperty(s *Space) {
	s.Owner = p
	switch s.Type {
	case SpaceLand:
		p.OwnedLands = append(p.OwnedLands, s)
	case SpaceUtility:
		p.OwnedUtilities = append(p.OwnedUtilities, s)
	case SpaceTransport:
		p.OwnedTransports = append(p.OwnedTransports, s)
	}
}

func (p *Player) RemoveProperty(s *Space) {
	s.Owner = nil
	switch s.Type {
	case SpaceLand:
		p.OwnedLands = removeSpace(p.OwnedLands, s)
	case SpaceUtility:
		p.OwnedUtilities = removeSpace(p.OwnedUtilities, s)
	case SpaceTransport:
		p.OwnedTransports = removeSpace(p.OwnedTransports, s)
	}
}

func removeSpace(list []*Space, s *Space) []*Space {
	for i, sp := range list {
		if sp == s {
			return append(list[:i], list[i+1:]...)
		}
	}
	return list
}

func (p *Player) TotalAssets() int {
	total := p.Money
	for _, s := range p.OwnedLands {
		total += s.Price / 2
		total += s.Houses * s.BuildCost() / 2
	}
	for _, s := range p.OwnedUtilities {
		total += s.Price / 2
	}
	for _, s := range p.OwnedTransports {
		total += s.Price / 2
	}
	return total
}

func (p *Player) GetCharge(s *Space, board *Board, diceSum int) int {
	if s.Owner == nil || s.Owner == p || s.Owner.InJail > 0 || s.Mortgaged {
		return 0
	}
	switch s.Type {
	case SpaceLand:
		if s.HasHotel {
			return s.Price * 3
		}
		if s.Houses > 0 {
			return s.Price * s.Houses / 2
		}
		if board.OwnsFullColorSet(s.Owner, s.Color) {
			return s.Price * 2 / 5
		}
		return s.Price / 5
	case SpaceUtility:
		return 8 * diceSum * len(s.Owner.OwnedUtilities)
	case SpaceTransport:
		return s.Price * len(s.Owner.OwnedTransports) / 5
	}
	return 0
}

func (p *Player) CanBuild(s *Space, board *Board) bool {
	if s.Type != SpaceLand || s.Owner != p || s.Mortgaged {
		return false
	}
	if !board.OwnsFullColorSet(p, s.Color) {
		return false
	}
	if s.HasHotel {
		return false
	}
	maxHouses := 0
	for _, sp := range board.ColorSets[s.Color] {
		if sp.Houses > maxHouses {
			maxHouses = sp.Houses
		}
	}
	return s.Houses <= maxHouses
}

func (p *Player) HasBlessing(category string) (Blessing, int) {
	for i, b := range p.Blessings {
		if b.Category == category {
			return b, i
		}
	}
	return Blessing{}, -1
}

func (p *Player) RemoveBlessing(idx int) {
	if idx >= 0 && idx < len(p.Blessings) {
		p.Blessings = append(p.Blessings[:idx], p.Blessings[idx+1:]...)
	}
}
