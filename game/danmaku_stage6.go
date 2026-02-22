package game

func newStage6Danmaku() Danmaku {
	return newStageDanmaku(func(d *stageDanmaku, frame int, playerX, playerY float64) {
		if frame%38 == 0 {
			x := 20.0 + float64((frame/38)%6)*20
			d.spawnHoming(x, -8, 0.95, 0.05, 70, playerX, playerY)
		}
		if frame%18 == 8 {
			d.spawnSpread(ScreenWidth/2, -8, 1.05, []float64{74, 90, 106})
		}
		if frame%20 == 0 {
			xs := []float64{20, 44, 68, 92, 116, 92, 68, 44}
			d.spawnCoin(xs[(frame/20)%len(xs)], -6, 0, 0.92, 24)
		}
	})
}
