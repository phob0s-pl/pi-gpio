package pigpio

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"time"

	"golang.org/x/sys/unix"
)

// Edge describes possible edge states of input pin
type Edge uint

const (
	// EdgeFalling represents falling edge
	EdgeFalling Edge = iota
	// EdgeRising represents rising edge
	EdgeRising
	// EdgeBoth represents both falling and rising edge
	EdgeBoth
	// EdgeNone represents no edge and disables interrupt
	EdgeNone
)

const (
	pinValuePath = "/sys/class/gpio/gpio%d/value"
	// FIONREAD gets the number of bytes that are immediately available for reading.
	FIONREAD = 21531
)

// String return string representation of Edge
// The value is compatible with gpio command
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

// PiGPIO is structure watching pin state change
type PiGPIO struct {
	Notify chan int
}

// NewPiGPIO returns allocated PiGPIO structure with selected channel size
func NewPiGPIO(channelSize uint) *PiGPIO {
	return &PiGPIO{
		Notify: make(chan int, channelSize),
	}
}

// WatchMultiPin registers multiple pins to watch for state change.
func (p *PiGPIO) WatchMultiPin(pinNum ...int) error {
	for i := range pinNum {
		if err := p.WatchPin(pinNum[i]); err != nil {
			return err
		}
	}
	return nil
}

// WatchPin registers single pin to watch for state change.
// When pin state changes the notification is send to PiGPIO.Notify channel
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

// Debouncer creates new channel which sends values from notify channel
// after timer expires. All other values are dropped.
func Debouncer(notify chan int, t time.Duration) chan int {
	var (
		changedState   bool
		capacity       = cap(notify)
		debuncedNotify = make(chan int, capacity)
		timer          = time.NewTimer(t)
	)

	go func() {
		for {
			select {
			case val := <-notify:
				if !changedState {
					changedState = true
					debuncedNotify <- val
					timer.Reset(t)
				}
			case <-timer.C:
				changedState = false
			}
		}
	}()

	return debuncedNotify
}
