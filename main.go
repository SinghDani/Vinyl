package main

import (
	"fmt"
	"math"
	"math/cmplx"
)

const PI float64 = math.Pi
const N = 16

func main() {
	samples := make([]float64, N)
	frequencies := make([]complex128, N)

	//generates samples
	for i := 0; i < N; i++ {
		t := float64(i) / N
		y := math.Cos(2*PI*1*t) + math.Cos(2*PI*2*t) + math.Sin(2*PI*1*t)
		samples[i] = y
		fmt.Printf("t: %.8f | val %.8f \n", t, y)
	}

	for f := 0; f < N; f++ {
		for s := 0; s < N; s++ {
			t := float64(s) / N
			e := cmplx.Exp(complex(0, 2*PI*float64(f)*t))
			frequencies[f] += e * complex(samples[s], 0)
		}
	}

	fmt.Println("sin:")
	for _, f := range frequencies {
		fmt.Printf("sin part: %.5f\n", imag(f))
	}

	fmt.Println("cos")
	for _, f := range frequencies {
		fmt.Printf("cos part: %.5f\n", real(f))
	}
}
