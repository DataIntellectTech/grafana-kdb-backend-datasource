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
	TlsCertificate      string
	TlsKey              string
	CaCert              string
	TlsServerConfig     *tls.Config
	DialTimeout         time.Duration
	KdbHandle           *kdb.KDBConn
	signals             chan int
	syncQueue           chan *kdbSyncQuery
	rawReadChan         chan *kdbRawRead
	syncResChan         chan *kdbSyncRes
	kdbSyncQueryCounter uint32
	IsOpen              bool
	KdbHandleListener   func()
	RunKdbQuerySync     func(string, time.Duration) (*kdb.K, error)
	OpenConnection      func() error
	CloseConnection     func() error
	WriteConnection     func(kdb.ReqType, *kdb.K) error
	ReadConnection      func() (*kdb.K, kdb.ReqType, error)
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
		client.TlsCertificate = tlsCertificate

		tlsKey, keyOk := settings.DecryptedSecureJSONData["tlsKey"]
		if !keyOk {
			log.DefaultLogger.Error("Error decoding TLS Key or no TLS Key provided")
		}
		client.TlsKey = tlsKey

		if client.SkipVertifyTLS {
			log.DefaultLogger.Info("-------HANDLE SKIP VERT-------")
		}

		if client.WithCACert {
			caCert, keyOk := settings.DecryptedSecureJSONData["caCert"]
			if !keyOk {
				log.DefaultLogger.Error("Error decoding CA Cert or no CA Cert provided")
			}
			client.CaCert = caCert
			log.DefaultLogger.Info("-------HANDLE CA CERT-------")
			tlsCaCert := x509.NewCertPool()
			tlsCaCert.AppendCertsFromPEM([]byte(client.CaCert))
			tlsServerConfig.ClientCAs = tlsCaCert
		}

		cert, err := tls.X509KeyPair([]byte(client.TlsCertificate), []byte(client.TlsKey))
		if err != nil {
			log.DefaultLogger.Error(fmt.Sprintf("Cert convert error %v", err))
		}

		tlsServerConfig.Certificates = []tls.Certificate{cert}
		tlsServerConfig.InsecureSkipVerify = client.SkipVertifyTLS
		client.TlsServerConfig = tlsServerConfig
	} else {
		log.DefaultLogger.Info("=========No TLS==========")
		timeOutDuration, err := time.ParseDuration(client.Timeout + "ms")
		if nil != err {
			log.DefaultLogger.Info("Using default timeout")
			timeOutDuration = time.Second
		}
		client.DialTimeout = timeOutDuration
	}
	// Set IPC handler functions
	client.setupKdbConnectionHandlers()
	client.IsOpen = false

	// make channel for synchronous queries
	log.DefaultLogger.Info("Making synchronous query channel")
	client.syncQueue = make(chan *kdbSyncQuery)

	// make channel for synchronous responses
	log.DefaultLogger.Info("Making synchronous response channel")
	client.syncResChan = make(chan *kdbSyncRes)

	// making signals channel (this should be done through ctx)
	log.DefaultLogger.Info("Making signals channel")
	client.signals = make(chan int)

	// Open the kdb Handle
	err = client.OpenConnection()
	if err != nil {
		log.DefaultLogger.Error(fmt.Sprintf("Error opening handle to kdb+ process when creating datasource: %v", err))
	}

	// start synchronous query listener
	go client.syncQueryRunner()

	log.DefaultLogger.Info("KDB Datasource created successfully")
	return &client, nil
}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created. As soon as datasource settings change detected by SDK old datasource instance will
// be disposed and a new one will be created using NewKdbDatasource factory function.
func (d *KdbDatasource) Dispose() {
	log.DefaultLogger.Info("Dispose called")
	if d.IsOpen {
		log.DefaultLogger.Info("Handle open when dispose called, closing handle")
		err := d.CloseConnection()
		if err != nil {
			log.DefaultLogger.Error("Error closing KDB connection", err)
		}
	}
	d.signals <- 3
	close(d.signals)
	close(d.syncQueue)
	close(d.syncResChan)
}

func (d *KdbDatasource) openConnection() error {
	log.DefaultLogger.Info(fmt.Sprintf("Opening connection to %s:%v ...", d.Host, d.Port))
	auth := fmt.Sprintf("%s:%s", d.user, d.pass)
	var conn *kdb.KDBConn = nil
	var err error
	if d.WithTls {
		conn, err = kdb.DialTLS(d.Host, d.Port, auth, d.TlsServerConfig)
	} else {
		conn, err = kdb.DialKDBTimeout(d.Host, d.Port, auth, d.DialTimeout)
	}
	if err != nil {
		log.DefaultLogger.Error(fmt.Sprintf("Error establishing kdb connection - %s", err.Error()))
		d.KdbHandle = nil
		return err
	}
	log.DefaultLogger.Info(fmt.Sprintf("Dialled %s:%v successfully", d.Host, d.Port))
	d.KdbHandle = conn
	d.IsOpen = true
	// making raw read channel
	log.DefaultLogger.Info("Making raw response channel")
	d.rawReadChan = make(chan *kdbRawRead)

	// start synchronous handle reader
	log.DefaultLogger.Info("Beginning handle listener")
	go d.KdbHandleListener()
	return nil
}

func (d *KdbDatasource) closeConnection() error {
	log.DefaultLogger.Info(fmt.Sprintf("Closing connection to %s:%v ...", d.Host, d.Port))
	err := d.KdbHandle.Close()
	if err == nil {
		log.DefaultLogger.Error(fmt.Sprintf("Error closing handle to %s:%v ...", d.Host, d.Port))
		d.IsOpen = false
	}
	return err
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifier).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (d *KdbDatasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
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
	err := json.Unmarshal(query.JSON, &MyQuery)
	if err != nil {
		log.DefaultLogger.Error("Error decoding query and field -%s", err.Error())

	}
	response := backend.DataResponse{}

	if err != nil {
		log.DefaultLogger.Info(err.Error())
		response.Error = err
		return response
	}

	if MyQuery.Timeout < 1 {
		MyQuery.Timeout = 10000
	}
	log.DefaultLogger.Info(strconv.Itoa(MyQuery.Timeout))

	kdbResponse, err := d.RunKdbQuerySync(MyQuery.QueryText, time.Duration(MyQuery.Timeout)*time.Millisecond)
	if err != nil {
		response.Error = err
		return response

	}

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

	response.Frames = append(response.Frames, frame)
	return response
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (d *KdbDatasource) CheckHealth(_ context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	log.DefaultLogger.Info("CheckHealth called", "request", req)

	test, err := d.RunKdbQuerySync("1+1", time.Duration(d.DialTimeout)*time.Millisecond)
	if err != nil {
		log.DefaultLogger.Error("CheckHealth error: %v", err)
		return &backend.CheckHealthResult{Status: backend.HealthStatusError, Message: fmt.Sprintf("Error querying kdb+ process: %v", err)}, nil
	}
	var status = backend.HealthStatusUnknown
	var message = ""

	// Test response type
	if test.Type != -kdb.KJ {
		status = backend.HealthStatusError
		message = fmt.Sprintf("kdb+ result not of expected type; received type %v", test.Type)
		log.DefaultLogger.Info(fmt.Sprintf("Response from kdb+ incorrect type. Received object: %v", test.Data))
		return &backend.CheckHealthResult{
			Status:  status,
			Message: message,
		}, nil
	}
	// Type assert response
	val := test.Data.(int64)

	if val == 2 {
		status = backend.HealthStatusOk
		message = "kdb+ connected succesfully"

	} else {
		status = backend.HealthStatusError
		message = fmt.Sprintf("kdb+ response to \"1+1\" was correct type but incorrect value (returned %v)", val)

	}

	return &backend.CheckHealthResult{
		Status:  status,
		Message: message,
	}, nil
}

// SubscribeStream is called when a client wants to connect to a stream. This callback
// allows sending the first message.
