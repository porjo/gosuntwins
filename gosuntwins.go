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

package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/tarm/goserial"
)

// This struct holds the binary data read from the inverter. 
// Order of fields is important!
type Output struct {
	// Temperature (degrees celcius)
	Temp     uint16

	Unknown1 uint16

	// PV input voltage (DC)
	VDC      uint16

	// Energy being produced now (kWH)
	NowE uint16

	Unknown2 uint16

	// Energy produced today (kWH)
	TodayE   uint16

	// PV output current (Amps)
	I        uint16

	// Grid voltage (AC)
	VAC      uint16


	// Grid frequency (Hz)
	Freq     uint16

	// Engergy being produced now (W)
	PAC      uint16
}

var outbuffer bytes.Buffer
var inbuffer bytes.Buffer

const sourceaddr byte = 1
const headerlen int = 7
const period int = 10 //seconds between reads

var destaddr byte = 0

var debug bool
var serialPort string
var dataFile *os.File

func main() {

	var err error

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, " -d          Enable debug messages  (false)\n")
		fmt.Fprintf(os.Stderr, " -p [port]   serial port            (/dev/ttyUSB0)\n")
		fmt.Fprintf(os.Stderr, " -f [file]   data file              (/tmp/gosuntwins.csv)\n\n")
	}

	flag.BoolVar(&debug, "d", false, "Enable debug messages")
	flag.StringVar(&serialPort, "p", "/dev/ttyUSB0", "Serial port")
	f := flag.String("f", "/tmp/gosuntwins.csv", "File to store output data")
	flag.Parse()

	dataFile, err = os.OpenFile(*f, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		log.Fatal(err)
	}
	defer dataFile.Close()

	c := &serial.Config{Name: serialPort, Baud: 9600}
	s, err := serial.OpenPort(c)
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()

	mylogf("Writing results to file '%s'\n", *f)

	err = initInverter(s)
	if err != nil {
		log.Fatal("Error initializing inverter, ", err)
	}

	for {
		err = readInverter(s)
		if err != nil {
			log.Fatal("Error reading from inverter, ", err)
			break
		}
		err = outputInverter()
		if err != nil {
			log.Fatal("Error outputing data, ", err)
			break
		}
		time.Sleep(time.Second * time.Duration(period))
	}
}

func mylogf(format string, args ...interface{}) {
	if debug {
		log.Printf(format, args...)
	}
}

func mylogln(args ...interface{}) {
	if debug {
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

	n, inbuf, err := readtoEOF(s)
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

	n, inbuf, err = readtoEOF(s)
	if err != nil {
		return err
	}

	mylogf("Read data: <=  %X\n", inbuf[:n])
	if n < headerlen {
		return fmt.Errorf("Too few bytes read. Expected >= %d, got %d\n", headerlen, n)
	}

	return nil
}

func readtoEOF(s io.ReadWriteCloser) (int, []byte, error) {
	var inbuf, tmpbuf []byte
	bytecount := 0
	for {
		tmpbuf = make([]byte, 256)
		n, err := s.Read(tmpbuf)
		//mylogf("Read %d bytes\n", n)
		inbuf = append(inbuf, tmpbuf[:n]...)
		bytecount += n
		if err != nil {
			if err == io.EOF {
				break
			}
			return bytecount, nil, err
		}
	}
	return bytecount, inbuf, nil
}

func readInverter(s io.ReadWriteCloser) error {
	var control byte = 0x31
	var function byte = 0x42 //Get Dynamic data
	destaddr = 1
	err := createCommand(control, function, nil)
	if err != nil {
		return err
	}

	mylogf("Requesting current readings: => %X\n", outbuffer.Bytes())

	_, err = s.Write(outbuffer.Bytes())
	if err != nil {
		return err
	}

	n, inbuf, err := readtoEOF(s)
	if err != nil {
		return err
	}
	inbuffer.Reset()
	inbuffer.Write(inbuf[:n])

	mylogf("Read data: <=  %X\n", inbuffer.Bytes())

	if n < headerlen {
		return fmt.Errorf("Too few bytes read. Expected >= %d, got %d\n", headerlen, n)
	}

	return nil
}

func outputInverter() error {
	if len(inbuffer.Bytes()) < headerlen+20 {
		return fmt.Errorf("Too few bytes read. Expected >= %d, got %d\n", headerlen+20, len(inbuffer.Bytes()))
	}

	data := inbuffer.Bytes()[headerlen:]

	b := bytes.NewBuffer(data)
	output := Output{}
	err := binary.Read(b, binary.BigEndian, &output)
	if err != nil {
		return err
	}

	results := make([]float32,10)

	results[0] = float32(output.Temp) / 10.0
	results[1] = float32(output.TodayE) / 100.0
	results[2] = float32(output.VDC) / 10.0
	results[3] = float32(output.I) / 10.0
	results[4] = float32(output.VAC) / 10.0
	results[5] = float32(output.Freq) / 100.0
	results[6] = float32(output.NowE) / 10.0
	results[7] = float32(output.Unknown1)
	results[8] = float32(output.Unknown2)
	results[9] = float32(output.PAC) / 10.0

	resultsStr := fmt.Sprintf("%s, ", time.Now())
	for i:= 0; i< len(results); i++ {
		resultsStr += fmt.Sprintf("%.3f, ", results[i])
	}

	mylogln(resultsStr)

	_, err = dataFile.WriteString(resultsStr + "\n")
	if err != nil {
		return err
	}
	return nil
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
