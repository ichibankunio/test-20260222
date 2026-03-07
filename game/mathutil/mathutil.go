package mathutil

// Clamp limits v into the inclusive [minV, maxV] range.
func Clamp(v, minV, maxV float64) float64 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

// Lerp returns a linear interpolation where t is clamped into [0, 1].
func Lerp(a, b, t float64) float64 {
	return a + (b-a)*Clamp(t, 0, 1)
}

// CirclesOverlap reports whether two circles intersect or touch.
func CirclesOverlap(ax, ay, ar, bx, by, br float64) bool {
	dx := ax - bx
	dy := ay - by
	distSq := dx*dx + dy*dy
	r := ar + br
	return distSq <= r*r
}
