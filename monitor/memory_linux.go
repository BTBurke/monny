// +build linux
package monitor

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
)

func calculateMemory(pid int) uint64 {
	f, err := os.Open(fmt.Sprintf("/proc/%d/smaps", pid))
	if err != nil {
		return 0
	}
	defer f.Close()

	res := uint64(0)
	pfx := []byte("Pss:")
	r := bufio.NewScanner(f)
	for r.Scan() {
		line := r.Bytes()
		if bytes.HasPrefix(line, pfx) {
			var size uint64
			_, err := fmt.Sscanf(string(line[4:]), "%d", &size)
			if err != nil {
				return 0
			}
			res += size
		}
	}
	if err := r.Err(); err != nil {
		return 0
	}
	return res
}
