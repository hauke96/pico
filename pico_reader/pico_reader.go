package main

import (
	"encoding/binary"
	"fmt"
	"github.com/mattn/go-gtk/gdk"
	"github.com/mattn/go-gtk/gtk"
	"io/ioutil"
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

func main() {
	os.Args = append(os.Args, "output.ipf")
	// ------------------------------
	// CHECK ARGS
	// ------------------------------
	fmt.Print("CHECKING ARGS...")
	if len(os.Args) < 2 {
		Panic("Please specify the *.ipf file as argument!")
	}
	fmt.Println("DONE")

	// ------------------------------
	// LOAD IPF
	// ------------------------------
	fmt.Print("OPEN IPF FILE...")
	// open ipf file
	rawData, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		Panic("Parsing argument error: %s\n", err.Error())
	}
	fmt.Println("DONE (", len(rawData)/1000, "kb read )")
	fmt.Println(rawData)

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
	fmt.Println(decodedData)

	// ------------------------------
	// CREATE WINDOW
	// ------------------------------
	drawable, gc := initWindow()
	//	drawable.DrawLine(gc, 10, 10, 100, 100)
	drawImage(decodedData, drawable, gc)

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
	window.SetTitle("GTK DrawingArea")
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
	window.SetSizeRequest(450, 400)
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
		B:      make([][]byte, height)}

	// ------------------------------
	// GO THROUGHT ALL ROWS
	// ------------------------------
	i := 8 // The current index in the raw array. The 8 is the amount for the size (width+heigth) thats parsed earlier
	for currentRow := 0; currentRow < height; currentRow++ {
		image.R[currentRow] = make([]byte, width)
		currentCol := 0 // The index in the color array
		decodeRow(rawImageData, &image.R, currentRow, currentCol, width, &i)
	}
	for currentRow := 0; currentRow < height; currentRow++ {
		image.G[currentRow] = make([]byte, width)
		currentCol := 0 // The index in the color array
		decodeRow(rawImageData, &image.G, currentRow, currentCol, width, &i)
	}
	for currentRow := 0; currentRow < height; currentRow++ {
		image.B[currentRow] = make([]byte, width)
		currentCol := 0 // The index in the color array
		decodeRow(rawImageData, &image.B, currentRow, currentCol, width, &i)
	}

	return image
}

func decodeRow(rawImageData []byte, array *[][]byte, currentRow, currentCol, width int, ipointer *int) {
	i := *ipointer
	for currentCol < width {
		// ------------------------------
		// CALC DATA FOR INTERPOLATION
		// ------------------------------
		fmt.Println(i, len(rawImageData))
		rawStart := rawImageData[i] // the start value of the interpolation
		offset := rawImageData[i+1] // the amount of pixel between start and end
		rawEnd := rawImageData[i+2] // the end value of the interpolation
		step := 0.0
		if offset != 0 {
			step = float64(rawEnd-rawStart) / float64(offset) // the difference between two interpolated pixel
		}
		i += 3

		// ------------------------------
		// SET START VALUE
		// ------------------------------
		//			fmt.Println(currentRow, " - ", currentCol, "-", i)
		(*array)[currentRow][currentCol] = rawStart
		currentCol++

		// ------------------------------
		// INTERPOLATE
		// ------------------------------
		for k := 0; byte(k) < offset; k++ {
			(*array)[currentRow][currentCol+k] = rawStart + byte(float64(k)*step)
			//				fmt.Println(rawStart + k*step)
		}
		currentCol += int(offset) - 1

		// ------------------------------
		// SET END VALUE
		// ------------------------------
		(*array)[currentRow][currentCol] = rawEnd
		currentCol++
	}
	*ipointer = i
}

// bytesToInt converts the four bytes to an int.
func bytesToInt(a, b, c, d byte) int {
	mySlice := []byte{a, b, c, d}
	return int(binary.BigEndian.Uint32(mySlice))
}

func drawImage(image image, drawable *gdk.Drawable, gc *gdk.GC) {
	for y := 0; y < image.height; y++ {
		for x := 0; x < image.width; x++ {
			gc.SetRgbFgColor(gdk.NewColorRGB(image.R[y][x], image.G[y][x], image.B[y][x]))
			drawable.DrawPoint(gc, x, y)
		}
	}
}
