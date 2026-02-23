package game

import (
	"fmt"
	"strings"
	"time"
)

const frameBudgetMs = 1000.0 / 60.0
const perfLogInterval = 2 * time.Second

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
	lastLog time.Time

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

func (p *framePerfProbe) formatLogReport(bullets, coins, particles, shards int) string {
	if !p.enabled {
		return ""
	}
	now := time.Now()
	if !p.lastLog.IsZero() && now.Sub(p.lastLog) < perfLogInterval {
		return ""
	}
	p.lastLog = now

	totalAvg := p.updateTotal.avgMs + p.drawTotal.avgMs
	totalMax := p.updateTotal.maxMs + p.drawTotal.maxMs
	bottleneckName, bottleneckMs := p.dominantSection()

	analysis := "within frame budget"
	switch {
	case totalAvg > frameBudgetMs*1.2:
		analysis = fmt.Sprintf("budget exceeded by %.2fms", totalAvg-frameBudgetMs)
	case totalAvg > frameBudgetMs:
		analysis = fmt.Sprintf("near budget limit (+%.2fms)", totalAvg-frameBudgetMs)
	}

	hints := make([]string, 0, 2)
	if p.drawDanmaku.avgMs >= 2.0 {
		hints = append(hints, "draw_danmaku dominates, reduce bullets or shader cost")
	}
	if p.updateDanmaku.avgMs >= 1.0 {
		hints = append(hints, "update_danmaku cost is rising, simplify pattern math")
	}
	if len(hints) == 0 {
		hints = append(hints, "no clear hotspot")
	}

	return fmt.Sprintf(
		"[PERF] target=%.2fms avg(upd=%.2f drw=%.2f total=%.2f) max(total=%.2f) obj(blt=%d coin=%d ptc=%d shd=%d) bottleneck=%s(%.2fms) analysis=%s hint=%s",
		frameBudgetMs,
		p.updateTotal.avgMs,
		p.drawTotal.avgMs,
		totalAvg,
		totalMax,
		bullets,
		coins,
		particles,
		shards,
		bottleneckName,
		bottleneckMs,
		analysis,
		strings.Join(hints, "; "),
	)
}

func (p *framePerfProbe) dominantSection() (string, float64) {
	candidates := []struct {
		name string
		ms   float64
	}{
		{name: "draw_danmaku", ms: p.drawDanmaku.avgMs},
		{name: "draw_particles", ms: p.drawParticles.avgMs},
		{name: "draw_portrait", ms: p.drawPortrait.avgMs},
		{name: "draw_backdrop", ms: p.drawBackdrop.avgMs},
		{name: "draw_ui", ms: p.drawUI.avgMs},
		{name: "update_danmaku", ms: p.updateDanmaku.avgMs},
		{name: "update_particles", ms: p.updateParticles.avgMs},
		{name: "update_portrait", ms: p.updatePortrait.avgMs},
	}
	name := "none"
	maxMs := 0.0
	for _, c := range candidates {
		if c.ms > maxMs {
			name = c.name
			maxMs = c.ms
		}
	}
	return name, maxMs
}
