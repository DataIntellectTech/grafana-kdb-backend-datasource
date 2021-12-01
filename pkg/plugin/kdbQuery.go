package plugin

import (
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

func (d *KdbDatasource) runKdbQuerySync(query string) (*kdb.K, error) {
	queryObj := &kdbSyncQuery{query: query, id: d.getKdbSyncQueryId()}
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

			var OurQuery = &kdb.K{kdb.KC, 0, query.query}
			err := d.kdbHandle.WriteMessage(1, OurQuery)
			if err != nil {
				log.DefaultLogger.Error("Error writing message", err.Error())
			}

			resdata, err := d.kdbHandleListener()
			if err != nil {
				log.DefaultLogger.Error("Error writing message", err.Error())
			}
			//res, err := d.kdbHandle.Call(query.query)
			d.syncResChan <- &kdbSyncRes{result: resdata, err: err, id: query.id}
		}
	}

}
func (d *KdbDatasource) kdbHandleListener() (*kdb.K, error) {

	res, _, err := d.kdbHandle.ReadMessage()

	if err != nil {
		return nil, err
	}
	return res, nil

}
