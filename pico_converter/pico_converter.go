package main

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	// ------------------------------
	// PARSE ARGUMENTS
	// ------------------------------
	fmt.Print("PARSING ARGS...")
	if len(os.Args) < 3 {
		Panic("Please specify the *.png file AND accuracy as argument!\n")
	}
	accuracy64, err := strconv.ParseFloat(os.Args[2], 32)
	if err != nil {
		Panic("Please enter a float number as accurancy!\n")
	}
	accuracy := float32(accuracy64)
	fmt.Println("DONE")

	// ------------------------------
	// READ PNG DATA
	// ------------------------------
	img := readPNG()

	// get size and amount of pixel
	bounds := img.Bounds()
	w, h := bounds.Max.X, bounds.Max.Y

	// ------------------------------
	// ALLOCATE SPACE FOR THE 8-BIT IMAGE DATA
	// ------------------------------
	image := allocateImageArrayMemory(img, w, h)

	// ------------------------------
	// CONVERT EVERYTHING TO byte (8-BIT)
	// ------------------------------
	convertTo8Bit(img, w, h, &image)

	// ------------------------------
	// INTERPOLATE DATA
	// ------------------------------
	results := interpolateImage(image, accuracy)

	// ------------------------------
	// WRITE TO FILE
	// ------------------------------
	writeToFile(results, w, h)
}

func readPNG() image.Image {
	fmt.Print("OPEN PNG FILE...")
	// open png file
	infile, err := os.Open(os.Args[1])
	if err != nil {
		Panic("Parsing argument error: %s\n", err.Error())
	}
	defer infile.Close()
	// get image object from file
	img, err := png.Decode(infile)
	if err != nil {
		Panic("Loading png file error: %s\n", err.Error())
	}
	fmt.Println("DONE")

	return img
}

func convertTo8Bit(img image.Image, w, h int, image *imageRGBA) {

	fmt.Print("CONVERT TO 8-BIT...")
	t1 := time.Now()
	// convert data from uint32 into bytes for smaller images
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			r, g, b, a := img.At(x, y).RGBA()
			r /= 256
			g /= 256
			b /= 256
			a /= 256
			if r > 255 ||
				g > 255 ||
				b > 255 ||
				a > 255 {

				Panic("Invalid amount of bits! Only 8-bit per channel are allowed (yet). The value was %d,%d,%d,%d.\n", r, g, b, a)
			}
			//			fmt.Printf("OK for %d and %d with values %d, %d, %d, %d\n", y, x, r, g, b, a)
			//			image2dR[y][x], image2dG[y][x], image2dB[y][x], image2dA[y][x] = byte(r), byte(g), byte(b), byte(a)
			(*image).R[y][x] = byte(r)
			(*image).G[y][x] = byte(g)
			(*image).B[y][x] = byte(b)
			(*image).A[y][x] = byte(a)
		}
	}
	duration := time.Since(t1)
	fmt.Printf("DONE (%d ms)\n", int(float32(duration.Nanoseconds())/1000000.0))
}

func interpolateImage(image imageRGBA, accuracy float32) []byte {

	fmt.Print("INTERPOLATE...")
	t1 := time.Now()

	// calc interpolation (final image data)
	results := interpolateChannel(image, accuracy)

	duration := time.Since(t1)
	fmt.Printf("DONE (%d ms)\n", int(float32(duration.Nanoseconds())/1000000.0))

	return results
}

func interpolateChannel(image imageRGBA, accuracy float32) []byte {
	amountRows := len(image.A)
	width := len(image.A[0])
	output := make([]byte, 0) // at least [vR, vG, vB, Offset, vR, vG, vB] due to definition
	for currentRow := 0; currentRow < amountRows; currentRow++ {
		index := 0 // like x-coordinate

		output = append(output, image.R[currentRow][0], image.G[currentRow][0], image.B[currentRow][0])

		offsetSum := 0

		for ; index < width; index++ {
			_, offset, value2 := findPoints(&image, currentRow, &index, accuracy)
			offsetSum += int(offset)
			output = append(output, offset, value2.R, value2.G, value2.B)
		}

		// FIXME sometimes the sum of all offsets if not width-1. Resolve this hack please!
		if offsetSum != width-1 {
			output[len(output)-4] = output[len(output)-4] + byte(width-1-offsetSum)
			offsetSum += width - 1 - offsetSum
			//			fmt.Println(offsetSum, ",", byte(width-1-offsetSum), "\n")
		}
	}

	return output
}

func findPoints(image *imageRGBA, currentRow int, index *int, deviation float32) (RGBAValue, byte, RGBAValue) {
	sumR := 0
	sumG := 0
	sumB := 0

	amount := 1
	data := *image
	width := len(data.A[0]) - 1

	// initialize return values
	value1 := RGBAValue{
		R: data.R[currentRow][*index],
		G: data.G[currentRow][*index],
		B: data.B[currentRow][*index],
	}
	value2 := value1
	offset := byte(0)

	for ; *index < width && amount < 256; *index++ {
		value2 = RGBAValue{
			R: data.R[currentRow][*index+1],
			G: data.G[currentRow][*index+1],
			B: data.B[currentRow][*index+1],
		}
		sumR += int(value2.R)
		sumG += int(value2.G)
		sumB += int(value2.B)

		dR := calcDeviation(float32(sumR), float32(amount), float32(value1.R), float32(value2.R))
		dG := calcDeviation(float32(sumG), float32(amount), float32(value1.G), float32(value2.G))
		dB := calcDeviation(float32(sumB), float32(amount), float32(value1.B), float32(value2.B))

		if dR > deviation ||
			dG > deviation ||
			dB > deviation {
			*index--
			break
		}

		value2 = RGBAValue{
			R: data.R[currentRow][*index],
			G: data.G[currentRow][*index],
			B: data.B[currentRow][*index],
		}
		amount++

	}
	offset = byte(amount - 1)

	return value1, offset, value2
}

func calcDeviation(actualSum, amount, value1, value2 float32) float32 {
	factor := (value2 - value1) / amount
	if factor == 0 {
		return float32(0)
	}
	// difference between two gauss formulas, multiplicated with the "frequency" so the difference between to ideal values
	// this gives us the expected sum between the two values (in other words: if you'd interpolate between value1 and value2 and sum it up, this would be your result)
	v1 := value1 / factor
	v2 := value2 / factor
	expectedSum := (((v2*v2 + v2) / 2) - ((v1*v1 + v1) / 2)) * factor
	//	fmt.Printf("Value are: %f, %f, %f, %f, %f\n", actualSum, v1, v2, factor, expectedSum)

	return float32(math.Abs(float64(actualSum-expectedSum))) / expectedSum * 100.0
}

func writeToFile(results []byte, w, h int) {

	fmt.Print("WRITING...")
	t1 := time.Now()

	f, err := os.Create("./output.ipf") // Interpolated Picture File
	if err != nil {
		Panic("Writing to file failed! Error: %s.\n", err.Error())
	}

	a := make([]byte, 4)
	binary.BigEndian.PutUint32(a, uint32(w))
	f.Write(a)
	binary.BigEndian.PutUint32(a, uint32(h))
	f.Write(a)

	f.Write(results)
	fmt.Println(len(results))

	duration := time.Since(t1)
	fmt.Printf("DONE (%d ms)\n", int(float32(duration.Nanoseconds())/1000000.0))
}
