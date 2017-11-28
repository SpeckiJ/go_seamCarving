package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"log"
	"math"
	"os"
	"strconv"
	"time"
)

// Tracks the time elapsed since start.
func Timetrack(start time.Time, name string) {
	elapsed := time.Since(start)
	fmt.Printf(" ... %s took %s ... \n", name, elapsed)
}

func main() {
	defer Timetrack(time.Now(), "seamcarving")
	args := os.Args[1:]
	if len(args) != 4 {
		log.Fatal("Invalid arguments supplied. Please supply 4 arguments: <inputfile> <outputfile> xsize ysize")
	}
	fmt.Println(" ... Starting seamcarving ... ")
	fmt.Println("(this may take a while - 0.2sec per pixel Difference)")
	// Read Input Image - errors handled in func
	img := readImage(args[0])

	// Get old image size
	xsizeold := img.Bounds().Max.X
	ysizeold := img.Bounds().Max.Y

	// Get new image size
	xsizenew, err1 := strconv.Atoi(args[2])
	ysizenew, err2 := strconv.Atoi(args[3])
	if err1 != nil || err2 != nil {
		log.Fatal("Failed to parse xsize or ysize: " + err1.Error() + "   " + err2.Error())
	}
	seamcounterX := xsizeold - xsizenew
	seamcounterY := ysizeold - ysizenew
	if seamcounterX < 0 {
		log.Fatal("New xsize is bigger than old one. Please supply valid sizes")
	}
	if seamcounterY < 0 {
		log.Fatal("New ysize is bigger than old one. Please supply valid sizes")
	}
	// calculate smaller counter to alternate between horizontal and vertical Seam removal
	minseamCounter := seamcounterY
	if seamcounterX <= seamcounterY {
		minseamCounter = seamcounterX
	}

	// Seams are removed alternating between horizontal and vertical
	for i := 0; i < minseamCounter*2; i++ {
		if i%2 == 0 {
			img = removeSeamH(img.(draw.Image))
			seamcounterY--
		} else {
			img = removeSeamV(img.(draw.Image))
			seamcounterX--
		}
	}

	// remove leftover Seams when alternating is not possible
	for i := 0; i < seamcounterX; i++ {
		img = removeSeamV(img.(draw.Image))
	}
	for i := 0; i < seamcounterY; i++ {
		img = removeSeamH(img.(draw.Image))
	}

	// Writing Output to file
	imgfile, err := os.Create(args[1])
	if err != nil {
		log.Fatal("Error creating output file: " + err.Error())
	}
	defer imgfile.Close()
	png.Encode(imgfile, img)
}

// removeSeamH removes the horizontal seam with the lowest Energy from the Image. It returns the new Image
func removeSeamH(img draw.Image) image.Image {
	// Transpose image and delete vertical Seam
	newimg := removeSeamV(transposeImage(img))
	// Transpose back
	return transposeImage(newimg.(draw.Image))
}

// removeSeamV removes the vertical seam with the lowest Energy from the Image. It returns the new Image
func removeSeamV(img draw.Image) image.Image {
	// Create energy matrix using sobel operator
	energy := energy(img)
	// calculate cumulative mininmal energy matrix
	cumMinEnergy := getCumMinEnergy(energy)
	// get least energy seam
	seam := getSeamV(cumMinEnergy)

	// new image size
	xsize := img.Bounds().Max.X - 1
	ysize := img.Bounds().Max.Y

	// create new image
	newbounds := image.Rect(0, 0, xsize, ysize)
	newimg := image.NewRGBA(newbounds)

	// set pixels in new image
	for r := 0; r < ysize; r++ {
		newx := 0
		for oldx := 0; oldx < xsize+1; oldx++ {
			newimg.Set(newx, r, img.At(oldx, r))
			if oldx != seam[r] {
				newx++
			}
		}
	}
	return newimg
}

// getCumMinEnergy returns the cumulative minimal energy matrix from a given energy matrix
func getCumMinEnergy(energy [][]float64) [][]float64 {
	xsize := len(energy[0])
	ysize := len(energy)

	// Create Seams Array
	cumMinEnergy := make([][]float64, ysize)
	for i := range cumMinEnergy {
		cumMinEnergy[i] = make([]float64, xsize)
	}

	// calculate cumulative minimum Energy per pixel
	for r := 1; r < ysize; r++ {
		// left border
		cumMinEnergy[r][1] = energy[r][1] + math.Min(cumMinEnergy[r-1][1], cumMinEnergy[r-1][2])
		// right border
		cumMinEnergy[r][xsize-2] = energy[r][xsize-2] + math.Min(cumMinEnergy[r-1][xsize-2], cumMinEnergy[r-1][xsize-3])
		// in between
		for c := 2; c < xsize-2; c++ {
			cumMinEnergy[r][c] = energy[r][c] + math.Min(cumMinEnergy[r-1][c-1], math.Min(cumMinEnergy[r-1][c], cumMinEnergy[r-1][c+1]))
		}
	}
	return cumMinEnergy
}

// getSeamV gets lowest energy seam from cumulative minimal energy matrix. seam is returned as slice of x-indices
func getSeamV(cumMinEnergy [][]float64) []int {
	ysize := len(cumMinEnergy)
	xsize := len(cumMinEnergy[0])

	// Reverse Path with smallest Seam
	// Get End of smallest Seam
	minIndex := 1
	for i := 1; i < len(cumMinEnergy[0])-3; i++ {
		if cumMinEnergy[ysize-1][i] <= cumMinEnergy[ysize-1][minIndex] {
			minIndex = i
		}
	}

	// Create Seam Array to store smallest seam
	seam := make([]int, ysize)
	seam[ysize-1] = minIndex

	// Traverse upwards from End and note all x-indexes of seam
	for i := ysize - 2; i > 0; i-- {
		// left border
		if minIndex == 1 {
			if cumMinEnergy[i][minIndex] < cumMinEnergy[i][minIndex+1] {
				// go up
				seam[i] = minIndex
			} else {
				// go right
				minIndex++
				seam[i] = minIndex
			}
		} else
		// right border
		if minIndex == xsize-1 {
			if cumMinEnergy[i][minIndex] < cumMinEnergy[i][minIndex-1] {
				// go up
				seam[i] = minIndex
			} else {
				// go left
				minIndex--
				seam[i] = minIndex
			}
		} else

		// inbetween borders
		if cumMinEnergy[i][minIndex-1] < cumMinEnergy[i][minIndex] && cumMinEnergy[i][minIndex-1] < cumMinEnergy[i][minIndex+1] {
			// go left
			minIndex--
			seam[i] = minIndex
		} else if cumMinEnergy[i][minIndex+1] < cumMinEnergy[i][minIndex] && cumMinEnergy[i][minIndex+1] < cumMinEnergy[i][minIndex-1] {
			// go right
			minIndex++
			seam[i] = minIndex
		} else {
			// go center
			seam[i] = minIndex
		}
	}
	return seam
}

// energy converts an image into a energy matrix using the sobel operator as derivative function.
// RGB colors are converted into single value by simple addition of all channels.
func energy(img image.Image) [][]float64 {
	imgXsize := img.Bounds().Max.X
	imgYsize := img.Bounds().Max.Y

	// Create Energy Array
	energy := make([][]float64, imgYsize)
	for i := range energy {
		energy[i] = make([]float64, imgXsize)
	}

	// Iterate through whole image
	for c := 0; c < imgXsize; c++ {
		for r := 0; r < imgYsize; r++ {
			// check for border cases
			if c == 0 || r == 0 || c == imgXsize-1 || r == imgYsize-1 {
				// Pixels at edge of image are mirrored - due to symmetry in sobel-operator value is always 0
				energy[r][c] = 0
			} else {
				dx := sobelX(
					img.At(c-1, r-1),
					img.At(c-1, r),
					img.At(c-1, r+1),
					img.At(c+1, r-1),
					img.At(c+1, r),
					img.At(c+1, r+1))
				dy := sobelY(
					img.At(c-1, r-1),
					img.At(c, r-1),
					img.At(c+1, r-1),
					img.At(c-1, r+1),
					img.At(c, r+1),
					img.At(c+1, r+1))
				energy[r][c] = dx + dy
			}
		}
	}
	return energy
}

// sobelY returns the result of the sobel operator for 6 given input pixels
// RGB channels from input pixels are compressed into single value by addition
func sobelY(s1, s2, s3, s4, s5, s6 color.Color) (output float64) {
	topleft := colorSlice(s1)
	topmiddle := colorSlice(s2)
	topright := colorSlice(s3)
	bottomleft := colorSlice(s4)
	bottommiddle := colorSlice(s5)
	bottomright := colorSlice(s6)

	for i := 0; i < 4; i++ {
		output += math.Abs(topleft[i] + 2*topmiddle[i] + topright[i] - bottomleft[i] - 2*bottommiddle[i] - bottomright[i])
	}
	return output
}

// sobelX returns the result of the sobel operator for 6 given input pixels
// RGB channels from input pixels are compressed into single value by addition
func sobelX(s1, s2, s3, s4, s5, s6 color.Color) (output float64) {
	lefttop := colorSlice(s1)
	leftmiddle := colorSlice(s2)
	leftbottom := colorSlice(s3)
	righttop := colorSlice(s4)
	rightmiddle := colorSlice(s5)
	rightbottom := colorSlice(s6)

	for i := 0; i < 4; i++ {
		output += math.Abs(lefttop[i] + 2*leftmiddle[i] + leftbottom[i] - righttop[i] - 2*rightmiddle[i] - rightbottom[i])
	}
	return output
}

// colorSlice creates slice with float64 rgba values from color
func colorSlice(color color.Color) []float64 {
	output := make([]float64, 4)
	r, g, b, a := color.RGBA()
	output[0] = float64(r)
	output[1] = float64(g)
	output[2] = float64(b)
	output[3] = float64(a)
	return output
}

// readImage reads Image from Filename
// terminates the program if loading does not succeed
func readImage(filename string) image.Image {
	// Read Input Image
	reader, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer reader.Close()
	img, _, err := image.Decode(reader)
	if err != nil {
		log.Fatal(err)
	}
	return img
}

// transposeImage diagonally mirrors the Image and returns the new image
func transposeImage(img draw.Image) draw.Image {
	xsize := img.Bounds().Max.X
	ysize := img.Bounds().Max.Y
	// new image with switched width/height
	newimg := image.NewRGBA(image.Rect(0, 0, ysize, xsize))

	for c := 0; c < ysize; c++ {
		for r := 0; r < xsize; r++ {
			newimg.Set(c, r, img.At(r, c))
		}
	}
	return newimg
}
