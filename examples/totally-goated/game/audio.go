package game

import (
	"bytes"
	"io"
	"io/fs"
	"log"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

const sampleRate = 44100

var audioCtx *audio.Context

type SoundPlayer struct {
	data []byte
}

var (
	sfxBellPickup    *SoundPlayer
	sfxBounce        *SoundPlayer
	sfxChargeRelease *SoundPlayer
	sfxCrumble       *SoundPlayer
	sfxDashPower5    *SoundPlayer
	sfxDashPower10   *SoundPlayer
	sfxDashPower15   *SoundPlayer
	sfxDeath         *SoundPlayer
	sfxIce           *SoundPlayer
	sfxMilestone     *SoundPlayer
	sfxPowerup       *SoundPlayer
	sfxShieldBreak   *SoundPlayer
	sfxWallHit       *SoundPlayer
)

func loadAudioAssets() {
	audioCtx = audio.NewContext(sampleRate)

	sfxBellPickup = loadWav("assets/sound/bell_pickup.wav")
	sfxBounce = loadWav("assets/sound/bounce.wav")
	sfxChargeRelease = loadWav("assets/sound/charge_release.wav")
	sfxCrumble = loadWav("assets/sound/crumble.wav")
	sfxDashPower5 = loadWav("assets/sound/dash_power5.wav")
	sfxDashPower10 = loadWav("assets/sound/dash_power10.wav")
	sfxDashPower15 = loadWav("assets/sound/dash_power15.wav")
	sfxDeath = loadWav("assets/sound/death.wav")
	sfxIce = loadWav("assets/sound/ice.wav")
	sfxMilestone = loadWav("assets/sound/milestone.wav")
	sfxPowerup = loadWav("assets/sound/powerup.wav")
	sfxShieldBreak = loadWav("assets/sound/shield_break.wav")
	sfxWallHit = loadWav("assets/sound/wall_hit.wav")
}

func loadWav(path string) *SoundPlayer {
	data, err := fs.ReadFile(assetsFS, path)
	if err != nil {
		log.Fatalf("audio: open %s: %v", path, err)
	}

	decoded, err := wav.DecodeF32(bytes.NewReader(data))
	if err != nil {
		log.Fatalf("audio: decode %s: %v", path, err)
	}

	pcm, err := io.ReadAll(decoded)
	if err != nil {
		log.Fatalf("audio: read %s: %v", path, err)
	}

	return &SoundPlayer{data: pcm}
}

func (s *SoundPlayer) Play() {
	if s == nil {
		return
	}
	p := audioCtx.NewPlayerF32FromBytes(s.data)
	p.Play()
}
