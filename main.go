package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"math/cmplx"
	"net/http"
	"os"
	"strconv"
)

const help = `
Generates the graph of the complex function 1/(1 + 𝑧²), and writes an svg file to std out`

var (
	flagHelp    = flag.Bool("help", false, "print usage and help, and exit")
	flagAddress = flag.String("a", "", "address on which to listen")
	flagWidth = flag.Int("w", 600, "width")
	flagHeight = flag.Int("h", 320, "height")
	flagXYRange = flag.Float64("r", 30.0, "range for x, y")
	flagCells = flag.Int("c", 100, "number of cells")
	flagScaleFactor = flag.Float64("s", 0.4, "scale factor")
	flagAngle = flag.Float64("angle", 1.0/12.0, "fraction of a circle to rotate by")
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage of complex-analysis:\n")
	fmt.Fprintf(os.Stderr, "\tcomplex-analysis [-a address] [-w width] [-h height] [-r range] [-c cells] [-s scale-factor] [-a angle]\n")
	if *flagHelp {
		fmt.Fprintln(os.Stderr, help)
	}
	fmt.Fprintf(os.Stderr, "Flags:\n")
	flag.PrintDefaults()
}

var defaultParam params

type params struct{
	width, height, cells int
	xyrange, xyscale, zscale, scaleFactor, angle float64
}

func main(){
	log.SetPrefix("px: ")
	log.SetFlags(0)

	flag.Usage = usage
	flag.Parse()

	if *flagHelp {
		flag.Usage()
		os.Exit(2)
	}
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "does not take any operands")
		flag.Usage()
		os.Exit(2)
	}
	defaultParam.width = *flagWidth
	defaultParam.height = *flagHeight
	defaultParam.cells = *flagCells
	defaultParam.xyrange = *flagXYRange
	defaultParam.xyscale= float64(defaultParam.width)/2.0/defaultParam.xyrange
	defaultParam.zscale = float64(defaultParam.height) * *flagScaleFactor
	defaultParam.angle = 2* math.Pi * *flagAngle
	if *flagAddress == "" {
		writesvg(os.Stdout, &defaultParam)
		return
	}
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(*flagAddress, nil))
}

func handler(w http.ResponseWriter, r *http.Request){
	params := defaultParam
	q := r.URL.Query()
	width, err := strconv.Atoi(q.Get("width"))
	if err == nil && width > 0 {
		params.width = width
	}
	height, err := strconv.Atoi(q.Get("height"))
	if err == nil && height > 0 {
		params.height = height
	}
	cells, err := strconv.Atoi(q.Get("cells"))
	if err == nil && cells > 0 {
		params.cells = cells
	}
	scaleFactor, err := strconv.ParseFloat(q.Get("scalefactor"), 64)
	if err == nil && scaleFactor > 0 {
		params.scaleFactor = scaleFactor
	}
	angle, err := strconv.ParseFloat(q.Get("angle"), 64)
	if err == nil && angle > 0 {
		params.angle = angle
	}
	params.xyscale= float64(params.width)/2.0/params.xyrange
	params.zscale = float64(params.height) * params.scaleFactor
	w.Header().Set("Content-Type", "image/svg+xml")
	writesvg(w, &params)
}

func corner(i, j int, p *params)(float64, float64){
	x := p.xyrange * (float64(i)/float64(p.cells)-0.5)
	y := p.xyrange * (float64(j)/float64(p.cells) - 0.5)
	cnum := complex(x,y)
	z:= f(cnum)
	sx := float64(p.width)/2+(x-y)*math.Cos(p.angle)*p.xyscale
	sy:=float64(p.height)/2+(x+y)*math.Sin(p.angle)*p.xyscale -z*p.zscale
	return sx, sy
}

func f(cnum complex128) float64{
	z := 1/(1+(cnum*cnum))
	return cmplx.Abs(z)
}

func writesvg(w io.Writer, p *params){
	fmt.Fprintf(w, "<svg xmlns='http://www.w3.org/2000/svg' "+
		"style='stroke: grey; fill:white ; stroke-width: 0.7' "+
		"width='%d' height='%d'>", p.width, p.height)
	for i := 0; i< p.cells;i++{
		for j := 0; j< p.cells;j++{
			ax, ay := corner(i+1, j, p)
			bx, by := corner(i,j, p)
			cx,cy := corner(i,j+1, p)
			dx, dy := corner(i+1,j+1, p)
			fmt.Fprintf(w, "<polygon points ='%g,%g %g,%g %g,%g %g,%g'/>\n",
				ax, ay, bx ,by ,cx,cy,dx,dy)
		}
	}
	fmt.Fprintln(w, "</svg>")
}