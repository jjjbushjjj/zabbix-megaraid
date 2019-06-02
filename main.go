package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/blacked/go-zabbix"
	// "github.com/buaazp/diskutil"
	"github.com/jjjbushjjj/diskutil"
)

var (
	megaPath     string
	adapterCount int
)

// Lld represents Phisycal drives lld data sended to zabbix-server
type Lld struct {
	Res []PD `json:"data"`
}

// Adapters discovered
// type Adapters struct {
// 	Adapter int `json:"{#ADAPTER}"`
// 	Pd      []PD
// }

// Physical drives is a list discovered via lld
type PD struct {
	Adapter int `json:"{#ADAPTER}"`
	Pd      int `json:"{#PD}"`
	// Sn      string
	// Status  string
}

func init() {
	flag.StringVar(&megaPath, "mega-path", "/opt/MegaRAID/MegaCli/MegaCli64", "megaCli binary path")
	flag.IntVar(&adapterCount, "adapter-count", 1, "adapter count in your server")
}

var (
	zbxServer = os.Args[1] // Zabbix Server address for zabbix sender
	zbxHost   = os.Args[2] // Zabbix var {HOST.HOST} used in zabbix sender
)

func main() {
	var res Lld
	var physicalDrive PD
	var metrics []*zabbix.Metric
	z := zabbix.NewSender(zbxServer, 10051)
	var packet *zabbix.Packet

	flag.Parse()
	ds, err := diskutil.NewDiskStatus(megaPath, adapterCount)
	if err != nil {
		fmt.Fprintf(os.Stderr, "DiskStatus New error: %v\n", err)
		return
	}

	err = ds.Get()
	if err != nil {
		fmt.Fprintf(os.Stderr, "DiskStatus Get error: %v\n", err)
		return
	}

	for i, ads := range ds.AdapterStats {
		physicalDrive.Adapter = i
		for _, pds := range ads.PhysicalDriveStats {

			// physicalDrive.Status = pds.FirmwareState
			pdName := []string{pds.Brand, pds.Model, pds.SerialNumber}
			sn := strings.Join(pdName, " ")
			physicalDrive.Pd = pds.SlotNumber
			res.Res = append(res.Res, physicalDrive)
			keySt := fmt.Sprintf("raidarray.pd.status[%v,%v]", physicalDrive.Adapter, physicalDrive.Pd)
			keySn := fmt.Sprintf("raidarray.pd.serial[%v,%v]", physicalDrive.Adapter, physicalDrive.Pd)
			metrics = append(metrics, zabbix.NewMetric(zbxHost, keySt, pds.FirmwareState))
			metrics = append(metrics, zabbix.NewMetric(zbxHost, keySn, sn))
			// fmt.Printf("PD%d: %s status: %s\n", pdSlot, pdSN, pdStatus)
		}
		resJSON, err := json.MarshalIndent(res, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Json Marshall error: %v\n", err)
		}
		// Output LLD struct for zabbix
		fmt.Println(string(resJSON))

		// Send metrics via zabbix sender
		packet = zabbix.NewPacket(metrics)
		// ok we got packet for zabbix sender let's send it
		// dataPacket, _ := json.MarshalIndent(packet, "", "  ")
		// fmt.Println(string(dataPacket))
		_, err = z.Send(packet)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Send to zabbix failed: %v", err)
		}
	}

}
