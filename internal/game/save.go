package game

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/PlasmolysisMango/Gonopoly/internal/model"
)

type SaveData struct {
	Players   []SavePlayer `json:"players"`
	ActiveIdx int          `json:"active_idx"`
	Spaces    []SaveSpace  `json:"spaces"`
}

type SavePlayer struct {
	Name          string        `json:"name"`
	Money         int           `json:"money"`
	Position      int           `json:"position"`
	Direction     int           `json:"direction"`
	InJail        int           `json:"in_jail"`
	JailPassports int           `json:"jail_passports"`
	SkillPoints   int           `json:"skill_points"`
	BonusCount    int           `json:"bonus_count"`
	Bankrupt      bool          `json:"bankrupt"`
	Blessings     []SaveBlessing `json:"blessings"`
}

type SaveBlessing struct {
	Category string `json:"category"`
	Modifier string `json:"modifier"`
}

type SaveSpace struct {
	ID        int    `json:"id"`
	OwnerName string `json:"owner_name"`
	Houses    int    `json:"houses"`
	HasHotel  bool   `json:"has_hotel"`
	Mortgaged bool   `json:"mortgaged"`
	Price     int    `json:"price"`
}

func (g *Game) Save(filename string) error {
	dir := filepath.Dir(filename)
	os.MkdirAll(dir, 0755)

	data := SaveData{
		ActiveIdx: g.activeIdx,
	}

	for _, p := range g.players {
		sp := SavePlayer{
			Name:          p.Name,
			Money:         p.Money,
			Position:      p.Position,
			Direction:     p.Direction,
			InJail:        p.InJail,
			JailPassports: p.JailPassports,
			SkillPoints:   p.SkillPoints,
			BonusCount:    p.BonusCount,
			Bankrupt:      p.Bankrupt,
		}
		for _, b := range p.Blessings {
			sp.Blessings = append(sp.Blessings, SaveBlessing{
				Category: b.Category,
				Modifier: b.Modifier,
			})
		}
		data.Players = append(data.Players, sp)
	}

	for _, s := range g.board.Spaces {
		ss := SaveSpace{
			ID:        s.ID,
			Houses:    s.Houses,
			HasHotel:  s.HasHotel,
			Mortgaged: s.Mortgaged,
			Price:     s.Price,
		}
		if s.Owner != nil {
			ss.OwnerName = s.Owner.Name
		}
		data.Spaces = append(data.Spaces, ss)
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, jsonData, 0644)
}

func (g *Game) Load(filename string) error {
	jsonData, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	var data SaveData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return err
	}

	g.activeIdx = data.ActiveIdx

	playerMap := make(map[string]int)
	for i, sp := range data.Players {
		if i >= len(g.players) {
			break
		}
		p := g.players[i]
		playerMap[p.Name] = i
		p.Money = sp.Money
		p.Position = sp.Position
		p.InJail = sp.InJail
		p.JailPassports = sp.JailPassports
		p.SkillPoints = sp.SkillPoints
		p.BonusCount = sp.BonusCount
		p.Bankrupt = sp.Bankrupt
		p.Blessings = nil
		for _, b := range sp.Blessings {
			p.Blessings = append(p.Blessings, model.Blessing{
				Category: b.Category,
				Modifier: b.Modifier,
			})
		}
		p.OwnedLands = nil
		p.OwnedUtilities = nil
		p.OwnedTransports = nil
	}

	for _, ss := range data.Spaces {
		if ss.ID >= len(g.board.Spaces) {
			continue
		}
		s := g.board.Spaces[ss.ID]
		s.Houses = ss.Houses
		s.HasHotel = ss.HasHotel
		s.Mortgaged = ss.Mortgaged
		s.Price = ss.Price
		s.Owner = nil

		if ss.OwnerName != "" {
			for _, p := range g.players {
				if p.Name == ss.OwnerName {
					p.AddProperty(s)
					break
				}
			}
		}
	}

	g.state = StatePlaying
	g.turnPhase = TurnWaitRoll
	g.menuState = MenuMain
	return nil
}
