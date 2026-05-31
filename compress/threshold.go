package compress

// ThresholdManager manages the compression threshold.
type ThresholdManager struct {
	threshold int
}

// NewThresholdManager creates a new manager with the given threshold.
// threshold = -1: compression disabled
// threshold = 0: all packets are compressed
// threshold > 0: only packets larger than this value are compressed
func NewThresholdManager(threshold int) *ThresholdManager {
	return &ThresholdManager{threshold: threshold}
}

// Threshold returns the current threshold value.
func (tm *ThresholdManager) Threshold() int {
	return tm.threshold
}

// SetThreshold sets the threshold value.
func (tm *ThresholdManager) SetThreshold(t int) {
	tm.threshold = t
}

// ShouldCompress determines whether the given dataLength should be compressed.
func (tm *ThresholdManager) ShouldCompress(dataLength int) bool {
	if tm.threshold < 0 {
		return false
	}
	return dataLength >= tm.threshold
}

// IsEnabled returns whether compression is active.
func (tm *ThresholdManager) IsEnabled() bool {
	return tm.threshold >= 0
}
