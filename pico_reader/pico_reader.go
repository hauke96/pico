package main

import (
	"encoding/binary"
	"fmt"
	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/gtk"
	"io/ioutil"
	"math"
	"os"
	"strings"
	"time"
)

// ------------------------------
//
// ------------------------------

type image struct {
	width, height int
	R             [][]byte
	G             [][]byte
	B             [][]byte
}

const WIN_HEIGHT = 600
const WIN_WIDTH = 800

var IPF_FILE string = "output.ipf"

func main() {
	// ------------------------------
	// CHECK ARGS
	// ------------------------------
	fmt.Print("CHECKING ARGS...")
	if len(os.Args) < 2 {
		fmt.Println("\nWARNING: No File specified! I'll use 'output.ipf' as image file.")
	} else {
		IPF_FILE = os.Args[1]
	}
	fmt.Println("DONE")

	// ------------------------------
	// LOAD IPF
	// ------------------------------
	fmt.Print("OPEN IPF FILE...")
	// open ipf file
	rawData, err := ioutil.ReadFile(IPF_FILE)
	if err != nil {
		Panic("Parsing argument error: %s\n", err.Error())
	}
	fmt.Println("DONE (", len(rawData)/1000, "kb read )")

	// ------------------------------
	// DECODING
	// ------------------------------
	fmt.Print("DECODE DATA...")
	t1 := time.Now()
	// get rgb-image object from file
	decodedData := decode(rawData)
	duration := time.Since(t1)
	fmt.Printf("DONE (%d ms)\n", int(float32(duration.Nanoseconds())/1000000.0))

	// ------------------------------
	// PRINT RESULT
	// ------------------------------
	//	fmt.Println(decodedData)

	// ------------------------------
	// CREATE WINDOW
	// ------------------------------
	drawable, gc := initWindow()
	drawImage(decodedData, drawable, gc, WIN_WIDTH, WIN_HEIGHT)

	// ------------------------------
	// DRAW IMAGE
	// ------------------------------

	gtk.Main()
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

func initWindow() (*gdk.Drawable, *gdk.GC) {
	win := createWindow(&[]string{os.Args[0]})
	pixmap, gc := createDrawingArea(win)
	showWindow(win)

	return pixmap.GetDrawable(), gc
}

func createWindow(args *[]string) *gtk.Window {
	gtk.Init(args)
	window := gtk.NewWindow(gtk.WINDOW_TOPLEVEL)
	window.SetTitle("PICO - Picture Interpolation COmpression - v.0.1 - View " + IPF_FILE)
	window.Connect("destroy", gtk.MainQuit)

	return window
}

func createDrawingArea(window *gtk.Window) (*gdk.Pixmap, *gdk.GC) {
	vbox := gtk.NewVBox(true, 0)
	vbox.SetBorderWidth(5)
	drawingarea := gtk.NewDrawingArea()

	var gdkwin *gdk.Window

	drawable := gtk.NewDrawingArea().GetWindow().GetDrawable()
	gc := gdk.NewGC(drawable)
	pixmap := gdk.NewPixmap(drawable, 1, 1, 24)

	drawingarea.Connect("configure-event", func() {
		fmt.Println("CONFIG...")
		if pixmap != nil {
			pixmap.Unref()
		}
		allocation := drawingarea.GetAllocation()

		newPixmap := gdk.NewPixmap(drawingarea.GetWindow().GetDrawable(), allocation.Width, allocation.Height, 24)
		*pixmap = *newPixmap

		newGC := gdk.NewGC(pixmap.GetDrawable())
		*gc = *newGC

		gc.SetRgbFgColor(gdk.NewColor("white"))
		pixmap.GetDrawable().DrawRectangle(gc, true, 0, 0, -1, -1)
		gc.SetRgbFgColor(gdk.NewColor("black"))
		gc.SetRgbBgColor(gdk.NewColor("white"))
	})

	drawingarea.Connect("expose-event", func() {
		if pixmap == nil {
			return
		}
		if gdkwin == nil {
			gdkwin = drawingarea.GetWindow()
		}
		gdkwin.GetDrawable().DrawDrawable(gc, pixmap.GetDrawable(), 0, 0, 0, 0, -1, -1)
	})

	vbox.Add(drawingarea)
	window.Add(vbox)

	return pixmap, gc
}

func showWindow(window *gtk.Window) {
	window.SetSizeRequest(WIN_WIDTH, WIN_HEIGHT)
	window.ShowAll()
}

func decode(rawImageData []byte) image {
	// ------------------------------
	// GET SIZE OF IMAGE
	// ------------------------------
	width := bytesToInt(rawImageData[0], rawImageData[1], rawImageData[2], rawImageData[3])
	height := bytesToInt(rawImageData[4], rawImageData[5], rawImageData[6], rawImageData[7])
	fmt.Println(width, "x", height)

	// ------------------------------
	// CREATE IMAGE
	// ------------------------------
	image := image{
		width:  width,
		height: height,
		R:      make([][]byte, height),
		G:      make([][]byte, height),
		B:      make([][]byte, height),
	}

	// ------------------------------
	// GO THROUGHT ALL ROWS
	// ------------------------------
	i := 8 // The current index in the raw array. The 8 is the amount for the size (width+heigth) thats parsed earlier
	for currentRow := 0; currentRow < height; currentRow++ {
		image.R[currentRow] = make([]byte, width)
		image.G[currentRow] = make([]byte, width)
		image.B[currentRow] = make([]byte, width)

		decodeRow(rawImageData, &image, currentRow, width, &i)
	}

	return image
}

func decodeRow(rawImageData []byte, image *image, currentRow, width int, ipointer *int) {
	i := *ipointer
	rawStartR := rawImageData[i+0] // the start value of the interpolation
	rawStartG := rawImageData[i+1] // the start value of the interpolation
	rawStartB := rawImageData[i+2] // the start value of the interpolation
	i += 3
	currentCol := 0

	(*image).R[currentRow][currentCol] = rawStartR
	(*image).G[currentRow][currentCol] = rawStartG
	(*image).B[currentRow][currentCol] = rawStartB

	currentCol++

	for currentCol < width {
		// ------------------------------
		// CALC DATA FOR INTERPOLATION
		// ------------------------------
		offset := rawImageData[i]    // the amount of pixel between start and end
		rawEndR := rawImageData[i+1] // the end value of the interpolation
		rawEndG := rawImageData[i+2] // the end value of the interpolation
		rawEndB := rawImageData[i+3] // the end value of the interpolation
		i += 4

		stepR := 0.0
		stepG := 0.0
		stepB := 0.0
		stepR = float64(int(rawEndR)-int(rawStartR)) / (float64(offset) + 2.0) // the difference between two interpolated pixel
		stepG = float64(int(rawEndG)-int(rawStartG)) / (float64(offset) + 2.0) // the difference between two interpolated pixel
		stepB = float64(int(rawEndB)-int(rawStartB)) / (float64(offset) + 2.0) // the difference between two interpolated pixel

		// ------------------------------
		// INTERPOLATE
		// ------------------------------
		for k := 0; byte(k) < offset; k++ {
			//			fmt.Println(currentRow, ",", width, ",", currentCol+k, ",", offset, ",", len((*image).R[0]))
			(*image).R[currentRow][currentCol+k] = getNewValue(k, rawStartR, stepR)

			(*image).G[currentRow][currentCol+k] = getNewValue(k, rawStartG, stepG)

			(*image).B[currentRow][currentCol+k] = getNewValue(k, rawStartB, stepB)

		}

		// ------------------------------
		// SET VALUES
		// ------------------------------
		(*image).R[currentRow][currentCol] = rawStartR
		(*image).G[currentRow][currentCol] = rawStartG
		(*image).B[currentRow][currentCol] = rawStartB

		currentCol += int(offset) + 0

		rawStartR = rawEndR
		rawStartG = rawEndG
		rawStartB = rawEndB

		// ------------------------------
		// SET END VALUE
		// ------------------------------
		//		(*image).R[currentRow][currentCol] = rawEndR
		//		(*image).G[currentRow][currentCol] = rawEndG
		//		(*image).B[currentRow][currentCol] = rawEndB
		//		currentCol++
	}
	*ipointer = i
}

func getNewValue(k int, rawValue byte, step float64) byte {
	value := float64(rawValue) + float64(k)*step
	if value < 0 {
		value = 0
	} else if value > 255 {
		value = 255
	}
	return byte(value)
}

// bytesToInt converts the four bytes to an int.
func bytesToInt(a, b, c, d byte) int {
	mySlice := []byte{a, b, c, d}
	return int(binary.BigEndian.Uint32(mySlice))
}

func drawImage(image image, drawable *gdk.Drawable, gc *gdk.GC, width, height int) {
	scale := 1.0
	for y := 0; y < image.height && y < int(WIN_HEIGHT*scale); y += int(math.Ceil(scale)) {
		for x := 0; x < image.width && x < int(WIN_WIDTH*scale); x += int(math.Ceil(scale)) {
			//			gc.SetRgbFgColor(gdk.NewColorRGB(image.R[y][x], 0, 0))
			//			drawable.DrawRectangle(gc, true, x*10, 0, 10, 10)
			//			gc.SetRgbFgColor(gdk.NewColorRGB(0, image.G[y][x], 0))
			//			drawable.DrawRectangle(gc, true, x*10, 10, 10, 10)
			//			gc.SetRgbFgColor(gdk.NewColorRGB(0, 0, image.B[y][x]))
			//			drawable.DrawRectangle(gc, true, x*10, 20, 10, 10)
			gc.SetRgbFgColor(gdk.NewColorRGB(image.R[y][x], image.G[y][x], image.B[y][x]))
			//			drawable.DrawRectangle(gc, true, x*10, 35, 10, 10)
			drawable.DrawPoint(gc, int(float64(x)/scale), int(float64(y)/scale))
		}
	}
}
