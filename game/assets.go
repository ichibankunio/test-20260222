package game

import (
	"bytes"
	"embed"
	"encoding/json"
	"log"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/ichibankunio/flib"
)

//go:embed assets/images
var imagesDir embed.FS

//go:embed assets/bgm
var bgmDir embed.FS

//go:embed assets/se
var seDir embed.FS

//go:embed assets/fonts
var fontsDir embed.FS

//go:embed assets/data
var dataDir embed.FS

type SampleJSON struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

var (
	assetAudioContext *audio.Context
	assetImages       = map[string]*ebiten.Image{}
	assetGoTextFaces  = map[int]*text.GoTextFace{}
	assetSE           [][]byte
	assetBGM          []*audio.Player
	assetJSON         = map[string]SampleJSON{}
)

func init() {
	assetAudioContext = audio.NewContext(SampleRate)
	loadImages()
	loadFonts()
	loadSE()
	loadBGM()
	loadJSON()
}

func loadImages() {
	entries, err := imagesDir.ReadDir("assets/images")
	if err != nil {
		log.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		b, err := imagesDir.ReadFile(filepath.Join("assets/images", name))
		if err != nil {
			log.Fatal(err)
		}
		assetImages[name] = flib.NewImageFromBytes(b)
	}
}

func loadFonts() {
	b, err := fontsDir.ReadFile("assets/fonts/sawarabi-gothic-medium.ttf")
	if err != nil {
		log.Fatal(err)
	}
	src, err := text.NewGoTextFaceSource(bytes.NewReader(b))
	if err != nil {
		log.Fatal(err)
	}
	assetGoTextFaces[16] = &text.GoTextFace{Source: src, Size: 16}
	assetGoTextFaces[28] = &text.GoTextFace{Source: src, Size: 28}
}

func loadSE() {
	entries, err := seDir.ReadDir("assets/se")
	if err != nil {
		log.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		b, err := seDir.ReadFile(filepath.Join("assets/se", e.Name()))
		if err != nil {
			log.Fatal(err)
		}
		assetSE = append(assetSE, b)
	}
}

func loadBGM() {
	entries, err := bgmDir.ReadDir("assets/bgm")
	if err != nil {
		log.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		b, err := bgmDir.ReadFile(filepath.Join("assets/bgm", e.Name()))
		if err != nil {
			log.Fatal(err)
		}
		assetBGM = append(assetBGM, flib.NewBGMFromBytes(b, SampleRate, assetAudioContext))
	}
}

func loadJSON() {
	entries, err := dataDir.ReadDir("assets/data")
	if err != nil {
		log.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		b, err := dataDir.ReadFile(filepath.Join("assets/data", e.Name()))
		if err != nil {
			log.Fatal(err)
		}
		var v SampleJSON
		if err := json.Unmarshal(b, &v); err != nil {
			log.Fatal(err)
		}
		assetJSON[e.Name()] = v
	}
}

func GetImage(name string) *ebiten.Image {
	return assetImages[name]
}

func GetGoTextFace(size int) *text.GoTextFace {
	if f := assetGoTextFaces[size]; f != nil {
		return f
	}
	return assetGoTextFaces[16]
}

func GetJSON(name string) (SampleJSON, bool) {
	v, ok := assetJSON[name]
	return v, ok
}

func FirstSE() []byte {
	if len(assetSE) == 0 {
		return nil
	}
	return assetSE[0]
}

func FirstBGM() *audio.Player {
	if len(assetBGM) == 0 {
		return nil
	}
	return assetBGM[0]
}

func PlaySE(b []byte) {
	s, err := mp3.DecodeWithSampleRate(SampleRate, bytes.NewReader(b))
	if err != nil {
		log.Fatal(err)
	}
	p, err := assetAudioContext.NewPlayer(s)
	if err != nil {
		log.Fatal(err)
	}
	flib.PlaySE(p)
}
