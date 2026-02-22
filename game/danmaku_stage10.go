package game

func newStage10Danmaku() Danmaku {
	return newStageDanmaku(func(d *stageDanmaku, frame int, playerX, playerY float64) {
		if frame%18 == 0 {
			offset := float64((frame / 18 * 11) % 360)
			d.spawnRing(ScreenWidth/2, 22, 1.08, 12, offset)
		}
		if frame%42 == 10 {
			d.spawnHoming(20, -8, 1.0, 0.045, 80, playerX, playerY)
			d.spawnHoming(ScreenWidth-20, -8, 1.0, 0.045, 80, playerX, playerY)
		}
		if frame%52 == 22 {
			xs := []float64{30, 114, 42, 102, 54, 90, 66, 78, 72}
			d.spawnCoin(xs[(frame/52)%len(xs)], -6, 0, 1.05, 20)
		}
	})
}
