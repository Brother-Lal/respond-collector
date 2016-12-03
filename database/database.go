package database

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/FreifunkBremen/respond-collector/models"
	"github.com/influxdata/influxdb/client/v2"
	imodels "github.com/influxdata/influxdb/models"
)

const (
	MeasurementNode   = "node"   // Measurement for per-node statistics
	MeasurementGlobal = "global" // Measurement for summarized global statistics
	batchMaxSize      = 500
	batchDuration     = time.Second * 5
)

type DB struct {
	config *models.Config
	client client.Client
	points chan *client.Point
	wg     sync.WaitGroup
	quit   chan struct{}
}

func New(config *models.Config) *DB {
	// Make client
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     config.Influxdb.Addr,
		Username: config.Influxdb.Username,
		Password: config.Influxdb.Password,
	})

	if err != nil {
		panic(err)
	}

	db := &DB{
		config: config,
		client: c,
		points: make(chan *client.Point, 1000),
		quit:   make(chan struct{}),
	}

	db.wg.Add(1)
	go db.addWorker()

	if db.config.Influxdb.DeleteInterval > 0 {
		go db.deleteWorker()
	}

	return db
}

func (db *DB) DeletePoints() {
	query := fmt.Sprintf("delete from %s where time < now() - %dm", MeasurementNode, db.config.Influxdb.DeleteAfter)
	log.Println("delete", MeasurementNode, "older than", db.config.Influxdb.DeleteAfter, "minutes")
	db.client.Query(client.NewQuery(query, db.config.Influxdb.Database, "m"))
}

func (db *DB) AddPoint(name string, tags imodels.Tags, fields imodels.Fields, time time.Time) {
	point, err := client.NewPoint(name, tags.Map(), fields, time)
	if err != nil {
		panic(err)
	}
	db.points <- point
}

// Add data for a single node
func (db *DB) Add(nodeId string, node *models.Node) {
	tags, fields := node.ToInflux()
	db.AddPoint(MeasurementNode, tags, fields, time.Now())
}

func (db *DB) Close() {
	close(db.quit)
	close(db.points)
	db.wg.Wait()
	db.client.Close()
}

func (db *DB) deleteWorker() {
	duration := time.Second * time.Duration(db.config.Influxdb.DeleteInterval)
	ticker := time.NewTicker(duration)
	for {
		select {
		case <-ticker.C:
			db.DeletePoints()
		case <-db.quit:
			ticker.Stop()
			return
		}
	}
}

// stores data points in batches into the influxdb
func (db *DB) addWorker() {
	bpConfig := client.BatchPointsConfig{
		Database:  db.config.Influxdb.Database,
		Precision: "m",
	}

	var bp client.BatchPoints
	var err error
	var writeNow, closed bool
	timer := time.NewTimer(batchDuration)

	for !closed {
		// wait for new points
		select {
		case point, ok := <-db.points:
			if ok {
				if bp == nil {
					// create new batch
					timer.Reset(batchDuration)
					if bp, err = client.NewBatchPoints(bpConfig); err != nil {
						log.Fatal(err)
					}
				}
				bp.AddPoint(point)
			} else {
				closed = true
			}
		case <-timer.C:
			if bp == nil {
				timer.Reset(batchDuration)
			} else {
				writeNow = true
			}
		}

		// write batch now?
		if bp != nil && (writeNow || closed || len(bp.Points()) >= batchMaxSize) {
			log.Println("saving", len(bp.Points()), "points")

			if err = db.client.Write(bp); err != nil {
				log.Fatal(err)
			}
			writeNow = false
			bp = nil
		}
	}
	timer.Stop()
	db.wg.Done()
}
