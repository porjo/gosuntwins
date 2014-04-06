# GoSuntwins

Gosuntwins is a simple utility to read data from JFY Suntwins Solar inverter.
It has been tested with Suntwins 5000TL inverter from Linux operating system
 
### Howto

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

### Credits

Code based on Solarmon: https://github.com/ridale/solarmon and inspiration from Solarmonj: http://code.google.com/p/solarmonj/
