package model

import (
	"bufio"
	"fmt"
	"image"
	"math/rand"
	"strconv"
	"strings"
)

type Board struct {
	Spaces    []*Space
	ColorSets map[string][]*Space
}

func ParseBoard(data string) *Board {
	b := &Board{
		ColorSets: make(map[string][]*Space),
	}

	scanner := bufio.NewScanner(strings.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		s := parseLine(line)
		if s.Type == SpaceLand {
			offset := (rand.Intn(7) - 3) * 10
			s.Price = s.BasePrice + offset
		} else {
			s.Price = s.BasePrice
		}
		b.Spaces = append(b.Spaces, s)
		if s.Color != "" && s.Color != "NONE" {
			b.ColorSets[s.Color] = append(b.ColorSets[s.Color], s)
		}
	}
	return b
}

func parseLine(line string) *Space {
	parts := strings.SplitN(line, "|", 7)
	if len(parts) < 7 {
		panic(fmt.Sprintf("invalid config line: %s", line))
	}

	id, _ := strconv.Atoi(parts[0])
	name := parts[1]
	rect := parseRect(parts[2])
	stype, _ := strconv.Atoi(parts[3])
	price, _ := strconv.Atoi(parts[4])
	color := parts[5]
	orient := parseOrient(parts[6])

	return &Space{
		ID:        id,
		Name:      name,
		Rect:      rect,
		Type:      SpaceType(stype),
		BasePrice: price,
		Color:     color,
		Orient:    orient,
	}
}

func parseRect(s string) image.Rectangle {
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return image.Rectangle{}
	}
	x, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
	y, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
	w, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
	h, _ := strconv.Atoi(strings.TrimSpace(parts[3]))
	return image.Rect(x, y, x+w, y+h)
}

func parseOrient(s string) Orientation {
	switch s {
	case "Left":
		return OrientLeft
	case "Right":
		return OrientRight
	case "Up":
		return OrientUp
	case "Down":
		return OrientDown
	case "LeftUp":
		return OrientLeftUp
	case "LeftDown":
		return OrientLeftDown
	case "RightUp":
		return OrientRightUp
	case "RightDown":
		return OrientRightDown
	}
	return OrientLeft
}

func (b *Board) SpaceCount() int {
	return len(b.Spaces)
}

func (b *Board) OwnsFullColorSet(p *Player, color string) bool {
	spaces := b.ColorSets[color]
	for _, s := range spaces {
		if s.Owner != p {
			return false
		}
	}
	return true
}
