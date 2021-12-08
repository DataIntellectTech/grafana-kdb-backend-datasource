package plugin

import (
	"context"
	"crypto/tls"
	"crypto/x509"
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

type QueryModel struct {
	QueryText string `json:"queryText"`
	Timeout   int    `json:"timeOut"`
}

type kdbSyncQuery struct {
	query   string
	id      uint32
	timeout time.Duration
}

type kdbRawRead struct {
	result *kdb.K
	err    error
}

type kdbSyncRes struct {
	result *kdb.K
	err    error
	id     uint32
}

type KdbDatasource struct {
	// Host for kdb connection
	Host string `json:"host"`
	// port for kdb connection
	Port                int    `json:"port"`
	Timeout             string `json:"timeout"`
	WithTls             bool   `json:"withTLS"`
	SkipVertifyTLS      bool   `json:"skipVerifyTLS"`
	WithCACert          bool   `json:"withCACert"`
	user                string
	pass                string
	tlsCertificate      string
	tlsKey              string
	caCert              string
	tlsServerConfig     *tls.Config
	dialTimeout         time.Duration
	kdbHandle           *kdb.KDBConn
	signals             chan int
	syncQueue           chan *kdbSyncQuery
	rawReadChan         chan *kdbRawRead
	syncResChan         chan *kdbSyncRes
	kdbSyncQueryCounter uint32
	IsOpen              bool
}

// NewKdbDatasource creates a new datasource instance.
func NewKdbDatasource(settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	log.DefaultLogger.Info(string(settings.JSONData))

	client := KdbDatasource{}
	err := json.Unmarshal(settings.JSONData, &client)
	if err != nil {
		log.DefaultLogger.Error("Error decoding Host and Port information", err.Error())
		return nil, err
	}

	username, ok := settings.DecryptedSecureJSONData["username"]
	if ok {
		client.user = username
	} else {
		client.user = ""
		log.DefaultLogger.Info("No username provided; using default")

	}

	pass, ok := settings.DecryptedSecureJSONData["password"]
	if ok {
		client.pass = pass
	} else {
		client.pass = ""
		log.DefaultLogger.Info("No password provided; using default")
	}

	if client.WithTls {
		tlsServerConfig := new(tls.Config)
		log.DefaultLogger.Info("=========USING TLS==========")
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

		if client.SkipVertifyTLS {
			log.DefaultLogger.Info("-------HANDLE SKIP VERT-------")
		}

		if client.WithCACert {
			caCert, keyOk := settings.DecryptedSecureJSONData["caCert"]
			if !keyOk {
				log.DefaultLogger.Error("Error decoding CA Cert or no CA Cert provided")
			}
			client.caCert = caCert
			log.DefaultLogger.Info("-------HANDLE CA CERT-------")
			tlsCaCert := x509.NewCertPool()
			tlsCaCert.AppendCertsFromPEM([]byte(client.caCert))
			tlsServerConfig.ClientCAs = tlsCaCert
		}

		cert, err := tls.X509KeyPair([]byte(client.tlsCertificate), []byte(client.tlsKey))
		if err != nil {
			log.DefaultLogger.Error(fmt.Sprintf("Cert convert error %v", err))
		}

		tlsServerConfig.Certificates = []tls.Certificate{cert}
		tlsServerConfig.InsecureSkipVerify = client.SkipVertifyTLS
		client.tlsServerConfig = tlsServerConfig
	} else {
		log.DefaultLogger.Info("=========No TLS==========")
		timeOutDuration, err := time.ParseDuration(client.Timeout + "ms")
		if nil != err {
			log.DefaultLogger.Info("Using default timeout")
			timeOutDuration = time.Second
		}
		client.dialTimeout = timeOutDuration
	}

	// make channel for synchronous queries
	log.DefaultLogger.Info("Making synchronous query channel")
	client.syncQueue = make(chan *kdbSyncQuery)
	// make channel for synchronous responses
	log.DefaultLogger.Info("Making synchronous response channel")
	client.syncResChan = make(chan *kdbSyncRes)

	// Open the kdb Handle
	err = client.openConnection()
	if err != nil {
		log.DefaultLogger.Error(fmt.Sprintf("Error opening handle to kdb+ process when creating datasource: %v", err))
	}
	// start synchronous query listener
	log.DefaultLogger.Info("Beginning synchronous query listener")
	go client.syncQueryRunner()
	// making signals channel (this should be done through ctx)
	log.DefaultLogger.Info("Making signals channel")
	client.signals = make(chan int)

	log.DefaultLogger.Info("KDB Datasource created successfully")
	return &client, nil
}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created. As soon as datasource settings change detected by SDK old datasource instance will
// be disposed and a new one will be created using NewKdbDatasource factory function.
func (d *KdbDatasource) Dispose() {
	log.DefaultLogger.Info("DEVDISPOSE Dispose called")
	if d.IsOpen {
		log.DefaultLogger.Info("DEVDISPOSE Handle open when dispose called, closing handle")
		err := d.closeConnection()
		if err != nil {
			log.DefaultLogger.Error("DEVDISPOSE Error closing KDB connection", err)
		}
	}
	d.signals <- 3
	close(d.signals)
	close(d.syncQueue)
	close(d.syncResChan)
}

func (d *KdbDatasource) openConnection() error {
	log.DefaultLogger.Info("DEVOPENCONNECTION called")
	auth := fmt.Sprintf("%s:%s", d.user, d.pass)
	var conn *kdb.KDBConn = nil
	var err error
	if d.WithTls {
		conn, err = kdb.DialTLS(d.Host, d.Port, auth, d.tlsServerConfig)
	} else {
		conn, err = kdb.DialKDBTimeout(d.Host, d.Port, auth, d.dialTimeout)
	}
	if err != nil {
		log.DefaultLogger.Error("Error establishing kdb connection - %s", err.Error())
		d.kdbHandle = nil
		return err
	}
	log.DefaultLogger.Info(fmt.Sprintf("Dialled %v:%v successfully", d.Host, d.Port))
	d.kdbHandle = conn
	d.IsOpen = true
	// making raw read channel
	log.DefaultLogger.Info("Making raw response channel")
	d.rawReadChan = make(chan *kdbRawRead)

	// start synchronous handle reader
	log.DefaultLogger.Info("Beginning handle listener")
	go d.kdbHandleListener()
	return nil
}

func (d *KdbDatasource) closeConnection() error {
	log.DefaultLogger.Info("DEVCLOSECONNECTION called")
	err := d.kdbHandle.Close()
	log.DefaultLogger.Info("DEVCLOSECONNECTION closed handle")
	if err == nil {
		log.DefaultLogger.Info("DEVCLOSECONNECTION error closing handle")
		d.IsOpen = false

	}
	log.DefaultLogger.Info("DEVCLOSECONNECTION returning")
	return err
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifier).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (d *KdbDatasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	log.DefaultLogger.Info("QueryData called", "request", req)
	log.DefaultLogger.Info(fmt.Sprintf("datasource %v", d.Host))
	log.DefaultLogger.Info(fmt.Sprintf("datasource %v", d.Port))
	log.DefaultLogger.Info(fmt.Sprintf("datasource %v", d.kdbHandle))
	// create response struct
	response := backend.NewQueryDataResponse()

	// loop over queries and execute them individually.
	for i, q := range req.Queries {
		res := d.query(ctx, req.PluginContext, q)
		log.DefaultLogger.Info(strconv.Itoa(i))
		// save the response in a hashmap
		// based on with RefID as identifier
		response.Responses[q.RefID] = res
	}

	return response, nil
}

func (d *KdbDatasource) query(_ context.Context, pCtx backend.PluginContext, query backend.DataQuery) backend.DataResponse {
	var MyQuery QueryModel
	log.DefaultLogger.Info(string(query.JSON))
	log.DefaultLogger.Info("DEVQUERY1 Unmarshalling JSON")
	err := json.Unmarshal(query.JSON, &MyQuery)
	if err != nil {
		log.DefaultLogger.Error("Error decoding query and field -%s", err.Error())

	}
	response := backend.DataResponse{}
	log.DefaultLogger.Info(fmt.Sprintf("DEVQUERY2 Interpreting timeout: %v", MyQuery.Timeout))

	if err != nil {
		log.DefaultLogger.Info(err.Error())
		response.Error = err
		return response
	}
	log.DefaultLogger.Info("DEVQUERY3 Running query against kdb+ process: ")
	if MyQuery.Timeout < 1 {
		MyQuery.Timeout = 10000
	}
	log.DefaultLogger.Info(strconv.Itoa(MyQuery.Timeout))

	kdbResponse, err := d.runKdbQuerySync(MyQuery.QueryText, time.Duration(MyQuery.Timeout)*time.Millisecond)
	if err != nil {
		response.Error = err
		return response

	}
	log.DefaultLogger.Info("DEVQUERY4 Received response from kdb+")

	//table and dicts types here
	frame := data.NewFrame("response")
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

	test, err := d.runKdbQuerySync("1+1", time.Duration(d.dialTimeout)*time.Millisecond)
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
