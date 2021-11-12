package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	kdb "github.com/sv/kdbgo"
)

// Make sure SampleDatasource implements required interfaces. This is important to do
// since otherwise we will only get a not implemented error response from plugin in
// runtime. In this example datasource instance implements backend.QueryDataHandler,
// backend.CheckHealthHandler, backend.StreamHandler interfaces. Plugin should not
// implement all these interfaces - only those which are required for a particular task.
// For example if plugin does not need streaming functionality then you are free to remove
// methods that implement backend.StreamHandler. Implementing instancemgmt.InstanceDisposer
// is useful to clean up resources used by previous datasource instance when a new datasource
// instance created upon datasource settings changed.
var (
	_ backend.QueryDataHandler      = (*KdbDatasource)(nil)
	_ backend.CheckHealthHandler    = (*KdbDatasource)(nil)
	_ instancemgmt.InstanceDisposer = (*KdbDatasource)(nil)
)

var client KdbDatasource

type QueryModel struct {
	QueryText string `json:"queryText"`
	Field     string `json:"field"`
}

type KdbDatasource struct {
	// Host for kdb connection
	Host string `json:"host"`

	// port for kdb connection
	Port int `json:"port"`

	Timeout string `json:"timeout"`

	user string

	pass string

	tlsCertificate string

	tlsKey string

	kdbHandle *kdb.KDBConn
}

// NewKdbDatasource creates a new datasource instance.
func NewKdbDatasource(settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {

	err := json.Unmarshal(settings.JSONData, &client)
	if err != nil {
		log.DefaultLogger.Error("Error decoding Host and Port information", err.Error())
		return nil, err
	}

	username, ok := settings.DecryptedSecureJSONData["username"]
	if !ok {
		log.DefaultLogger.Error("Error - Username property is required")
		return nil, err
	}
	client.user = username

	pass, ok := settings.DecryptedSecureJSONData["password"]
	if !ok {
		log.DefaultLogger.Error("Error - Pass property is required")
		return nil, err
	}
	client.pass = pass
	auth := fmt.Sprintf("%s:%s", client.user, client.pass)

	//TLS and Cert
	tlsCertificate, certOk := settings.DecryptedSecureJSONData["tlsCertificate"]
	if !certOk {
		log.DefaultLogger.Info("Error decoding TLS Cert or no TLS Cert provided")
	}
	client.tlsCertificate = tlsCertificate

	tlsKey, keyOk := settings.DecryptedSecureJSONData["tlsKey"]
	if !keyOk {
		log.DefaultLogger.Error("Error decoding TLS Key or no TLS Key provided")
	}
	client.tlsKey = tlsKey

	if keyOk && certOk {
		//Create TLS connection
	}

	timeOutDuration, _ := time.ParseDuration(client.Timeout + "ms")
	conn, err := kdb.DialKDBTimeout(client.Host, client.Port, auth, timeOutDuration)
	if err != nil {
		log.DefaultLogger.Error("Error establishing kdb connection - %s", err.Error())
		return nil, err
	}

	client.kdbHandle = conn
	return &client, nil
}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created. As soon as datasource settings change detected by SDK old datasource instance will
// be disposed and a new one will be created using NewKdbDatasource factory function.
func (d *KdbDatasource) Dispose() {
	err := d.kdbHandle.Close()
	if err != nil {
		log.DefaultLogger.Error("Error closing KDB connection", err)
	}
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifier).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (d *KdbDatasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	log.DefaultLogger.Info("QueryData called", "request", req)

	// create response struct
	response := backend.NewQueryDataResponse()

	// loop over queries and execute them individually.
	for i, q := range req.Queries {
		res := d.query(ctx, req.PluginContext, q)
		log.DefaultLogger.Info("Query Num")
		log.DefaultLogger.Info(strconv.Itoa(i))
		// save the response in a hashmap
		// based on with RefID as identifier
		response.Responses[q.RefID] = res
	}

	return response, nil
}

func (d *KdbDatasource) query(_ context.Context, pCtx backend.PluginContext, query backend.DataQuery) backend.DataResponse {
	var MyQuery QueryModel

	err := json.Unmarshal(query.JSON, &MyQuery)
	if err != nil {
		log.DefaultLogger.Error("Error decoding query and field -%s", err.Error())

	}
	response := backend.DataResponse{}
	if response.Error != nil {
		return response
	}
	log.DefaultLogger.Info("Line 160")

	kdbResponse, err := d.kdbHandle.Call(MyQuery.QueryText)
	if err != nil {
		log.DefaultLogger.Info(err.Error())

	}

	//table and dicts types here
	frame := data.NewFrame("response")
	log.DefaultLogger.Info("Line 170")
	switch {
	case kdbResponse.Type == kdb.XT:
		kdbTable := kdbResponse.Data.(kdb.Table)

		tabCols := kdbTable.Columns
		tabData := kdbTable.Data

		for colIndex, column := range tabCols {
			frame.Fields = append(frame.Fields, data.NewField(column, nil, tabData[colIndex].Data))
		}

	default:
		e := "returned value of unexpected type, need table"
		log.DefaultLogger.Error(e)
		return response
	}
	log.DefaultLogger.Info("Line 186")
	response.Frames = append(response.Frames, frame)

	return response
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (d *KdbDatasource) CheckHealth(_ context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	log.DefaultLogger.Info("CheckHealth called", "request", req)

	test, err := d.kdbHandle.Call("{1+1}", kdb.Int(2))
	if err != nil {
		log.DefaultLogger.Info(err.Error())
		return nil, err
	}
	var status = backend.HealthStatusError
	var message = ""
	x, _ := test.Data.(int64)

	if x == 2 {
		status = backend.HealthStatusOk
		message = "kdb connected succesfully"

	} else {
		status = backend.HealthStatusError
		message = "kdb connection failed"

	}

	return &backend.CheckHealthResult{
		Status:  status,
		Message: message,
	}, nil
}

// SubscribeStream is called when a client wants to connect to a stream. This callback
// allows sending the first message.
