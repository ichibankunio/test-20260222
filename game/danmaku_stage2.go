package game

func newStage2Danmaku() Danmaku {
	return newStageDanmaku(func(d *stageDanmaku, frame int, _, _ float64) {
		if frame%22 == 0 {
			idx := (frame / 22) % 6
			x := 12.0 + float64(idx)*20.0
			d.spawnSpread(x, -8, 1.08, []float64{82, 90, 98})
		}
		if frame%44 == 12 {
			y := 66.0 + float64((frame/44)%4)*30.0
			d.spawnSpread(-8, y, 0.98, []float64{18})
			d.spawnSpread(ScreenWidth+8, y+14, 0.98, []float64{162})
		}
		if frame%56 == 16 {
			xs := []float64{18, 126, 38, 106, 58, 86, 72}
			d.spawnCoin(xs[(frame/56)%len(xs)], -6, 0, 0.94, 22)
		}
	})
}
