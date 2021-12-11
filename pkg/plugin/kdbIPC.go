package plugin

import (
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

func (d *KdbDatasource) setupKdbConnectionHandlers() {
	log.DefaultLogger.Info("Setting kdb+ connection handlers...")
	d.KdbHandleListener = d.kdbHandleListener
	d.OpenConnection = d.openConnection
	d.CloseConnection = d.closeConnection
	d.WriteConnection = d.KdbHandle.WriteMessage
	d.ReadConnection = d.KdbHandle.ReadMessage
}
