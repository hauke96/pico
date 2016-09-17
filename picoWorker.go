package main

import (
	"fmt"
	"time"
)

type ImageRGBA struct {
	R [][]byte
	G [][]byte
	B [][]byte
	A [][]byte
}
type RGBAValue struct {
	R byte
	G byte
	B byte
	A byte
}

type PicoWorker interface {
	ProcessFile(file string) ImageRGBA
	SpecificActionOnData()
}

func AllocateImageArrayMemory(w, h int) ImageRGBA {
	fmt.Print("ALLOCATE MEMORY...")
	t1 := time.Now()

	// create data storage for the channels RGBA
	image2dR := make([][]byte, h)
	image2dG := make([][]byte, h)
	image2dB := make([][]byte, h)
	image2dA := make([][]byte, h)

	// allocate composed 2d array
	for i := 0; i < h; i++ {
		image2dR[i] = make([]byte, w)
		image2dG[i] = make([]byte, w)
		image2dB[i] = make([]byte, w)
		image2dA[i] = make([]byte, w)
	}
	duration := time.Since(t1)
	fmt.Printf("DONE (%d ms)\n", int(float32(duration.Nanoseconds())/1000000.0))

	return ImageRGBA{R: image2dR, G: image2dG, B: image2dB, A: image2dA}
}
