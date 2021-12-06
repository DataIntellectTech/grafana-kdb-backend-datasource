package plugin

import (
	"fmt"
	"io"
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
				close(d.syncQueue)
				close(d.syncResChan)
				return
			}
		case query := <-d.syncQueue:
			log.DefaultLogger.Info("QUERY ENTER CHANNEL")
			var err error
			// If handle isn't open, attempt to open
			if !d.IsOpen {
				err = d.openConnection()
				// Return error if unable to open handle
				if err != nil {
					log.DefaultLogger.Info("UNABLE TO OPEN HANDLE, RET1")
					d.syncResChan <- &kdbSyncRes{result: nil, err: err, id: query.id}
					continue
				}
			}
			// If handle is open, query the kdb+ process
			var kdbQueryObj = &kdb.K{Type: kdb.KC, Attr: kdb.NONE, Data: query.query}
			err = d.kdbHandle.WriteMessage(kdb.SYNC, kdbQueryObj)
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
				d.syncResChan <- &kdbSyncRes{result: nil, err: fmt.Errorf("Queried timed out after %v miliseconds", query.timeout), id: query.id}
				d.closeConnection()
			}
		}
	}

}
func (d *KdbDatasource) kdbHandleListener() {
	for {
		log.DefaultLogger.Info("READING HANDLE RESPONSE")
		res, _, err := d.kdbHandle.ReadMessage()
		log.DefaultLogger.Info("RESPONSE RECEIVED")
		d.rawReadChan <- &kdbRawRead{result: res, err: err}
		if err == io.EOF {
			log.DefaultLogger.Info("Handle closing, returning from kdbHandleListener")
			d.IsOpen = false
			close(d.rawReadChan)
			d.signals <- 3
			return
		}
	}
}
