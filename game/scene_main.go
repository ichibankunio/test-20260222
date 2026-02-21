package game

import (
	"fmt"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/ichibankunio/flib"
)

type MainScene struct {
	playerX  float64
	playerY  float64
	playerVY float64
	cameraX  float64
}

func (s *MainScene) Init(_ *flib.Game) {
	s.playerX = 160
	s.playerY = groundTop() - playerHeight
	if bgm := FirstBGM(); bgm != nil {
		bgm.SetVolume(0.3)
		bgm.Play()
	}
}

func (s *MainScene) Start(_ *flib.Game) {}

func (s *MainScene) Update(_ *flib.Game) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return fmt.Errorf("exit")
	}

	moveX := 0.0
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) || ebiten.IsKeyPressed(ebiten.KeyD) {
		moveX += playerSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) || ebiten.IsKeyPressed(ebiten.KeyA) {
		moveX -= playerSpeed
	}
	s.cameraX += autoScrollSpeed
	s.playerX += autoScrollSpeed + moveX
	s.playerX = clamp(s.playerX, s.cameraX, s.cameraX+ScreenWidth-playerWidth)

	onGround := s.playerY >= groundTop()-playerHeight
	if onGround && (inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW)) {
		s.playerVY = jumpVelocity
		if se := FirstSE(); len(se) > 0 {
			PlaySE(se)
		}
	}
	s.playerVY += gravity
	s.playerY += s.playerVY

	floorY := groundTop() - playerHeight
	if s.playerY > floorY {
		s.playerY = floorY
		s.playerVY = 0
	}

	return nil
}

func (s *MainScene) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 92, G: 148, B: 252, A: 255})

	drawClouds(screen, s.cameraX)
	drawGround(screen, s.cameraX)
	drawPlayer(screen, s.playerX-s.cameraX, s.playerY)

	ebitenutil.DebugPrintAt(screen, "MOVE: <- -> / A D, JUMP: SPACE", 20, 20)
	ebitenutil.DebugPrintAt(screen, "ESC: EXIT", 20, 42)
}

const (
	groundH         = 360.0
	playerWidth     = 70.0
	playerHeight    = 96.0
	playerSpeed     = 7.5
	autoScrollSpeed = 4.0
	gravity         = 1.1
	jumpVelocity    = -21.0
	groundTileW     = 96.0
)

func groundTop() float64 {
	return ScreenHeight - groundH
}

func drawClouds(screen *ebiten.Image, cameraX float64) {
	const cloudSpacing = 420.0
	cloudCameraX := cameraX * 0.35
	start := -math.Mod(cloudCameraX, cloudSpacing)
	for x := start - cloudSpacing; x < ScreenWidth+cloudSpacing; x += cloudSpacing {
		screenX := x + 60
		drawFilledRect(screen, screenX, 140, 180, 54, color.RGBA{255, 255, 255, 255})
		drawFilledRect(screen, screenX+18, 116, 132, 34, color.RGBA{255, 255, 255, 255})
	}
}

func drawGround(screen *ebiten.Image, cameraX float64) {
	drawFilledRect(screen, 0, groundTop()-16, ScreenWidth, 16, color.RGBA{R: 225, G: 167, B: 98, A: 255})
	start := -math.Mod(cameraX, groundTileW)
	for x := start - groundTileW; x < ScreenWidth+groundTileW; x += groundTileW {
		tile := int((cameraX + x) / groundTileW)
		fill := color.RGBA{R: 188, G: 111, B: 52, A: 255}
		if tile%2 == 0 {
			fill = color.RGBA{R: 177, G: 103, B: 49, A: 255}
		}
		drawFilledRect(screen, x, groundTop(), groundTileW, groundH, fill)
		drawFilledRect(screen, x+8, groundTop()+8, 16, 16, color.RGBA{R: 227, G: 183, B: 129, A: 255})
	}
}

func drawPlayer(screen *ebiten.Image, x, y float64) {
	drawFilledRect(screen, x, y, playerWidth, playerHeight, color.RGBA{R: 228, G: 87, B: 64, A: 255})
	drawFilledRect(screen, x, y, playerWidth, 24, color.RGBA{R: 185, G: 34, B: 28, A: 255})
	drawFilledRect(screen, x+18, y+26, 34, 36, color.RGBA{R: 255, G: 218, B: 174, A: 255})
	drawFilledRect(screen, x+8, y+66, 20, 30, color.RGBA{R: 95, G: 118, B: 210, A: 255})
	drawFilledRect(screen, x+42, y+66, 20, 30, color.RGBA{R: 95, G: 118, B: 210, A: 255})
}

func drawFilledRect(screen *ebiten.Image, x, y, width, height float64, c color.Color) {
	vector.DrawFilledRect(screen, float32(x), float32(y), float32(width), float32(height), c, true)
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func (s *MainScene) GetStatus() int { return 0 }

func (s *MainScene) GetID() flib.SceneID { return SceneMain }
