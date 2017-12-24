package main

import (
	"fmt"
	"log"
	"time"

	"github.com/phob0s-pl/pigpio"
)

const (
	PinLeft   = 27
	PinRight  = 23
	PinCenter = 4
	PinUp     = 17
	PinDown   = 22
	PinA      = 5
	PinB      = 6
)

func main() {
	if err := preparePins(); err != nil {
		log.Fatalf("failed to prepare pins, err=%v", err)
	}

	gpio := pigpio.NewPiGPIO(1)
	if err := gpio.WatchMultiPin(
		PinLeft,
		PinRight,
		PinCenter,
		PinUp,
		PinDown,
		PinA,
		PinB); err != nil {
		log.Fatalf("failed to watch pins, err=%v", err)
	}

	debounced := pigpio.Debouncer(gpio.Notify, time.Millisecond*250)
	for {
		select {
		case pin := <-debounced:
			fmt.Printf("State changed on pin=%d\n", pin)
		}
	}
}

func preparePins() error {
	if err := pigpio.SetPinEdge(PinLeft, pigpio.EdgeFalling); err != nil {
		return err
	}
	if err := pigpio.SetPinEdge(PinRight, pigpio.EdgeFalling); err != nil {
		return err
	}
	if err := pigpio.SetPinEdge(PinCenter, pigpio.EdgeFalling); err != nil {
		return err
	}
	if err := pigpio.SetPinEdge(PinUp, pigpio.EdgeFalling); err != nil {
		return err
	}
	if err := pigpio.SetPinEdge(PinDown, pigpio.EdgeFalling); err != nil {
		return err
	}
	if err := pigpio.SetPinEdge(PinA, pigpio.EdgeFalling); err != nil {
		return err
	}
	if err := pigpio.SetPinEdge(PinB, pigpio.EdgeFalling); err != nil {
		return err
	}
	return nil
}
