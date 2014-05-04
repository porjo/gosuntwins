[![Build Status](https://travis-ci.org/porjo/gosuntwins.svg)](https://travis-ci.org/porjo/gosuntwins)

# GoSuntwins

Gosuntwins is a simple utility to read data from JFY Suntwins Solar inverter.
It has been tested with Suntwins 5000TL inverter from Linux operating system
 
### Howto

Precompiled binaries for Linux are available for download on the [releases page](https://github.com/porjo/gosuntwins/releases).

Otherwise, compile Gosuntwins yourself as follows:

First [install Golang](http://golang.org/doc/install#install). On Redhat Linux that should be as simple as `yum install golang`. Then:

- Install the repository `go get github.com/porjo/gosuntwins`
- Change to install loction `cd $GOPATH/src/github.com/porjo/gosuntwins`
- Build the binary `go build gosuntwins.go`
- Copy the binary wherever you want it

**Example usage:**

```
./gosuntwins -d -p /dev/ttyUSB01 -f /tmp/data.csv
 ```

Output file will contain one reading per line e.g.:

```
2014-04-05 13:33:43.863091911 +1000 EST, 47.700, 19.290, 254.000, 6.700, 244.900, 49.970, 4.700, 1731.000, 41.000, 1790.800, 
2014-04-05 13:33:54.97314362 +1000 EST, 47.700, 19.290, 253.400, 3.500, 244.000, 49.990, 1.900, 1719.000, 18.000, 808.700, 
```

### PVOutput.org

Support for http://pvoutput.org is enabled by default. Gosuntwins will look for the following environment variables: `PVSTATUSURL`, `PVAPIKEY` & `PVSYSTEMID`. If these are set, then the following values will be uploaded every 5mins:

```
v1 - Energy Consumption (TodayE * 1000)
v2 - Power Generation (PAC)
v5 - Temperature
v6 - Voltage (VDC)
```

Refer to [pvoutput.org API doco](http://pvoutput.org/help.html#api-addstatus) for more info on the above values.

### Credits

Code based on Solarmon: https://github.com/ridale/solarmon and inspiration from Solarmonj: http://code.google.com/p/solarmonj/
