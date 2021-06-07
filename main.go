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
	"strings"
	"text/scanner"
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
	flagEval = flag.String("expr", "1/(1+(z*z))", "expression to be evaluated")
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
	expression string
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
	defaultParam.scaleFactor=*flagScaleFactor
	defaultParam.zscale = float64(defaultParam.height) * defaultParam.scaleFactor
	defaultParam.angle = 2* math.Pi * *flagAngle
	defaultParam.expression = *flagEval
	expr, err := parseAndCheck(defaultParam.expression)
	if err != nil{
		fmt.Println("error, bad expression")
		return
	}
	//      {600 320 100 30 10 128 0 0.5235987755982988 1/(1+(z*z))}
	//      {600 320 100 30 10 0 0 0.5235987755982988 1/(1+(z*z))}
	if *flagAddress == "" {
		fmt.Println(defaultParam)
		writesvg(os.Stdout, &defaultParam, func(z complex128)complex128{return expr.Eval(Env{"z":z})})
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
	expression := q.Get("expr")
	if expression != "" {
		params.expression = expression
	}
	expr, err := parseAndCheck(params.expression)
	if err != nil{
		fmt.Fprintln(w, "error, bad expression")
		return
	}
	params.xyscale= float64(params.width)/2.0/params.xyrange
	params.zscale = float64(params.height) * params.scaleFactor
	w.Header().Set("Content-Type", "image/svg+xml")
	fmt.Println(params)
	writesvg(w, &params, func(z complex128)complex128{return expr.Eval(Env{"z":z})})
}

func corner(f func(complex128)complex128,i, j int, p *params)(float64, float64){
	x := p.xyrange * (float64(i)/float64(p.cells)-0.5)
	y := p.xyrange * (float64(j)/float64(p.cells) - 0.5)
	cnum := complex(x,y)
	n := f(cnum)
	//n:= f(cnum)
	z := cmplx.Abs(n)

	sx := float64(p.width)/2+(x-y)*math.Cos(p.angle)*p.xyscale
	sy:=float64(p.height)/2+(x+y)*math.Sin(p.angle)*p.xyscale -z*p.zscale
	return sx, sy
}

func f(cnum complex128) complex128{
	z := 1/(1+(cnum*cnum))
	return z
}

func writesvg(w io.Writer, p *params, f func(z complex128) complex128){
	fmt.Fprintf(w, "<svg xmlns='http://www.w3.org/2000/svg' "+
		"style='stroke: grey; fill:white ; stroke-width: 0.7' "+
		"width='%d' height='%d'>", p.width, p.height)
	for i := 0; i< p.cells;i++{
		for j := 0; j< p.cells;j++{
			ax, ay := corner(f,i+1, j, p)
			bx, by := corner(f,i,j, p)
			cx,cy := corner(f,i,j+1, p)
			dx, dy := corner(f,i+1,j+1, p)
			fmt.Fprintf(w, "<polygon points ='%g,%g %g,%g %g,%g %g,%g'/>\n",
				ax, ay, bx ,by ,cx,cy,dx,dy)
		}
	}
	fmt.Fprintln(w, "</svg>")
}

type Expr interface{
	Eval(env Env) complex128
	Check(vars map[Var]bool) error
}

type Var string

type literal complex128

type unary struct {
	op rune
	z Expr
}

type binary struct {
	op rune
	z,w Expr
}

type call struct{
	fn string
	args []Expr
}

type Env map[Var]complex128

func (v Var) Eval(env Env) complex128{
	return env[v]
}

func (l literal) Eval(_ Env) complex128{
	return complex128(l)
}

func (u unary) Eval(env Env) complex128{
	switch u.op {
	case '+':
		return +u.z.Eval(env)
	case '-':
		return -u.z.Eval(env)
	}
	panic(fmt.Sprintf("unsupported unary operator: %q", u.op))
}

func (b binary) Eval(env Env) complex128{
	switch b.op {
	case '+':
		return b.z.Eval(env) + b.w.Eval(env)
	case '-':
		return b.z.Eval(env) - b.w.Eval(env)
	case '*':
		return b.z.Eval(env) * b.w.Eval(env)
	case '/':
		return b.z.Eval(env) / b.w.Eval(env)
	}
	panic(fmt.Sprintf("unsupported binary operator: %q", b.op))
}

func (c call) Eval(env Env) complex128{
	switch c.fn{
	case "pow":
		return cmplx.Pow(c.args[0].Eval(env), c.args[1].Eval(env))
	case "sin":
		return cmplx.Sin(c.args[0].Eval(env))
	case "cos":
		return cmplx.Cos(c.args[0].Eval(env))
	case "sqrt":
		return cmplx.Sqrt(c.args[0].Eval(env))
	case "exp":
		return cmplx.Exp(c.args[0].Eval(env))
	case "Log":
		return cmplx.Log(c.args[0].Eval(env))
	}
	panic(fmt.Sprintf("unsupported function call: %q", c.fn))
}

func (v Var) Check(vars map[Var]bool) error {
	vars[v] = true
	return nil
}

func (literal) Check(vars map[Var]bool) error {
	return nil
}

func (u unary) Check(vars map[Var]bool) error{
	if !strings.ContainsRune("+-", u.op){
		return fmt.Errorf("unexpected unary op %q", u.op)
	}
	return u.z.Check(vars)
}

func (b binary) Check(vars map[Var]bool) error {
	if !strings.ContainsRune("+-*/", b.op){
		return fmt.Errorf("unexpected binary op %q", b.op)
	}
	if err := b.z.Check(vars); err != nil{
		return err
	}
	return b.w.Check(vars)
}

var numParams = map[string]int{"pow":2,"sin":1,"sqrt":1,"exp":1,"Log":1}

func (c call) Check(vars map[Var]bool) error{
	arity, ok := numParams[c.fn]
	if !ok {
		return fmt.Errorf("unkown function %q", c.fn)
	}
	if len(c.args) != arity{
		return fmt.Errorf("call to %q has %d args, want %d", c.fn, len(c.args), arity)
	}
	for _, arg := range c.args {
		if err := arg.Check(vars); err != nil{
			return err
		}
	}
	return nil
}

func parseAndCheck(s string) (Expr, error){
	if s == ""{
		return nil, fmt.Errorf("empty expression")
	}
	expr, err := Parse(s)
	if err != nil{
		return nil, err
	}
	vars := make(map[Var]bool)
	if err := expr.Check(vars); err != nil{
		return nil, err
	}
	if len(vars) > 1 {
		return nil, fmt.Errorf("too many variables")
	}
	for v := range vars {
		if v != "z"{
			return nil, fmt.Errorf("undefined variable: %s", v)
		}
	}
	return expr, nil
}

type lexer struct {
	scan  scanner.Scanner
	token rune // current lookahead token
}

func (lex *lexer) next()        { lex.token = lex.scan.Scan() }
func (lex *lexer) text() string { return lex.scan.TokenText() }

type lexPanic string

// describe returns a string describing the current token, for use in errors.
func (lex *lexer) describe() string {
	switch lex.token {
	case scanner.EOF:
		return "end of file"
	case scanner.Ident:
		return fmt.Sprintf("identifier %s", lex.text())
	case scanner.Int, scanner.Float:
		return fmt.Sprintf("number %s", lex.text())
	}
	return fmt.Sprintf("%q", rune(lex.token)) // any other rune
}

func precedence(op rune) int {
	switch op {
	case '*', '/':
		return 2
	case '+', '-':
		return 1
	}
	return 0
}

func Parse(input string) (_ Expr, err error) {
	defer func() {
		switch x := recover().(type) {
		case nil:
		case lexPanic:
			err = fmt.Errorf("%s", x)
		default:
			panic(x)
		}
	}()
	lex := new(lexer)
	lex.scan.Init(strings.NewReader(input))
	lex.scan.Mode = scanner.ScanIdents | scanner.ScanInts | scanner.ScanFloats
	lex.next()
	e := parseExpr(lex)
	if lex.token != scanner.EOF {
		return nil, fmt.Errorf("unexpected %s", lex.describe())
	}
	return e, nil
}

func parseExpr(lex *lexer) Expr { return parseBinary(lex, 1) }

func parseBinary(lex *lexer, prec1 int) Expr {
	lhs := parseUnary(lex)
	for prec := precedence(lex.token); prec >= prec1; prec-- {
		for precedence(lex.token) == prec {
			op := lex.token
			lex.next() // consume operator
			rhs := parseBinary(lex, prec+1)
			lhs = binary{op, lhs, rhs}
		}
	}
	return lhs
}

func parseUnary(lex *lexer) Expr {
	if lex.token == '+' || lex.token == '-' {
		op := lex.token
		lex.next() // consume '+' or '-'
		return unary{op, parseUnary(lex)}
	}
	return parsePrimary(lex)
}

func parsePrimary(lex *lexer) Expr {
	switch lex.token {
	case scanner.Ident:
		id := lex.text()
		lex.next()
		if lex.token != '(' {
			return Var(id)
		}
		lex.next()
		var args []Expr
		if lex.token != ')' {
			for {
				args = append(args, parseExpr(lex))
				if lex.token != ',' {
					break
				}
				lex.next()
			}
			if lex.token != ')' {
				msg := fmt.Sprintf("got %s, want ')'", lex.describe())
				panic(lexPanic(msg))
			}
		}
		lex.next()
		return call{id, args}

	case scanner.Int, scanner.Float:
		f, err := strconv.ParseComplex(lex.text(), 64)
		if err != nil {
			panic(lexPanic(err.Error()))
		}
		lex.next()
		return literal(f)

	case '(':
		lex.next()
		e := parseExpr(lex)
		if lex.token != ')' {
			msg := fmt.Sprintf("got %s, want ')'", lex.describe())
			panic(lexPanic(msg))
		}
		lex.next()
		return e
	}
	msg := fmt.Sprintf("unexpected %s", lex.describe())
	panic(lexPanic(msg))
}