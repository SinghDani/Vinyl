package audioconfig

// Fingerprint settings must match those used to generate the stored fingerprints.
// Rebuild WASM and regenerate stored fingerprints when changing these values.
const (
	SampleRate = 11025
	WindowSize = 1024 // Samples per FFT window.
	HopFactor  = 2
	HopSize    = WindowSize / HopFactor // Samples between consecutive windows.

	// Peak neighbourhood dimensions must be odd to centre them on a bin/window.
	PeakFrequencyBins = 9
	PeakTimeWindows   = 9

	// Target-zone time bounds are measured from the anchor peak.
	TargetStartSeconds       = 0.05
	TargetEndSeconds         = 2
	TargetFrequencyBinsAbove = 100
	TargetFrequencyBinsBelow = 100
	PairsPerAnchor           = 5
)
