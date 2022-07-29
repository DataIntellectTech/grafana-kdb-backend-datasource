package plugin

import (
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

func (d *KdbDatasource) setupKdbConnectionHandlers() {
	log.DefaultLogger.Debug("Setting kdb+ connection handlers...")
	d.KdbHandleListener = d.kdbHandleListener
	d.RunKdbQuerySync = d.runKdbQuerySync
	d.OpenConnection = d.openConnection
	d.CloseConnection = d.closeConnection
	d.WriteConnection = d.writeMessage
	d.ReadConnection = d.readMessage
}
