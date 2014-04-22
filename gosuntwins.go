/*
Gosuntwins is a simple utility to read data from JFY Suntwins Solar inverter.
It has been tested with Suntwins 5000TL inverter from Linux operating system

Example usage:

  ./gosuntwins -d -p /dev/ttyUSB01 -f /tmp/data.csv

Output file will contain one reading per line e.g.:

  2014-04-05 13:33:43.863091911 +1000 EST, 47.700, 19.290, 254.000, 6.700, 244.900, 49.970, 4.700, 1731.000, 41.000, 1790.800,
  2014-04-05 13:33:54.97314362 +1000 EST, 47.700, 19.290, 253.400, 3.500, 244.000, 49.990, 1.900, 1719.000, 18.000, 808.700,

Credits

Code based on Solarmon: https://github.com/ridale/solarmon plus inspiration from Solarmonj: http://code.google.com/p/solarmonj/
*/
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"
	"reflect"

	"github.com/porjo/gosuntwins/serial"
	"github.com/porjo/gosuntwins/pvoutput"
)

const period int = 10 //seconds between reads

var debug bool
var dataFile *os.File

func main() {

	var err error

	var serialPort string

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

	fmt.Printf("Writing results to file '%s'\n", *f)

	config := &serial.Config{Port: serialPort, Debug: debug}

	s, err := serial.OpenPort(config)
	if err != nil {
		log.Fatal("Error initializing inverter, ", err)
	}
	defer s.Close()

	for {
		reading := &serial.Reading{}
		err := reading.LoadData()
		if err != nil {
			log.Fatal("Error reading from inverter, ", err)
			break
		}

		err = pvoutput.Upload(reading)
		if err != nil {
			log.Fatal("Error uploading data to PVoutput, ", err)
			break
		}

		err = outputInverter(reading)
		if err != nil {
			log.Fatal("Error outputing data, ", err)
			break
		}
		time.Sleep(time.Second * time.Duration(period))
	}
}

func outputInverter(reading *serial.Reading) error {

	resultsStr := fmt.Sprintf("%s, ", time.Now())

	s := reflect.ValueOf(reading).Elem()
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		iface := f.Interface()
		if val, ok := iface.(float32); ok {
			resultsStr += fmt.Sprintf("%.3f, ", val)
		} else {
			return fmt.Errorf("Error reading value %v\n", i)
		}
	}

	if debug {
		fmt.Println(resultsStr)
	}

	_, err := dataFile.WriteString(resultsStr + "\n")
	if err != nil {
		return err
	}

	return nil
}
