package main

import (
	"fmt"
	"image"
	_ "image/png"
	"log"
	"net/http"
	"os"
)

const errorPage = `
<html>
  <body>
	  <p>Invalid uploaded file 🙀</p>
	  <p>Go back to <a href="/">homepage</a></p>
  </body>
</html>
`

const indexPage = `
<html>
	<body>
		<h1>Your favorite color detection service 👁🤖!</h1>
		<form enctype="multipart/form-data" action="/upload" name="fileupload" method="post">
			<p>Upload an image file!</p>
			<input type="file" id="file" name="file">
			<input type="submit" value="Send file">
		</form>
	</body>
</html>
`

func main() {
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/", indexHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Default to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
	log.Fatal(err)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("indexHandler", r.Method, r.URL.Path)

	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	fmt.Fprintln(w, indexPage)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("uploadHandler", r.Method, r.URL.Path)

	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	file, _, err := r.FormFile("file")
	if err == http.ErrMissingFile {
		log.Println("no file uploaded")
		w.Header().Add("Location", "/")
		w.WriteHeader(http.StatusFound)
		return
	}
	if err != nil {
		log.Println("uploaded file: error:", err)
		w.Write([]byte(errorPage))
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		log.Println("cannot decode image: error:", err)
		w.Write([]byte(errorPage))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	fmt.Fprintf(w, `
		<html>
			<body>
				<h1>Thank you for the upload!</h1>
				<p>16-bin histogram:</p>
				<pre>%s</pre>
				<p>Color histogram:</p>
				<p>%s</p>
				<p>Go back to <a href="/">homepage</a></p>
			</body>
		</html>
	`,
		histogram16Bin(img),
		histogramColor(img),
	)
}

func histogram16Bin(img image.Image) string {
	// A color's RGBA method returns values in the range [0, 65535].
	// Shifting by 12 reduces this to the range [0, 15].
	max := 65535
	var shift uint = 12
	entries := (max >> shift) + 1

	bounds := img.Bounds()

	var histogram = make([][4]int, entries)
	var zeroes [4]int
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			// Count values at 0 in separate slice.
			if r == 0 {
				zeroes[0]++
			} else {
				histogram[r>>12][0]++
			}
			if g == 0 {
				zeroes[1]++
			} else {
				histogram[g>>12][1]++
			}
			if b == 0 {
				zeroes[2]++
			} else {
				histogram[b>>12][2]++
			}
			if a == 0 {
				zeroes[3]++
			} else {
				histogram[a>>12][3]++
			}
		}
	}

	var tableHistogram string
	tableHistogram = fmt.Sprintf("%-14s %6s %6s %6s %6s\n", "bin", "red", "green", "blue", "alpha")
	for i, x := range histogram {
		from := i << 12
		// We don't have zeroes in our histogram, so our first bin starts at 0x0001 instead of 0x0000
		if i == 0 {
			from = 1
		}
		tableHistogram += fmt.Sprintf("0x%04x-0x%04x: %6d %6d %6d %6d\n",
			from, (i+1)<<12-1, x[0], x[1], x[2], x[3])
	}
	return tableHistogram
}

func histogramColor(img image.Image) string {
	// A color's RGBA method returns values in the range [0, 65535].
	// Shifting by 14 reduces this to the range [0, 3].
	maxColor := 65535
	var shift uint = 14
	bins := (maxColor >> shift) + 1
	mean := (1 << (shift - 1)) / 2

	var histogram = make([][][]int, bins)
	// initialize
	for r := range histogram {
		histogram[r] = make([][]int, bins)
		for g := range histogram[r] {
			histogram[r][g] = make([]int, bins)
			for b := range histogram[r][g] {
				histogram[r][g][b] = 0
			}
		}
	}

	bounds := img.Bounds()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			histogram[r>>shift][g>>shift][b>>shift]++
		}
	}

	var highestCount int
	var mostFrequentR int
	var mostFrequentG int
	var mostFrequentB int

	for r := range histogram {
		for g := range histogram[r] {
			for b := range histogram[r][g] {
				if count := histogram[r][g][b]; count > highestCount {
					mostFrequentR = r
					mostFrequentG = g
					mostFrequentB = b
					highestCount = count
				}
			}
		}
	}

	return fmt.Sprintf(`most frequent (bin, mean-val): r=(%d, %d), g=(%d, %d), b=(%d, %d) => <p style="background: rgb(%d, %d, %d)">main</p>`,
		mostFrequentR, (mostFrequentR<<shift)+mean,
		mostFrequentG, (mostFrequentG<<shift)+mean,
		mostFrequentB, (mostFrequentB<<shift)+mean,
		int((float64((mostFrequentR<<shift)+mean)/float64(maxColor))*255),
		int((float64((mostFrequentG<<shift)+mean)/float64(maxColor))*255),
		int((float64((mostFrequentB<<shift)+mean)/float64(maxColor))*255),
	)
}
