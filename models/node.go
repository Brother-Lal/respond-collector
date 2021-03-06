package models

import (
	"github.com/FreifunkBremen/respond-collector/data"
	"github.com/FreifunkBremen/respond-collector/jsontime"
	"github.com/FreifunkBremen/respond-collector/meshviewer"
	imodels "github.com/influxdata/influxdb/models"
	"strconv"
)

// Node struct
type Node struct {
	Firstseen  jsontime.Time    `json:"firstseen"`
	Lastseen   jsontime.Time    `json:"lastseen"`
	Flags      meshviewer.Flags `json:"flags"`
	Statistics *data.Statistics `json:"statistics"`
	Nodeinfo   *data.NodeInfo   `json:"nodeinfo"`
	Neighbours *data.Neighbours `json:"-"`
}

// Returns tags and fields for InfluxDB
func (node *Node) ToInflux() (tags imodels.Tags, fields imodels.Fields) {
	stats := node.Statistics

	tags.SetString("nodeid", stats.NodeId)

	fields = map[string]interface{}{
		"load":           stats.LoadAverage,
		"time.up":        int64(stats.Uptime),
		"time.idle":      int64(stats.Idletime),
		"proc.running":   stats.Processes.Running,
		"clients.wifi":   stats.Clients.Wifi,
		"clients.wifi24": stats.Clients.Wifi24,
		"clients.wifi5":  stats.Clients.Wifi5,
		"clients.total":  stats.Clients.Total,
		"memory.buffers": stats.Memory.Buffers,
		"memory.cached":  stats.Memory.Cached,
		"memory.free":    stats.Memory.Free,
		"memory.total":   stats.Memory.Total,
	}

	if nodeinfo := node.Nodeinfo; nodeinfo != nil {
		if owner := nodeinfo.Owner; owner != nil {
			tags.SetString("owner", owner.Contact)
		}
		if wireless := nodeinfo.Wireless; wireless != nil {
			fields["wireless.txpower24"] = wireless.TxPower24
			fields["wireless.txpower5"] = wireless.TxPower5
		}
		// morpheus needs
		tags.SetString("hostname", nodeinfo.Hostname)
	}

	if t := stats.Traffic.Rx; t != nil {
		fields["traffic.rx.bytes"] = int64(t.Bytes)
		fields["traffic.rx.packets"] = t.Packets
	}
	if t := stats.Traffic.Tx; t != nil {
		fields["traffic.tx.bytes"] = int64(t.Bytes)
		fields["traffic.tx.packets"] = t.Packets
		fields["traffic.tx.dropped"] = t.Dropped
	}
	if t := stats.Traffic.Forward; t != nil {
		fields["traffic.forward.bytes"] = int64(t.Bytes)
		fields["traffic.forward.packets"] = t.Packets
	}
	if t := stats.Traffic.MgmtRx; t != nil {
		fields["traffic.mgmt_rx.bytes"] = int64(t.Bytes)
		fields["traffic.mgmt_rx.packets"] = t.Packets
	}
	if t := stats.Traffic.MgmtTx; t != nil {
		fields["traffic.mgmt_tx.bytes"] = int64(t.Bytes)
		fields["traffic.mgmt_tx.packets"] = t.Packets
	}

	for _, airtime := range stats.Wireless {
		suffix := airtime.FrequencyName()
		fields["airtime"+suffix+".chan_util"] = airtime.ChanUtil
		fields["airtime"+suffix+".rx_util"] = airtime.RxUtil
		fields["airtime"+suffix+".tx_util"] = airtime.TxUtil
		fields["airtime"+suffix+".noise"] = airtime.Noise
		fields["airtime"+suffix+".frequency"] = airtime.Frequency
		tags.SetString("frequency"+suffix, strconv.Itoa(int(airtime.Frequency)))
	}

	return
}
