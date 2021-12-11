package plugin

import (
	"fmt"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	kdb "github.com/sv/kdbgo"
)

// support maximum queue of 100 000 per handle
func (d *KdbDatasource) getKdbSyncQueryId() uint32 {
	if d.kdbSyncQueryCounter > 100000 {
		d.kdbSyncQueryCounter = 0
	}
	d.kdbSyncQueryCounter += 1
	return d.kdbSyncQueryCounter
}

func (d *KdbDatasource) runKdbQuerySync(query string, timeout time.Duration) (*kdb.K, error) {
	queryObj := &kdbSyncQuery{query: query, id: d.getKdbSyncQueryId(), timeout: timeout}
	d.syncQueue <- queryObj
	for {
		res := <-d.syncResChan
		if res.id != queryObj.id {
			continue
		}
		return res.result, res.err
	}
}

func (d *KdbDatasource) syncQueryRunner() {
	for {
		select {
		case signal := <-d.signals:
			if signal == 3 {
				log.DefaultLogger.Info("Returning from query runner")
				return
			}
		case query := <-d.syncQueue:
			var err error
			// If handle isn't open, attempt to open
			if !d.IsOpen {
				log.DefaultLogger.Info("Handle not open, opening new handle...")
				err = d.OpenConnection()
				// Return error if unable to open handle
				if err != nil {
					log.DefaultLogger.Info("Unable to open handle on-demand in syncQueryRunner")
					d.syncResChan <- &kdbSyncRes{result: nil, err: err, id: query.id}
					continue
				}
			}
			// If handle is open, query the kdb+ process
			var kdbQueryObj = &kdb.K{Type: kdb.KC, Attr: kdb.NONE, Data: query.query}
			err = d.WriteConnection(kdb.SYNC, kdbQueryObj)
			if err != nil {
				log.DefaultLogger.Error("Error writing message", err.Error())
				d.syncResChan <- &kdbSyncRes{result: nil, err: err, id: query.id}
				continue
			}

			select {
			case msg := <-d.rawReadChan:
				log.DefaultLogger.Info("RECEIVED RESULT, RET2")
				d.syncResChan <- &kdbSyncRes{result: msg.result, err: msg.err, id: query.id}
			case <-time.After(query.timeout):
				log.DefaultLogger.Info("QUERY TIMEOUT, RET3")
				d.syncResChan <- &kdbSyncRes{result: nil, err: fmt.Errorf("Queried timed out after %v", query.timeout), id: query.id}
				d.CloseConnection()
			}
		}
	}
}

func (d *KdbDatasource) kdbHandleListener() {
	kdbEOF := "Failed to read message header:"
	for {
		res, _, err := d.ReadConnection()
		if err != nil {
			log.DefaultLogger.Info(err.Error())
			if strings.Contains(err.Error(), kdbEOF) {
				log.DefaultLogger.Info("Handle read error, publishing error and returning from kdbHandleListener")
				d.IsOpen = false
				d.rawReadChan <- &kdbRawRead{result: res, err: err}
				close(d.rawReadChan)
				return
			}
		}
		d.rawReadChan <- &kdbRawRead{result: res, err: err}
	}
}
