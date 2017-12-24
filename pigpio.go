package pigpio

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"

	"golang.org/x/sys/unix"
)

type Edge uint

const (
	EdgeFalling Edge = iota
	EdgeRising
	EdgeBoth
	EdgeNone
)

const (
	pinValuePath = "/sys/class/gpio/gpio%d/value"
	FIONREAD     = 21531
)

func (e Edge) String() string {
	switch e {
	case EdgeFalling:
		return "falling"
	case EdgeRising:
		return "rising"
	case EdgeBoth:
		return "both"
	case EdgeNone:
		return "none"
	default:
		return ""
	}
}

// SetPinEdge sets pin to output, enables pull up register
// and sets proper edge interrupt.
// The pin number is in BCM notation
func SetPinEdge(pinNum int, edge Edge) error {
	pin := strconv.Itoa(pinNum)

	out, err := exec.Command("gpio", "-g", "mode", pin, "in").CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to make pin %d input, err=%v, msg=%s", pinNum, err, string(out))
	}
	out, err = exec.Command("gpio", "-g", "mode", pin, "up").CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to make pin %d up, err=%v, msg=%s", pinNum, err, string(out))
	}

	out, err = exec.Command("gpio", "edge", pin, "falling").CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to set pin %d edge=%s interrupt, err=%v, msg=%s", pinNum, edge, err, string(out))
	}
	return nil
}

type PiGPIO struct {
	Notify chan int
}

func NewPiGPIO(channelSize uint) *PiGPIO {
	return &PiGPIO{
		Notify: make(chan int, channelSize),
	}
}

func (p *PiGPIO) WatchMultiPin(pinNum ...int) error {
	for i := range pinNum {
		if err := p.WatchPin(pinNum[i]); err != nil {
			return err
		}
	}
	return nil
}

func (p *PiGPIO) WatchPin(pinNum int) error {
	const ()
	b := make([]byte, 1)

	path := fmt.Sprintf(pinValuePath, pinNum)
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("WatchPin failed for pin=%d, err=%v", pinNum, err)
	}

	count, err := unix.IoctlGetInt(int(f.Fd()), FIONREAD)
	if err != nil {
		return fmt.Errorf("WatchPin failed for pin=%d, err=%v", pinNum, err)
	}
	// clear interrupts
	for i := 0; i < count; i++ {
		_, _ = f.Read(b)
	}

	go pinPoll(f, pinNum, p.Notify)
	return nil
}

func pinPoll(f *os.File, pinNum int, notify chan int) {
	fd := unix.PollFd{
		Fd:     int32(f.Fd()),
		Events: unix.POLLPRI | unix.POLLERR,
	}
	b := make([]byte, 1)

	for {
		interruptCount, err := unix.Poll([]unix.PollFd{fd}, -1)
		if err != nil || interruptCount <= 0 {
			continue
		}
		unix.Seek(int(fd.Fd), 0, io.SeekStart)
		_, _ = f.Read(b)
		notify <- pinNum
	}
}
