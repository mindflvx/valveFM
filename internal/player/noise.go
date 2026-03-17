package player

import (
	"math/rand"
	"sync"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/speaker"
)

// NoisePlayer plays infinite pink radio static during loading/buffering periods.
type NoisePlayer struct {
	mu      sync.Mutex
	ctrl    *beep.Ctrl
	playing bool
}

// NewNoisePlayer creates a NoisePlayer instance.
func NewNoisePlayer() *NoisePlayer {
	return &NoisePlayer{}
}

// Start begins playing radio static. Safe to call multiple times.
func (n *NoisePlayer) Start() {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.playing {
		return
	}

	if err := EnsureSpeaker(); err != nil {
		return
	}

	ctrl := &beep.Ctrl{Streamer: &radioStaticStreamer{}, Paused: false}
	speaker.Play(ctrl)
	n.ctrl = ctrl
	n.playing = true
}

// Stop halts static playback. Safe to call when not playing.
func (n *NoisePlayer) Stop() {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.playing || n.ctrl == nil {
		return
	}

	speaker.Lock()
	n.ctrl.Paused = true
	speaker.Unlock()

	n.ctrl = nil
	n.playing = false
}

// radioStaticStreamer generates infinite pink noise (radio static).
// Implements beep.Streamer.
type radioStaticStreamer struct {
	// Pink filter state — left channel
	lb0, lb1, lb2 float64
	// Pink filter state — right channel
	rb0, rb1, rb2 float64
	// Smoothed output for soft click removal
	lPrev, rPrev float64
	// Crackle burst counter
	crackle int
}

const (
	noiseAmp   = 0.09
	crackleAmp = 0.30
	smoothCoef = 0.75
)

func (s *radioStaticStreamer) Stream(samples [][2]float64) (int, bool) {
	for i := range samples {
		lw := (rand.Float64()*2 - 1)
		rw := (rand.Float64()*2 - 1)

		// Kellet 3-pole pink filter — left
		s.lb0 = 0.99886*s.lb0 + lw*0.0555179
		s.lb1 = 0.99332*s.lb1 + lw*0.0750759
		s.lb2 = 0.96900*s.lb2 + lw*0.1538520
		lPink := (s.lb0 + s.lb1 + s.lb2 + lw*0.5362) * 0.11

		// Kellet 3-pole pink filter — right
		s.rb0 = 0.99886*s.rb0 + rw*0.0555179
		s.rb1 = 0.99332*s.rb1 + rw*0.0750759
		s.rb2 = 0.96900*s.rb2 + rw*0.1538520
		rPink := (s.rb0 + s.rb1 + s.rb2 + rw*0.5362) * 0.11

		lSample := lPink * noiseAmp
		rSample := rPink * noiseAmp

		// Random crackle burst
		if s.crackle > 0 {
			burst := (rand.Float64()*2 - 1) * (crackleAmp + rand.Float64()*0.30)
			lSample += burst
			rSample += (rand.Float64()*2 - 1) * (crackleAmp + rand.Float64()*0.30)
			s.crackle--
		} else if rand.Intn(4000) == 0 {
			s.crackle = 3 + rand.Intn(8)
		}

		// Single-pole low-pass smoother to soften clicks
		lOut := smoothCoef*s.lPrev + (1-smoothCoef)*lSample
		rOut := smoothCoef*s.rPrev + (1-smoothCoef)*rSample
		s.lPrev = lOut
		s.rPrev = rOut

		samples[i][0] = lOut
		samples[i][1] = rOut
	}
	return len(samples), true
}

func (s *radioStaticStreamer) Err() error { return nil }
