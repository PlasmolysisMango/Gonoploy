package game

import (
	"bytes"

	"github.com/PlasmolysisMango/Gonopoly/asset/music"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
)

var audioContext *audio.Context

func initAudio() *audio.Player {
	if audioContext == nil {
		audioContext = audio.NewContext(44100)
	}

	stream, err := mp3.DecodeF32(bytes.NewReader(music.BackgroundMP3))
	if err != nil {
		return nil
	}

	loop := audio.NewInfiniteLoop(stream, stream.Length())
	player, err := audioContext.NewPlayerF32(loop)
	if err != nil {
		return nil
	}

	player.SetVolume(0.05)
	player.Play()
	return player
}

type AudioManager struct {
	bgPlayer *audio.Player
	muted    bool
}

func NewAudioManager() *AudioManager {
	am := &AudioManager{}
	am.bgPlayer = initAudio()
	return am
}

func (am *AudioManager) ToggleMute() {
	if am.bgPlayer == nil {
		return
	}
	am.muted = !am.muted
	if am.muted {
		am.bgPlayer.Pause()
	} else {
		am.bgPlayer.Play()
	}
}

func (am *AudioManager) IsMuted() bool {
	return am.muted
}
