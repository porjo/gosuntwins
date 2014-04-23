package serial

import (
	"encoding/hex"
//	"io"
	"testing"
//	"fmt"
)

// Implements io.ReaderWriterCloser interface, allowing us to mock a serial port
type rWC struct{}

// Keep track of Which value to send back to client
var readStage int = 0

func TestSerial(t *testing.T) {

	s = &rWC{}
	config = &Config{}

	err := initInverter()
	if err != nil {
		t.Fatal(err)
	}

	reading := &Reading{}
	err = reading.LoadData()
	if err != nil {
		t.Fatal(err)
	}

	if reading.Temp != 47.7 {
		t.Fatalf("Temperature reading does not match. Expected 47.7, got %f\n", reading.Temp)
	}
}

func (r *rWC) Read(p []byte) (int, error) {

	var b []byte
	var err error

	switch readStage {
	case 0:
		b, err = hex.DecodeString("A5A5000030BF1031353232313334343130323038202020FAC60A0D")
	case 1:
		b, err = hex.DecodeString("A5A5010130BE0106FDBF0A0D")
	case 2:
		b, err = hex.DecodeString("A5A5010131BD3001DD09C9095E001600160516002C096B138E27F4FFFF0000120C00000000000100000000000000000000000000000000F6BF0A0D")
	}

	readStage++

	copy(p, b)
	return len(b), err
}

func (r *rWC) Write(p []byte) (int, error) {
	return len(p), nil
}

func (r *rWC) Close() error {
	return nil
}
