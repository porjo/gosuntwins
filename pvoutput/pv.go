package pvoutput

import (
	"bytes"
	"log"
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
	lastUpload time.Time = time.Now()
	addCount   float32       = 1
	total      serial.Reading   = serial.Reading{}
	pv         pvoutput  = pvoutput{}
	client *http.Client 
)

//seconds between uploads
const interval int = 300

func init() {
	client = &http.Client{}
	pv.statusURL = "http://pvoutput.org/service/r2/addstatus.jsp"
	pv.apiKey = os.Getenv("pvapikey")
	pv.systemID = os.Getenv("pvsystemid")
}

func Upload(r *serial.Reading) (err error) {

	AddReading(r)

	if time.Now().Sub(lastUpload) >= (time.Second * time.Duration(interval)) {
		var req *http.Request
		//var res *http.Response
		data := url.Values{}
		data.Set("d", time.Now().Format("20060102"))
		data.Set("t", time.Now().Format("15:04"))
		data.Set("v1", strconv.FormatFloat(float64(total.TodayE), 'f', 3, 32))
		data.Set("v2", strconv.FormatFloat(float64(total.PAC), 'f', 3, 32))
		data.Set("v5", strconv.FormatFloat(float64(total.Temp), 'f', 3, 32))
		data.Set("v6", strconv.FormatFloat(float64(total.VDC), 'f', 3, 32))

		req, err = http.NewRequest("POST", pv.statusURL, bytes.NewBufferString(data.Encode()))
		if err != nil {
			log.Println(err)
			return
		}
		req.Header.Set("X-Pvoutput-Apikey", pv.apiKey)
		req.Header.Set("X-Pvoutput-SystemId", pv.systemID)

		log.Println("POSTing to URL: " + pv.statusURL)
		log.Printf("POSTing data: %v\n", data)
		//res, err = client.Do(req)
		_, err = client.Do(req)
		if err != nil {
			log.Println(err)
			return
		}

		lastUpload = time.Now()
		total = serial.Reading{}
		addCount = 1
	}
	return
}

func AddReading(r *serial.Reading) {

	total.Temp = (total.Temp + r.Temp) / addCount
	total.VDC = (total.VDC + r.VDC) / addCount
	total.NowE = (total.NowE + r.NowE) / addCount
	total.TodayE = r.TodayE
	total.I = (total.I + r.I) / addCount
	total.VAC = (total.VAC + r.VAC) / addCount
	total.Freq = (total.Freq + r.Freq) / addCount
	total.PAC = (total.PAC + r.PAC) / addCount

	addCount++
}
