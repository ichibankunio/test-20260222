package game

func newStage9Danmaku() Danmaku {
	return newStageDanmaku(func(d *stageDanmaku, frame int, _, _ float64) {
		if frame%14 == 0 {
			turn := float64((frame / 14) % 72)
			d.spawnSpread(ScreenWidth/2, 18, 1.04, []float64{60 + turn*5, 120 + turn*5})
		}
		if frame%46 == 14 {
			d.spawnSpread(-8, 84, 1.0, []float64{14, 26})
			d.spawnSpread(ScreenWidth+8, 112, 1.0, []float64{154, 166})
		}
		if frame%56 == 12 {
			xs := []float64{24, 48, 72, 96, 120, 96, 72, 48}
			d.spawnCoin(xs[(frame/56)%len(xs)], -6, 0, 0.96, 22)
		}
	})
}
