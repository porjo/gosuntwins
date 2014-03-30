package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	//"time"

//	"github.com/tarm/goserial"
)

var outbuffer bytes.Buffer
var inbuffer bytes.Buffer

const sourceaddr byte = 1

var destaddr byte = 0

const headerlen int = 7

var serialPort string = "/dev/ttyUSB0"

type RWC struct{}

func main() {
	/*
		c := &serial.Config{Name: serialPort, Baud: 9600}
		s, err := serial.OpenPort(c)
		if err != nil {
			log.Fatal(err)
		}
	*/

	s := RWC{}
	err := initInverter(s)
	if err != nil {
		log.Println(err)
		return
	}
	err = readInverter(s)
	if err != nil {
		log.Println(err)
		return
	}
	err = outputInverter()
	if err != nil {
		log.Println(err)
		return
	}

}

func (rwc RWC) Read(p []byte) (n int, err error) {
	copy(p, []byte("Header812345678912345678912"))
	if p != nil {
		p[6] = 20 
	}
	return 27, nil
}

func (rwc RWC) Write(p []byte) (n int, err error) {
	return len(p), nil
}

func (rwc RWC) Close() (err error) {
	return nil
}

func initInverter(s io.ReadWriteCloser) error {
	var control byte = 0x30
	var function byte = 0x44
	err := createCommand(control, function, nil)
	if err != nil {
		return err
	}
	log.Printf("outbuffer %#v\n", outbuffer.Bytes())

	n, err := s.Write(outbuffer.Bytes())
	if err != nil {
		return err
	}

	// wait before sending next command
	//time.Sleep(time.Second)

	function = 0x40
	err = createCommand(control, function, nil)
	if err != nil {
		return err
	}
	log.Printf("outbuffer %#v\n", outbuffer.Bytes())
	n, err = s.Write(outbuffer.Bytes())
	if err != nil {
		return err
	}
	inbuf := make([]byte, 256)
	n, err = s.Read(inbuf)
	if err != nil {
		return err
	}
	if n < headerlen {
		return fmt.Errorf("Too few bytes read. Expected >= %d, got %d\n", headerlen, n)
	}

	inbuffer.Write(inbuf[:n])

	// wait before sending next command
	//time.Sleep(time.Second)

	log.Printf("inbuf %#v, n=%d\n", inbuf, n)
	function = 0x41
	// get the serial number from the response
	//serno := make([]byte,inbuf[6])
	serno := inbuf[headerlen:(int(inbuf[6]) + headerlen + 1)]
	log.Printf("headerlen %d inbuf6 %#v\n", headerlen, inbuf[6])

	// set the device id
	serno[inbuf[6]] = 1
	log.Printf("serno %#v len %d\n", serno, len(serno))
	// now register the inverter as device id 1
	err = createCommand(control, function, serno[:inbuf[6]+1])
	if err != nil {
		return err
	}
	log.Printf("outbuffer %#v\n", outbuffer.Bytes())
	n, err = s.Write(outbuffer.Bytes())
	if err != nil {
		return err
	}
	n, err = s.Read(nil)
	if err != nil {
		return err
	}
	if n < headerlen {
		return fmt.Errorf("Too few bytes read. Expected >= %d, got %d\n", headerlen, n)
	}

	return nil
}

func readInverter(s io.ReadWriteCloser) error {
	var control byte = 0x31
	var function byte = 0x42
	destaddr = 1
	err := createCommand(control, function, nil)
	if err != nil {
		return err
	}
	log.Printf("outbuffer %#v\n", outbuffer.Bytes())
	n, err := s.Write(outbuffer.Bytes())
	if err != nil {
		return err
	}
	inbuf := make([]byte, 256)
	n, err = s.Read(inbuf)
	if err != nil {
		return err
	}
	if n < headerlen {
		return fmt.Errorf("Too few bytes read. Expected >= %d, got %d\n", headerlen, n)
	}
	inbuffer.Reset()
	inbuffer.Write(inbuf[:n])

	return nil
}

func outputInverter() error {
	if len(inbuffer.Bytes()) < headerlen+20 {
		return fmt.Errorf("Too few bytes read. Expected >= %d, got %d\n", headerlen+20, len(inbuffer.Bytes()))
	}
	buf := inbuffer.Bytes()[headerlen:]

	temp := float32((buf[0]<<8)+buf[1]) / 10.0
	todayE := float32((buf[2]<<8)+buf[3]) / 100.0
	VDC := float32((buf[4]<<8)+buf[5]) / 10.0
	I := float32((buf[6]<<8)+buf[7]) / 10.0
	VAC := float32((buf[8]<<8)+buf[9]) / 10.0
	freq := float32((buf[10]<<8)+buf[11]) / 100.0
	currE := float32((buf[12] << 8) + buf[13])
	unk1 := float32((buf[14] << 8) + buf[15])
	unk2 := float32((buf[16] << 8) + buf[17])
	totE := float32((buf[18]<<8)+buf[19]) / 10.0

	fmt.Printf("temp:%.3f TodayE:%.3f VDC:%.3f I:%.3f VAC:%.3f Freq:%.3f CurrE:%.3f unk1:%.3f unk2:%.3f TotE:%.3f\n", temp, todayE, VDC, I, VAC, freq, currE, unk1, unk2, totE)

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

	sum1, sum2, err := checksum(headerlen + len(data))
	if err != nil {
		return err
	}


	outbuffer.WriteByte(byte(sum1))
	outbuffer.WriteByte(byte(sum2))
	outbuffer.WriteByte('\n')
	outbuffer.WriteByte('\r')

	return nil
}

// Fletcher's checksum
// http://en.wikipedia.org/wiki/Fletcher%27s_checksum
func checksum(length int) (uint64, uint64, error) {
	var sum1, sum2 uint64 = 0,0

	reader := bytes.NewReader(outbuffer.Bytes())
	for idx := 0; idx < length; idx++ {
		i, err := binary.ReadUvarint(reader)
		if err != nil {
			if err == io.EOF {
				break
			}
			return 0,0, err
		}
		sum1 += i

	}
	// Toggle bits (XOR)
	sum1 ^= 0xffff
	//apply bitmask
	sum2 = (sum1 + 1) & 0x00ff
	sum1 = (sum1 & 0xff00) >> 8
	return sum1, sum2, nil
}
