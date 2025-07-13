package main

import "math/cmplx"

func dft(samples []float64) []complex128 {
	N := len(samples)
	frequencies := make([]complex128, N)
	for f := 0; f < N; f++ {
		for s := 0; s < N; s++ {
			t := float64(s) / float64(N)
			//(s[s]+0i )* e^(-i2pift)
			e := cmplx.Exp(complex(0, -2*PI*float64(f)*t))
			frequencies[f] += complex(samples[s], 0) * e
		}
	}
	return frequencies
}

// TODO not power of two
func fft(samples []float64, N int) []complex128 {
	if N == 0 {
		return []complex128{}
	}
	if N <= 1 {
		return []complex128{complex(samples[0], 0)}
	}

	t1 := make([]float64, N/2)
	t2 := make([]float64, N/2)
	for i := 0; i < N/2; i++ {
		t1[i] = samples[2*i]
		t2[i] = samples[2*i+1]
	}
	even := fft(t1, N/2)
	odd := fft(t2, N/2)

	res := make([]complex128, N)
	for k := 0; k < N/2; k++ {
		T := odd[k] * cmplx.Exp(complex(0, -2*PI*float64(k)/float64(N)))
		res[k] = even[k] + T
		res[k+N/2] = even[k] - T
	}
	return res
}

func fft2(samples []float64, N int, frequencies []complex128, start, outIndex, stride int) {
	if N <= 1 {
		frequencies[outIndex] = complex(samples[start], 0)
		return
	}

	fft2(samples, N/2, frequencies, start, outIndex, stride*2)
	fft2(samples, N/2, frequencies, start+stride, outIndex+N/2, stride*2)

	for k := 0; k < N/2; k++ {
		T := frequencies[outIndex+N/2+k] * cmplx.Exp(complex(0, -2*PI*float64(k)/float64(N)))
		even := frequencies[outIndex+k]
		frequencies[outIndex+k] = even + T
		frequencies[outIndex+N/2+k] = even - T
	}
}

// 0 1 2 3 4 5 6 7
// 0 2 4 6 | 1 3 5 7
// 0 4 | 2 6 | 1 5 | 3 7
// 0 | 4 | 2 | 6 | 1 | 5 | 3 | 7
