package main

import (
	"fmt"
	"math"
	"math/cmplx"
)

const FrequencyRate = 10000

func main() {
	N := 1024
	//signal generation

	samples := make([]float64, N)
	for i := 0; i < N; i++ {
		t := float64(i) / FrequencyRate
		signal := math.Cos(2*PI*426*t) + math.Cos(2*PI*200.5*t) + math.Sin(2*PI*100.20*t) + math.Sin(2*PI*10.56*t)
		samples[i] = signal
	}

	frequencies := generateSpectogram(samples)

	//frequenices
	for i := 0; i < N; i++ {
		cur := frequencies[i]
		fmt.Printf("%.16f 	|| 	sin: %.16f 	|| 	cos: %.16f \n", cmplx.Abs(cur), imag(cur), real(cur))
	}
}
