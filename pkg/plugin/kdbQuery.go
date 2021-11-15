package plugin

import kdb "github.com/sv/kdbgo"

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
				return
			}
		case query := <-d.syncQueue:
			res, err := d.kdbHandle.Call(query.query)
			d.syncResChan <- &kdbSyncRes{result: res, err: err, id: query.id}
		}
	}
}
