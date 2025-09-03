package main

import (
	"errors"
	"log"
	"os"
)

const FrequencyRate = 44100

func main() {
	/*
		//signal generation
		N := 5000
		samples := make([]float64, N)
		for i := 0; i < N; i++ {
			t := float64(i) / FrequencyRate
			signal := math.Cos(2*PI*426*t) + math.Cos(2*PI*200.5*t) + math.Sin(2*PI*100.20*t) + math.Sin(2*PI*20000*t) + math.Sin(2*PI*400*t)
			//signal := math.Cos(2*PI*400*t) + math.Cos(2*PI*200*t) + math.Sin(2*PI*100*t) + math.Sin(2*PI*10*t) + math.Sin(2*PI*400*t)
			samples[i] = signal
		}

		frequencies := generateSpectogram(samples)
		err := spectogramImage(frequencies)
		if err != nil {
			log.Fatal(err)
		}

		/*
			//frequenices
			for i := 0; i < len(frequencies); i++ {
				cur := frequencies[i]
				fmt.Printf("%.16f   |  i:%d 	|| 	sin: %.16f 	|| 	cos: %.16f \n", cmplx.Abs(cur), i, imag(cur), real(cur))
			}
	*/
	dir, err := os.ReadDir("./audioFiles")
	if err != nil {
		log.Fatal(err)
	}
	if len(dir) == 0 {
		log.Fatal(errors.New("no file in dir"))
	}
	file := dir[0].Name()
	err = ParseWav("./audioFiles/" + file)
	if err != nil {
		log.Fatal(err)
	}

}
