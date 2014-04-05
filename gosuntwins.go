/*
goSuntwins is a simple utility to read data from JFY Suntwins Solar inverter
 
Tested with Suntwins 5000TL on Linux

Example usage:

   ./gosuntwins -d -p /dev/ttyUSB01 -f /tmp/data.json

Output file will contain a json object per line e.g.:

   {"Current":14.7,"Frequency":50.09,"KW now":7.8,"KW today":10.35,"PV AC":3643.1,"Temperature":46,"Time":"2014-04-05T10:49:52.29109101+10:00","Unknown 1":2494,"Unknown 2":75,"Volts AC":244,"Volts DC":255.1}
   {"Current":14.5,"Frequency":50.11,"KW now":7.9,"KW today":10.36,"PV AC":3636.1,"Temperature":46,"Time":"2014-04-05T10:50:03.40009637+10:00","Unknown 1":2470,"Unknown 2":75,"Volts AC":244.9,"Volts DC":255.7}

*/

package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/tarm/goserial"
)

type Output struct {
	Temp     uint16
	Unknown1 uint16
	VDC      uint16
	CurrentE uint16
	Unknown2 uint16
	TodayE   uint16
	I        uint16
	VAC      uint16
	Freq     uint16
	PAC      uint16
}

var outbuffer bytes.Buffer
var inbuffer bytes.Buffer

const sourceaddr byte = 1
const headerlen int = 7
const period int = 10 //seconds between reads

var destaddr byte = 0
var results map[string]interface{} = make(map[string]interface{})

var debug bool
var serialPort string
var dataFile *os.File

func main() {

	var err error

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, " -d          Enable debug messages  (false)\n")
		fmt.Fprintf(os.Stderr, " -p [port]   serial port            (/dev/ttyUSB0)\n")
		fmt.Fprintf(os.Stderr, " -f [file]   data file              (/tmp/solarmon.csv)\n\n")
	}

	flag.BoolVar(&debug, "d", false, "Enable debug messages")
	flag.StringVar(&serialPort, "p", "/dev/ttyUSB0", "Serial port")
	f := flag.String("f", "/tmp/solarmon.csv", "File to store output data")
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

	results["Time"] = time.Now()
	results["Temperature"] = float32(output.Temp) / 10.0
	results["KW today"] = float32(output.TodayE) / 100.0
	results["Volts DC"] = float32(output.VDC) / 10.0
	results["Current"] = float32(output.I) / 10.0
	results["Volts AC"] = float32(output.VAC) / 10.0
	results["Frequency"] = float32(output.Freq) / 100.0
	results["KW now"] = float32(output.CurrentE) / 10.0
	results["Unknown 1"] = float32(output.Unknown1)
	results["Unknown 2"] = float32(output.Unknown2)
	results["PV AC"] = float32(output.PAC) / 10.0


	jb, err := json.Marshal(results)
	if err != nil {
		return err
	}

	mylogf("%s\n", jb)
	_, err = dataFile.WriteString(string(jb)+"\n")
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
