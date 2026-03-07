package mathutil

import "testing"

func TestClamp(t *testing.T) {
	tests := []struct {
		name        string
		v           float64
		minV        float64
		maxV        float64
		wantClamped float64
	}{
		{name: "below min", v: -1, minV: 0, maxV: 10, wantClamped: 0},
		{name: "within range", v: 5, minV: 0, maxV: 10, wantClamped: 5},
		{name: "above max", v: 11, minV: 0, maxV: 10, wantClamped: 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Clamp(tt.v, tt.minV, tt.maxV)
			if got != tt.wantClamped {
				t.Fatalf("Clamp(%v, %v, %v) = %v, want %v", tt.v, tt.minV, tt.maxV, got, tt.wantClamped)
			}
		})
	}
}

func TestLerp(t *testing.T) {
	tests := []struct {
		name string
		a    float64
		b    float64
		t    float64
		want float64
	}{
		{name: "middle", a: 0, b: 10, t: 0.5, want: 5},
		{name: "clamp low", a: 10, b: 20, t: -1, want: 10},
		{name: "clamp high", a: 10, b: 20, t: 2, want: 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Lerp(tt.a, tt.b, tt.t)
			if got != tt.want {
				t.Fatalf("Lerp(%v, %v, %v) = %v, want %v", tt.a, tt.b, tt.t, got, tt.want)
			}
		})
	}
}

func TestCirclesOverlap(t *testing.T) {
	tests := []struct {
		name       string
		ax         float64
		ay         float64
		ar         float64
		bx         float64
		by         float64
		br         float64
		wantResult bool
	}{
		{name: "overlap", ax: 0, ay: 0, ar: 5, bx: 3, by: 4, br: 1, wantResult: true},
		{name: "touching", ax: 0, ay: 0, ar: 3, bx: 6, by: 0, br: 3, wantResult: true},
		{name: "separate", ax: 0, ay: 0, ar: 2, bx: 10, by: 0, br: 2, wantResult: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CirclesOverlap(tt.ax, tt.ay, tt.ar, tt.bx, tt.by, tt.br)
			if got != tt.wantResult {
				t.Fatalf("CirclesOverlap(...) = %v, want %v", got, tt.wantResult)
			}
		})
	}
}
