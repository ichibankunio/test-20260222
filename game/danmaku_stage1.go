package game

func newStage1Danmaku() Danmaku {
	return newStageDanmaku(func(d *stageDanmaku, frame int, _, _ float64) {
		if frame%34 == 0 {
			d.spawnSpread(ScreenWidth/2, -8, 1.05, []float64{76, 90, 104})
		}
		if frame%68 == 20 {
			d.spawnSpread(-8, 92, 0.95, []float64{18, 30})
			d.spawnSpread(ScreenWidth+8, 122, 0.95, []float64{150, 162})
		}
		if frame%60 == 12 {
			xs := []float64{24, 48, 72, 96, 120, 96, 72, 48}
			d.spawnCoin(xs[(frame/60)%len(xs)], -6, 0, 0.88, 25)
		}
	})
}
