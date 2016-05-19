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

type imageRGBA struct {
	R [][]byte
	G [][]byte
	B [][]byte
	A [][]byte
}

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
	// GET RGBA IMAGE DATA FROM PNG DATA
	// ------------------------------
	image := decodePNGData(img, w, h)

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

func Panic(s string, a ...interface{}) {
	if !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	if !strings.HasPrefix(s, "\n") {
		if !strings.HasPrefix(s, "ERROR: ") {
			s = "ERROR: " + s
		}
		s = "\n" + s
	} else {
		if !strings.HasPrefix(s, "\nERROR: ") {
			s = "\nERROR: " + s
		}
	}

	if len(a) == 0 {
		fmt.Printf(s)
	} else {
		fmt.Printf(s, a)
	}
	os.Exit(1)
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

func decodePNGData(img image.Image, w, h int) imageRGBA {
	fmt.Print("DECODE PNG DATA...")
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

	return imageRGBA{R: image2dR, G: image2dG, B: image2dB, A: image2dA}
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

func interpolateImage(image imageRGBA, accuracy float32) [][]byte {

	fmt.Print("INTERPOLATE...")
	t1 := time.Now()

	results := make([][]byte, 4)

	// calc interpolation (final image data)
	results[0] = interpolateChannel(image.R, accuracy)
	results[1] = interpolateChannel(image.G, accuracy)
	results[2] = interpolateChannel(image.B, accuracy)
	results[3] = interpolateChannel(image.A, accuracy)

	duration := time.Since(t1)
	fmt.Printf("DONE (%d ms)\n", int(float32(duration.Nanoseconds())/1000000.0))

	return results
}

func interpolateChannel(data [][]byte, accuracy float32) []byte {
	amountRows := len(data)
	output := make([]byte, 0) // at least [Value, Offset, Value] due to definition
	for currentRow := 0; currentRow < amountRows; currentRow++ {
		//	currentRow := 0           // like y-coordinate
		index := 0 // like x-coordinate
		width := len(data[0])

		for ; index < width; index++ {
			value1, offset, value2 := findPoints(&data[currentRow], &index, accuracy)
			output = append(output, value1, offset, value2)
		}
	}

	return output
}

func findPoints(dataPointer *[]byte, index *int, deviation float32) (value1, offset, value2 byte) {
	sum := 0
	amount := 1
	data := *dataPointer
	width := len(data) - 1

	// initialize return values
	value1 = data[*index]
	value2 = value1
	offset = byte(0)

	for ; *index < width; *index++ {
		value2 = data[*index+1]
		sum += int(value2)

		d := calcDeviation(float32(sum), float32(amount), float32(value1), float32(value2))
		//		fmt.Printf("Calced deviation: %f\n", d)
		if d > deviation {
			//			fmt.Printf("Values are not precise enough! Calced deviation: %f - Max. deviation: %f\n", d, deviation)
			value2 = data[*index]
			break
		}

		amount++
	}

	offset = byte(amount - 1)

	return
}

func calcDeviation(actualSum, amount, value1, value2 float32) float32 {
	factor := (value2 - value1) / amount
	// difference between two gauss formulas, multiplicated with the "frequency" so the difference between to ideal values
	// this gives us the expected sum between the two values (in other words: if you'd interpolate between value1 and value2 and sum it up, this would be your result)
	v1 := value1 / factor
	v2 := value2 / factor
	expectedSum := (((v2*v2 + v2) / 2) - ((v1*v1 + v1) / 2)) * factor
	//	fmt.Printf("Value are: %f, %f, %f, %f, %f\n", actualSum, v1, v2, factor, expectedSum)

	return float32(math.Abs(float64(actualSum-expectedSum))) / expectedSum * 100.0
}

func writeToFile(results [][]byte, w, h int) {

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

	for i := 0; i < len(results); i++ {
		f.Write(results[i])
	}

	duration := time.Since(t1)
	fmt.Printf("DONE (%d ms)\n", int(float32(duration.Nanoseconds())/1000000.0))
}
