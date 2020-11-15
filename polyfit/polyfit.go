// Package polyfit models a polynomial y from sample points xs and ys, to minimize the squared residuals.
//
// E.g., to fit a line `y = β₂x + β₁` with two point (1, 0) and (2, 1):
// The input is `xs = [1, 2], ys = [0, 1], degree=1`.
// The result polynomial is `y = x - 1`, e.g., `β₂ = 1, β₁ = -1`.
//
// This package provides a incremental-fitting API, that let caller to merge two
// set of points efficiently.
//
// See https://en.wikipedia.org/wiki/Least_squares#Linear_least_squares
//
// Since 0.1.0
package polyfit

import (
	"fmt"
	"strings"

	"gonum.org/v1/gonum/mat"
)

// Fitting models a polynomial y from sample points xs and ys, to minimizes the squared residuals.
// It returns coefficients of the polynomial y:
//
//    f(x) = β₁ + β₂x + β₃x² + ...
//
// It use linear regression, which assumes f(x) is in form of:
//           m
//    f(x) = ∑ βⱼ Φⱼ(x),   Φⱼ(x) = xʲ⁻¹
//           j=1
//
// Find β to minimize (f(xᵢ) - yᵢ)²,
// e.g., ||Xβ - Y||² = (Xβ −Y)ᵀ(Xβ −Y) = YᵀY − YᵀXβ − βᵀXᵀY + βᵀXᵀXβ
// where
//
//        | 1 x₁ x₁²... |
//    X = | 1 x₂ x₂²... |
//        | 1 x₃ x₃²... |
//
//    β = [β₁, β₂...]ᵀ
//
//    Y = [y₁, y₂...]ᵀ
//
// Solve for β:
//    ∂||Xβ −Y||²
//    ---------- = −2XᵀY + 2XᵀXβ = 0
//        ∂β
//
// Finally we get:
//    β = (XᵀX)⁻¹XᵀY
//
// See https://en.wikipedia.org/wiki/Least_squares#Linear_least_squares
//
// Since 0.1.0
type Fitting struct {
	N      int
	Degree int

	// cache XᵀX
	xtx []float64
	// cache XᵀY
	xty []float64
}

// NewFitting creates a new polynomial fitting context, with points and the
// degree of the polynomial.
//
// Since 0.1.0
func NewFitting(xs, ys []float64, degree int) *Fitting {

	n := len(xs)

	m := degree + 1

	f := &Fitting{
		N:      0,
		Degree: degree,

		xtx: make([]float64, m*m),
		xty: make([]float64, m),
	}

	for i := 0; i < m*m; i++ {
		f.xtx[i] = 0
	}

	for i := 0; i < m; i++ {
		f.xty[i] = 0
	}

	for i := 0; i < n; i++ {
		f.Add(xs[i], ys[i])
	}

	return f
}

// Copy into a new instance.
//
// Since 0.1.3
func (f *Fitting) Copy() *Fitting {
	b := &Fitting{
		N:      f.N,
		Degree: f.Degree,

		xtx: make([]float64, 0, len(f.xtx)),
		xty: make([]float64, 0, len(f.xty)),
	}

	b.xtx = append(b.xtx, f.xtx...)
	b.xty = append(b.xty, f.xty...)

	return b
}

// Add a point(x, y) into this fitting.
//
// Since 0.1.0
func (f *Fitting) Add(x, y float64) {

	m := f.Degree + 1

	xpows := make([]float64, m)
	v := float64(1)
	for i := 0; i < m; i++ {
		xpows[i] = v
		v *= x
	}

	for i := 0; i < m; i++ {
		for j := 0; j < m; j++ {
			f.xtx[i*m+j] += xpows[i] * xpows[j]
		}
	}

	for i := 0; i < m; i++ {
		f.xty[i] += xpows[i] * y
	}

	f.N++
}

// Merge two sets of sample data.
//
// This can be done because:
//
//    |X₁|ᵀ × |X₁| = X₁ᵀX₁ + X₂ᵀX₂
//    |X₂|    |X₂|
//
// Since 0.1.0
func (f *Fitting) Merge(b *Fitting) {

	if f.Degree != b.Degree {
		panic(fmt.Sprintf("different degree: %d %d", f.Degree, b.Degree))
	}

	f.N += b.N

	m := f.Degree + 1

	for i := 0; i < m; i++ {
		f.xty[i] += b.xty[i]
		for j := 0; j < m; j++ {
			f.xtx[i*m+j] += b.xtx[i*m+j]
		}
	}
}

// Solve the equation and returns coefficients of the result polynomial.
// The number of coefficients is f.Degree + 1.
//
// It tries to reduce degree of the result polynomial. Since there is a
// polynomial of degree n that passes exactly n+1 points.
//
// Since 0.1.0
func (f *Fitting) Solve() []float64 {

	m := f.Degree + 1

	if m <= f.N {
		// quick path
		rst := make([]float64, m)
		if m == 1 {
			solve1(f.xtx, f.xty, rst)
			return rst
		} else if m == 2 {
			solve2(f.xtx, f.xty, rst)
			return rst
		} else if m == 3 {
			solve3(f.xtx, f.xty, rst)
			return rst
		}
	}

	coef := mat.NewDense(m, m, f.xtx)
	right := mat.NewDense(m, 1, f.xty)

	if f.Degree+1 > f.N {

		m = f.N

		coef = coef.Slice(0, m, 0, m).(*mat.Dense)
		right = right.Slice(0, m, 0, 1).(*mat.Dense)
	}

	var beta mat.Dense
	err := beta.Solve(coef, right)

	// Sometimes it returns error about a large condition number, e.g.: matrix
	// singular or near-singular with condition number 1.3240e+16.
	// The β is inaccurate in this case(near sigular matrix) but it does not
	// matteer. The most common case having this error is to fit points less
	// than degree+1, e.g., fit y = ax² + bx + c with only two points, or with
	// several points on a straight line.
	_ = err

	rst := make([]float64, f.Degree+1)
	for i := 0; i < m; i++ {
		rst[i] = beta.At(i, 0)
	}

	for i := m; i < f.Degree+1; i++ {
		rst[i] = 0
	}
	return rst
}

func determinant2(v []float64) float64 {
	a, b, c, d := v[0], v[1], v[2], v[3]
	return a*d - b*c
}

func determinant3(v []float64) float64 {
	a, b, c, d, e, f, g, h, i := v[0], v[1], v[2], v[3], v[4], v[5], v[6], v[7], v[8]
	return a*e*i + b*f*g + c*d*h - c*e*g - b*d*i - a*f*h
}

func solve1(v []float64, y []float64, into []float64) {
	into[0] = y[0] / v[0]
}

func solve2(v []float64, y []float64, into []float64) {

	a, b, c, d := v[0], v[1], v[2], v[3]

	dd := determinant2(v)
	dx1 := y[0]*d - b*y[1]
	dx2 := a*y[1] - y[0]*c

	into[0] = dx1 / dd
	into[1] = dx2 / dd
}

func solve3(v []float64, y []float64, into []float64) {

	a, b, c, d, e, f, g, h, i := v[0], v[1], v[2], v[3], v[4], v[5], v[6], v[7], v[8]

	dd := determinant3(v)
	// a d g
	dx1 := y[0]*e*i + b*f*y[2] + c*y[1]*h - c*e*y[2] - b*y[1]*i - y[0]*f*h
	// b e h
	dx2 := a*y[1]*i + y[0]*f*g + c*d*y[2] - c*y[1]*g - y[0]*d*i - a*f*y[2]
	// c f i
	dx3 := a*e*y[2] + b*y[1]*g + y[0]*d*h - y[0]*e*g - b*d*y[2] - a*y[1]*h

	into[0] = dx1 / dd
	into[1] = dx2 / dd
	into[2] = dx3 / dd
}

// String converts the object into human readable format.
// It includes:
// n: the number of points.
// degree: expected degree of polynomial.
// and two cached matrix XᵀX and XᵀY.
//
// E.g.:
//     n=1 degree=3
//     1.000 1.000 1.000 1.000
//     1.000 1.000 1.000 1.000
//     1.000 1.000 1.000 1.000
//     1.000 1.000 1.000 1.000
//
//     1.000
//     1.000
//     1.000
//     1.000
//
// Since 0.1.0
func (f *Fitting) String() string {

	m := f.Degree + 1
	ss := []string{}

	xtx := f.matrixStrings(f.xtx)

	ss = append(ss, fmt.Sprintf("n=%d degree=%d", f.N, f.Degree))
	ss = append(ss, xtx...)
	ss = append(ss, "")
	for i := 0; i < m; i++ {
		s := fmt.Sprintf("%3.3f", f.xty[i])
		ss = append(ss, s)
	}
	return strings.Join(ss, "\n")
}

func (f *Fitting) matrixStrings(mat []float64) []string {

	m := f.Degree + 1

	ss := []string{}

	for i := 0; i < m; i++ {
		line := []string{}
		for j := 0; j < m; j++ {
			s := fmt.Sprintf("%3.3f", mat[i*m+j])
			line = append(line, s)
		}

		linestr := strings.Join(line, " ")
		ss = append(ss, linestr)
	}

	return ss
}
