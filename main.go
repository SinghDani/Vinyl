package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"math/cmplx"
	"os"
	"strconv"
)

const PI float64 = math.Pi
const N = 16

func main() {
	samples := make([]float64, N)
	frequencies := make([]complex128, N/2)

	//generates samples
	for i := 0; i < N; i++ {
		t := float64(i) / N
		y := math.Sin(2*PI*1*t) + math.Sin(2*PI*4*t) + math.Cos(2*PI*4*t)
		samples[i] = y
		fmt.Printf("t: %.8f | val %.8f \n", t, y)
	}

	//dft
	for f := 0; f < N/2; f++ {
		for s := 0; s < N; s++ {
			t := float64(s) / N
			e := cmplx.Exp(complex(0, -2*PI*float64(f)*t))
			frequencies[f] += e * complex(samples[s], 0)
		}
	}

	fmt.Println("\nsin:")
	for i, f := range frequencies {
		fmt.Printf("%d: sin part: %.5f\n", i, imag(f))
	}

	fmt.Println("\ncos:")
	for i, f := range frequencies {
		fmt.Printf("%d: cos part: %.5f\n", i, real(f))
	}

	fmt.Println("\nmagnitude")
	for i, f := range frequencies {
		fmt.Printf("%d: magnitude: %.5f\n", i, cmplx.Abs(f))
	}
	writeMagnitudeToFile(frequencies)

}

func writeMagnitudeToFile(frequencies []complex128) {
	file, err := os.Create("audioFiles/frequencies.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	for i, f := range frequencies {
		magnitude := cmplx.Abs(f)
		writer.Write([]string{
			strconv.Itoa(i),
			fmt.Sprintf("%.8f", magnitude),
		})
	}
}
