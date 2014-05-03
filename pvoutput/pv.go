package pvoutput

import (
	"bytes"
	"fmt"
	//	"io/ioutil"
	//	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/porjo/gosuntwins/serial"
)

type pvoutput struct {
	statusURL string
	apiKey    string
	systemID  string
}

var (
	lastUpload   time.Time      = time.Now()
	addCount     float32        = 1
	totalReading serial.Reading = serial.Reading{}
	pv           pvoutput       = pvoutput{}
	client       *http.Client
)

//seconds between uploads
const interval int = 300

func init() {
	client = &http.Client{}
	pv.statusURL = os.Getenv("PVSTATUSURL")
	pv.apiKey = os.Getenv("PVAPIKEY")
	pv.systemID = os.Getenv("PVSYSTEMID")
}

func Upload(r *serial.Reading) error {

	if pv.statusURL == "" {
		return fmt.Errorf("PV status URL is not set")
	}

	AddReading(r)

	if time.Now().Sub(lastUpload) >= (time.Second * time.Duration(interval)) {

		avg := avgReading(addCount, totalReading)

		data := url.Values{}
		data.Set("d", time.Now().Format("20060102"))
		data.Set("t", time.Now().Format("15:04"))
		data.Set("v1", strconv.FormatFloat(float64(avg.TodayE)*1000, 'f', 3, 32)) //Convert to watt hours
		data.Set("v2", strconv.FormatFloat(float64(avg.PAC), 'f', 3, 32))
		data.Set("v5", strconv.FormatFloat(float64(avg.Temp), 'f', 3, 32))
		data.Set("v6", strconv.FormatFloat(float64(avg.VDC), 'f', 3, 32))

		req, err := http.NewRequest("POST", pv.statusURL, bytes.NewBufferString(data.Encode()))
		if err != nil {
			return err
		}
		req.Header.Set("X-Pvoutput-Apikey", pv.apiKey)
		req.Header.Set("X-Pvoutput-SystemId", pv.systemID)
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

		//log.Println("POSTing to URL: " + pv.statusURL)
		//log.Printf("POSTing data: %v\n", data)
		res, err := client.Do(req)
		if err != nil {
			return err
		}

		/*
			defer res.Body.Close()
			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				return
			}
			log.Printf("Body %s\n", body)
		*/

		if res.StatusCode != 200 {
			return fmt.Errorf("Server responded with code %d\n", res.StatusCode)
		}

		lastUpload = time.Now()
		totalReading = serial.Reading{}
		addCount = 1
	}
	return nil
}

// Add new reading to running total
func AddReading(r *serial.Reading) {

	totalReading.Temp += r.Temp
	totalReading.VDC += r.VDC
	totalReading.NowE += r.NowE
	totalReading.TodayE = r.TodayE //this one isn't summed as it's already cumulative
	totalReading.I += r.I
	totalReading.VAC += r.VAC
	totalReading.Freq += r.Freq
	totalReading.PAC += r.PAC

	addCount++
}

// Calculate average of supplied reading
func avgReading(count float32, r serial.Reading) serial.Reading {

	avg := serial.Reading{}

	avg.Temp = r.Temp / count
	avg.VDC = r.VDC / count
	avg.NowE = r.NowE / count
	avg.TodayE = r.TodayE
	avg.I = r.I / count
	avg.VAC = r.VAC / count
	avg.Freq = r.Freq / count
	avg.PAC = r.PAC / count

	return avg
}
