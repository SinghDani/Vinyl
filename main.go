package main

import (
	"fmt"
	"math"
	"math/cmplx"
)

const PI = math.Pi
const FrequencyRate = 64

// 43.06
func main() {
	N := 64
	samples := make([]float64, N)
	for i := 0; i < N; i++ {
		t := float64(i) / FrequencyRate
		signal := math.Cos(2*PI*20*t) + math.Sin(2*PI*2*t) + math.Sin(2*PI*4*t)
		samples[i] = signal
	}

	frequencies := fft(samples, len(samples))
	//frequencies := dft(samples)
	//frequencies := make([]complex128, len(samples))
	//fft2(samples, len(samples), frequencies, 0, 0, 1)

	//frequenices
	for i := 0; i < N; i++ {
		fmt.Printf("%.16f\n", cmplx.Abs(frequencies[i]))
	}

	//sin
	/*
		fmt.Println("sin")
		for i := 0; i < N; i++ {
			fmt.Printf(".%.16f\n", imag(frequencies[i]))
		}
	*/

	//cos
	/*
		fmt.Println("cos")
		for i := 0; i < N; i++ {
			fmt.Printf("%.16f\n", real(frequencies[i]))
		}
	*/
}
