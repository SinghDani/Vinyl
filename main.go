package main

import (
	"fmt"
	"math"
	"math/cmplx"
)

const FrequencyRate = 44100

func main() {
	//signal generation
	N := 1024
	samples := make([]float64, N)
	for i := 0; i < N; i++ {
		t := float64(i) / FrequencyRate
		signal := math.Cos(2*PI*426*t) + math.Cos(2*PI*200.5*t) + math.Sin(2*PI*100.20*t) + math.Sin(2*PI*10.56*t) + math.Sin(2*PI*400*t)
		//signal := math.Cos(2*PI*400*t) + math.Cos(2*PI*200*t) + math.Sin(2*PI*100*t) + math.Sin(2*PI*10*t) + math.Sin(2*PI*400*t)
		samples[i] = signal
	}

	frequencies := generateSpectogram(samples)[0]

	//frequenices
	for i := 0; i < N/2; i++ {
		cur := frequencies[i]
		fmt.Printf("%.16f   |  i:%d 	|| 	sin: %.16f 	|| 	cos: %.16f \n", cmplx.Abs(cur), i, imag(cur), real(cur))
	}
}
