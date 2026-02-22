package game

func newStage5Danmaku() Danmaku {
	return newStageDanmaku(func(d *stageDanmaku, frame int, _, _ float64) {
		if frame%20 == 0 {
			step := (frame / 20) % 10
			x := 8.0 + float64(step)*14.0
			d.spawnSpread(x, -8, 1.15, []float64{86, 94})
			d.spawnSpread(ScreenWidth-x, -8, 1.15, []float64{86, 94})
		}
		if frame%48 == 12 {
			y := 70.0 + float64((frame/48)%3)*34.0
			d.spawnSpread(-8, y, 1.0, []float64{14})
			d.spawnSpread(ScreenWidth+8, y+10, 1.0, []float64{166})
		}
		if frame%56 == 18 {
			xs := []float64{30, 114, 42, 102, 54, 90, 66, 78}
			d.spawnCoin(xs[(frame/56)%len(xs)], -6, 0, 1.04, 20)
		}
	})
}
