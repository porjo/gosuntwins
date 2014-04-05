# GoSuntwins

goSuntwins is a simple utility to read data from JFY Suntwins Solar inverter
 
Tested with Suntwins 5000TL on Linux

### Howto

First [install Golang](http://golang.org/doc/install#install). On Redhat Linux that should be as simple as `yum install golang`. Then:

- Install the repository `go get github.com/porjo/gosuntwins`
- Change to install loction `cd $GOPATH/src/github.com/porjo/gosuntwins`
- Build the binary `go build gosuntwins.go`
- Copy the binary wherever you want it

**Example usage:**

```
./gosuntwins -d -p /dev/ttyUSB01 -f /tmp/data.json
 ```

Output file will contain a json object per line e.g.:

```
{"Current":14.7,"Frequency":50.09,"KW now":7.8,"KW today":10.35,"PV AC":3643.1,"Temperature":46,"Time":"2014-04-05T10:49:52.29109101+10:00","Unknown 1":2494,"Unknown 2":75,"Volts AC":244,"Volts DC":255.1}
{"Current":14.5,"Frequency":50.11,"KW now":7.9,"KW today":10.36,"PV AC":3636.1,"Temperature":46,"Time":"2014-04-05T10:50:03.40009637+10:00","Unknown 1":2470,"Unknown 2":75,"Volts AC":244.9,"Volts DC":255.7}
```

### Credits

Code based on Solarmon: https://github.com/ridale/solarmon and inspiration from Solarmonj: http://code.google.com/p/solarmonj/
