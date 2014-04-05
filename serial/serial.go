/*
goSuntwins is a simple utility to read data from JFY Suntwins Solar inverter

Tested with Suntwins 5000TL on Linux

Example usage:

  ./gosuntwins -d -p /dev/ttyUSB01 -f /tmp/data.csv

Output file will contain one reading per line e.g.:

  2014-04-05 13:33:43.863091911 +1000 EST, 47.700, 19.290, 254.000, 6.700, 244.900, 49.970, 4.700, 1731.000, 41.000, 1790.800, 
  2014-04-05 13:33:54.97314362 +1000 EST, 47.700, 19.290, 253.400, 3.500, 244.000, 49.990, 1.900, 1719.000, 18.000, 808.700, 

Credit:

Code based on Solarmon: https://github.com/ridale/solarmon and other inspiration from Solarmonj: http://code.google.com/p/solarmonj/
*/

package serial

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/tarm/goserial"
)

// This struct holds the binary data read from the inverter. 
// Order of fields is important!
type rawData struct {
	Temp     uint16
	Unknown1 uint16
	VDC      uint16
	NowE uint16
	Unknown2 uint16
	TodayE   uint16
	I        uint16
	VAC      uint16
	Freq     uint16
	PAC      uint16
}

// Holds decimal values 
type Reading struct {
	// Temperature (degrees celcius)
	Temp float32
	// PV input voltage (DC)
	VDC float32
	// Energy being produced now (kWH)
	NowE float32
	// Energy produced today (kWH)
	TodayE float32
	// PV output current (Amps)
	I float32
	// Grid voltage (AC)
	VAC float32
	// Grid frequency (Hz)
	Freq float32
	// Engergy being produced now (W)
	PAC float32
}

type Config struct {
	// Serial port device name
	Name string

	// Enable debug output
	Debug bool
}

var outbuffer bytes.Buffer

const sourceaddr byte = 1
const headerlen int = 7

var destaddr byte = 0

var config *Config

func OpenPort(c *Config) (io.ReadWriteCloser,error) {

	config = c

	c2 := &serial.Config{Name: config.Name, Baud: 9600}
	s, err := serial.OpenPort(c2)
	if err != nil {
		return nil, err
	}

	err = initInverter(s)
	if err != nil {
		return nil, fmt.Errorf("Error initializing inverter, %s", err)
	}

	return s, nil
}

func mylogf(format string, args ...interface{}) {
	if config.Debug {
		log.Printf(format, args...)
	}
}

func mylogln(args ...interface{}) {
	if config.Debug {
		log.Println(args...)
	}
}

func initInverter(s io.ReadWriteCloser) error {
	var control byte = 0x30
	var function byte = 0x44 //Initialize inverter
	err := createCommand(control, function, nil)
	if err != nil {
		return err
	}

	mylogf("Initializing inverter: => %X\n", outbuffer.Bytes())

	_, err = s.Write(outbuffer.Bytes())
	if err != nil {
		return err
	}

	// wait before sending next command
	time.Sleep(time.Millisecond * 500)

	function = 0x40 //Identify inverter
	err = createCommand(control, function, nil)
	if err != nil {
		return err
	}

	mylogf("Identifying inverter: =>  %X\n", outbuffer.Bytes())

	n, err := s.Write(outbuffer.Bytes())
	if err != nil {
		return err
	}

	//mylogf("Wrote %d bytes\n", n)

	inbuf, err := readtoEOF(s)
	if err != nil {
		return err
	}

	mylogf("Read data: <=  %X\n", inbuf[:n])
	if n < headerlen {
		return fmt.Errorf("Too few bytes read. Expected >= %d, got %d\n", headerlen, n)
	}

	// wait before sending next command
	time.Sleep(time.Millisecond * 500)

	function = 0x41 // Register inverter
	// get the serial number from the response
	//serno := make([]byte,inbuf[6])
	serno := inbuf[headerlen:(int(inbuf[6]) + headerlen + 1)]

	//mylogf("headerlen %d inbuf6 %#v\n", headerlen, inbuf[6])

	// set the device id
	serno[inbuf[6]] = 1

	//mylogf("serno %#v len %d\n", serno, len(serno))

	// now register the inverter as device id 1
	err = createCommand(control, function, serno[:inbuf[6]+1])
	if err != nil {
		return err
	}

	mylogf("Register inverter: =>  %X\n", outbuffer.Bytes())

	_, err = s.Write(outbuffer.Bytes())
	if err != nil {
		return err
	}

	inbuf, err = readtoEOF(s)
	if err != nil {
		return err
	}

	mylogf("Read data: <=  %X\n", inbuf)
	if len(inbuf) < headerlen {
		return fmt.Errorf("Too few bytes read. Expected >= %d, got %d\n", headerlen, len(inbuf))
	}

	return nil
}

func readtoEOF(s io.ReadWriteCloser) ([]byte, error) {
	var inbuf, tmpbuf []byte
	for {
		tmpbuf = make([]byte, 256)
		n, err := s.Read(tmpbuf)
		//mylogf("Read %d bytes\n", n)
		inbuf = append(inbuf, tmpbuf[:n]...)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
	}
	return inbuf, nil
}

func ReadInverter(s io.ReadWriteCloser) (*Reading, error) {
	var control byte = 0x31
	var function byte = 0x42 //Get Dynamic data
	destaddr = 1
	err := createCommand(control, function, nil)
	if err != nil {
		return nil, err
	}

	mylogf("Requesting current readings: => %X\n", outbuffer.Bytes())

	_, err = s.Write(outbuffer.Bytes())
	if err != nil {
		return nil, err
	}

	inbuf, err := readtoEOF(s)
	if err != nil {
		return nil, err
	}

	mylogf("Read data: <=  %X\n", inbuf)

	if len(inbuf) < headerlen+20 {
		return nil, fmt.Errorf("Too few bytes read. Expected >= %d, got %d\n", headerlen+20, inbuf)
	}

	b := bytes.NewBuffer(inbuf)
	raw := rawData{}
	err = binary.Read(b, binary.BigEndian, &raw)
	if err != nil {
		return nil, err
	}

	reading := &Reading{}

	reading.Temp = float32(raw.Temp) / 10.0
	reading.TodayE = float32(raw.TodayE) / 100.0
	reading.VDC = float32(raw.VDC) / 10.0
	reading.I = float32(raw.I) / 10.0
	reading.VAC = float32(raw.VAC) / 10.0
	reading.Freq = float32(raw.Freq) / 100.0
	reading.NowE = float32(raw.NowE) / 10.0
	reading.PAC = float32(raw.PAC) / 10.0

	return reading, nil
}

func createCommand(control byte, function byte, data []byte) error {
	// the command cannot be greater than max unsigned byte minus overhead
	if len(data) > 240 {
		return errors.New("Command length too long")
	}

	outbuffer.Reset()
	outbuffer.WriteByte(0xa5)
	outbuffer.WriteByte(0xa5)
	outbuffer.WriteByte(sourceaddr)
	outbuffer.WriteByte(destaddr)
	outbuffer.WriteByte(control)
	outbuffer.WriteByte(function)
	outbuffer.WriteByte(byte(len(data)))

	if data != nil {
		outbuffer.Write(data)
	}

	check1, check2 := checksum(outbuffer.Bytes())

	//mylogf("check1 %#v check2 %#v\n", check1, check2)

	outbuffer.WriteByte(check1)
	outbuffer.WriteByte(check2)
	outbuffer.WriteByte('\n')
	outbuffer.WriteByte('\r')

	return nil
}

func checksum(data []byte) (byte, byte) {
	var sum uint16 = 0

	for i := 0; i < len(data); i++ {
		//mylogf("datai sum %v %v\n", data[i], sum)
		sum += uint16(data[i])
	}

	// Flip bits (XOR)
	sum ^= 0xffff
	sum++

	check1 := byte((sum & 0xff00) >> 8)
	check2 := byte(sum & 0x00ff)
	return check1, check2
}
