package game

import "time"

const frameBudgetMs = 1000.0 / 60.0

type perfSample struct {
	lastMs float64
	avgMs  float64
	maxMs  float64
}

func (s *perfSample) update(d time.Duration) {
	ms := float64(d) / float64(time.Millisecond)
	s.lastMs = ms
	if s.avgMs == 0 {
		s.avgMs = ms
	} else {
		s.avgMs = s.avgMs*0.9 + ms*0.1
	}
	if ms > s.maxMs {
		s.maxMs = ms
	}
}

type framePerfProbe struct {
	enabled bool

	updateTotal     perfSample
	updateParticles perfSample
	updatePortrait  perfSample
	updateDanmaku   perfSample

	drawTotal     perfSample
	drawBackdrop  perfSample
	drawPortrait  perfSample
	drawDanmaku   perfSample
	drawParticles perfSample
	drawUI        perfSample
}

func (p *framePerfProbe) measure(sample *perfSample, fn func()) {
	if !p.enabled {
		fn()
		return
	}
	start := time.Now()
	fn()
	sample.update(time.Since(start))
}
