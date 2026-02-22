package game

func newStage3Danmaku() Danmaku {
	return newStageDanmaku(func(d *stageDanmaku, frame int, _, _ float64) {
		if frame%26 == 0 {
			fibX := []float64{13, 21, 34, 55, 89, 123, 68, 42}
			x := fibX[(frame/26)%len(fibX)]
			d.spawnSpread(x, -8, 1.12, []float64{84, 96})
		}
		if frame%52 == 24 {
			d.spawnSpread(ScreenWidth/2, 42, 1.08, []float64{202, 338})
		}
		if frame%18 == 2 {
			xs := []float64{24, 48, 72, 96, 120, 96, 72}
			d.spawnCoin(xs[(frame/18)%len(xs)], -6, 0, 1.0, 20)
		}
	})
}
