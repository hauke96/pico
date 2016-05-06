package main

import (
	"encoding/binary"
	"fmt"
	"image/png"
	"math"
	"os"
	"time"
)

func main() {
	fmt.Print("PARSING ARGS...")
	if len(os.Args) < 2 {
		fmt.Printf("Please specify the *.png file as argument!\n")
		return
	}
	fmt.Println("DONE")

	fmt.Print("OPEN PNG FILE...")
	// open png file
	infile, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Printf("Parsing argument error: %s\n", err.Error())
		return
	}
	defer infile.Close()
	fmt.Println("DONE")

	fmt.Print("DECODE DATA...")
	t1 := time.Now()
	// get image object from file
	img, err := png.Decode(infile)
	if err != nil {
		fmt.Printf("Loading png file error: %s\n", err.Error())
		return
	}

	// get size and amount of pixel
	bounds := img.Bounds()
	w, h := bounds.Max.X, bounds.Max.Y

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

	fmt.Print("CONVERT TO 8-BIT...")
	t1 = time.Now()
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

				fmt.Printf("Invalid amount of bits! Only 8-bit per channel are allowed (yet). The value was %d,%d,%d,%d.\n", r, g, b, a)
				return
			}
			//			fmt.Printf("OK for %d and %d with values %d, %d, %d, %d\n", y, x, r, g, b, a)
			image2dR[y][x], image2dG[y][x], image2dB[y][x], image2dA[y][x] = byte(r), byte(g), byte(b), byte(a)
		}
	}
	duration = time.Since(t1)
	fmt.Printf("DONE (%d ms)\n", int(float32(duration.Nanoseconds())/1000000.0))

	fmt.Print("INTERPOLATE...")
	t1 = time.Now()

	results := make([][]byte, 4)

	// calc interpolation (final image data)
	results[0] = interpolate(image2dR)
	results[1] = interpolate(image2dG)
	results[2] = interpolate(image2dB)
	results[3] = interpolate(image2dA)

	duration = time.Since(t1)
	fmt.Printf("DONE (%d ms)\n", int(float32(duration.Nanoseconds())/1000000.0))

	fmt.Print("WRITING...")
	t1 = time.Now()

	f, err := os.Create("./output.ipf")
	if err != nil {
		fmt.Printf("Writing to file failed! Error: %s.\n", err.Error())
		return
	}

	a := make([]byte, 4)
	binary.BigEndian.PutUint32(a, uint32(w))
	f.Write(a)
	binary.BigEndian.PutUint32(a, uint32(h))
	f.Write(a)
	//	ioutil.WriteFile("output.ipf", w_array, os.O_CREATE)
	//	ioutil.WriteFile("output.ipf", h_array, os.O_CREATE)
	for i := 0; i < 4; i++ {
		//		filewriter := ioutil.WriteFile("output.ipf", results[0], os.O_APPEND)
		f.Write(results[i])
	}

	duration = time.Since(t1)
	fmt.Printf("DONE (%d ms)\n", int(float32(duration.Nanoseconds())/1000000.0))

	//	fmt.Println(resultR)
	//	fmt.Println(resultG)
	//	fmt.Println(resultB)
	//	fmt.Println(resultA)

	//	fmt.Println(result) //TODO print result
}

func interpolate(data [][]byte) []byte {
	amountRows := len(data)
	output := make([]byte, 0) // at least [Value, Offset, Value] due to definition
	for currentRow := 0; currentRow < amountRows; currentRow++ {
		//	currentRow := 0           // like y-coordinate
		index := 0 // like x-coordinate
		width := len(data[0])

		for ; index < width; index++ {
			value1, offset, value2 := findPoints(&data[currentRow], &index, 30.0)
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
