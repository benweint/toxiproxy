package toxics

import (
	"math/rand"
	"time"
)

// The LatencyToxic passes data through with the a delay of latency +/- jitter added.
type LatencyToxic struct {
	// Times in milliseconds
	Latency     int64 `json:"latency"`
	Jitter      int64 `json:"jitter"`
	OnsetDelay  int64 `json:"onset_delay"`
	OnsetJitter int64 `json:"onset_jitter"`
}

type LatencyToxicState struct {
	connectionStartTime time.Time
	onsetTime           time.Time
}

func (t *LatencyToxic) NewState() interface{} {
	// connectionStartTime := time.Now()
	// onsetTime := connectionStartTime.Add(t.onsetDelay())
	// return &LatencyToxicState{connectionStartTime, onsetTime}
	return new(LatencyToxicState)
}

func (t *LatencyToxic) GetBufferSize() int {
	return 1024
}

func delay(baseDelay int64, jitter int64) time.Duration {
	// Delay = t.Latency +/- t.Jitter
	delay := baseDelay
	if jitter > 0 {
		delay += rand.Int63n(jitter*2) - jitter
	}
	return time.Duration(delay) * time.Millisecond
}

func (t *LatencyToxic) onsetDelay() time.Duration {
	return delay(int64(t.OnsetDelay), int64(t.OnsetJitter))
}

func (t *LatencyToxic) chunkDelay() time.Duration {
	return delay(int64(t.Latency), int64(t.Jitter))
}

func (t *LatencyToxic) Pipe(stub *ToxicStub) {
	state := stub.State.(*LatencyToxicState)
	if state.connectionStartTime.IsZero() {
		state.connectionStartTime = time.Now()
		state.onsetTime = state.connectionStartTime.Add(t.onsetDelay())
	}

	for {
		select {
		case <-stub.Interrupt:
			return
		case c := <-stub.Input:
			if c == nil {
				stub.Close()
				return
			}

			timeSinceChunkArrival := time.Since(c.Timestamp)

			if time.Now().After(state.onsetTime) {
				sleep := t.chunkDelay() - timeSinceChunkArrival
				select {
				case <-time.After(sleep):
					c.Timestamp = c.Timestamp.Add(sleep)
					stub.Output <- c
				case <-stub.Interrupt:
					// Exit fast without applying latency.
					stub.Output <- c // Don't drop any data on the floor
					return
				}
			} else {
				stub.Output <- c
			}
		}
	}
}

func init() {
	Register("latency", new(LatencyToxic))
}
