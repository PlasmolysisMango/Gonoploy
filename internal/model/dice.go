package model

import "math/rand"

type Dice struct {
	Values   [2]int
	Sum      int
	IsDouble bool
	Rolled   bool

	Rolling      bool
	RollTicks    int
	RollDuration int
}

func NewDice() *Dice {
	return &Dice{
		RollDuration: 30,
	}
}

func (d *Dice) StartRoll() {
	d.Rolling = true
	d.Rolled = false
	d.RollTicks = 0
}

func (d *Dice) Tick() {
	if !d.Rolling {
		return
	}
	d.RollTicks++
	d.Values[0] = rand.Intn(6) + 1
	d.Values[1] = rand.Intn(6) + 1

	if d.RollTicks >= d.RollDuration {
		d.Rolling = false
		d.Rolled = true
		d.Sum = d.Values[0] + d.Values[1]
		d.IsDouble = d.Values[0] == d.Values[1]
	}
}

func (d *Dice) Reset() {
	d.Rolled = false
	d.Rolling = false
	d.RollTicks = 0
}
