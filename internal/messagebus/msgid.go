package messagebus

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"
)

var msgSequence uint32

// GenerateMessageID returns a unique, lexically sortable message ID.
func GenerateMessageID() string {
	now := time.Now().UTC()
	seq := atomic.AddUint32(&msgSequence, 1) % 10000
	pid := os.Getpid() % 100000
	return fmt.Sprintf("MSG-%s-%09d-PID%05d-%04d", now.Format("20060102-150405"), now.Nanosecond(), pid, seq)
}
