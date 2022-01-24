package plugin

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	kdb "github.com/sv/kdbgo"
)

const ADAPTOR_VERSION = float64(1.0)

var (
	_ backend.QueryDataHandler      = (*KdbDatasource)(nil)
	_ backend.CheckHealthHandler    = (*KdbDatasource)(nil)
	_ instancemgmt.InstanceDisposer = (*KdbDatasource)(nil)
)

type QueryModel struct {
	QueryText         string `json:"queryText"`
	Timeout           int    `json:"timeOut"`
	UseTimeColumn     bool   `json:"useTimeColumn"`
	TimeColumn        string `json:"timeColumn"`
	IncludeKeyColumns bool   `json:"includeKeyColumns"`
}

type kdbSyncQuery struct {
	query   *kdb.K
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
	Host                string `json:"host"`
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
	RunKdbQuerySync     func(*kdb.K, time.Duration) (*kdb.K, error)
	OpenConnection      func() error
	CloseConnection     func() error
	WriteConnection     func(kdb.ReqType, *kdb.K) error
	ReadConnection      func() (*kdb.K, kdb.ReqType, error)
}

// NewKdbDatasource creates a new datasource instance.
func NewKdbDatasource(settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {

	client := KdbDatasource{}
	err := json.Unmarshal(settings.JSONData, &client)
	if err != nil {
		log.DefaultLogger.Error("Error decrypting Host and Port information", err.Error())
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
		log.DefaultLogger.Info("TLS enabled for new kdb datasource, creating tls config...")
		tlsCertificate, certOk := settings.DecryptedSecureJSONData["tlsCertificate"]
		if !certOk {
			log.DefaultLogger.Info("Error decrypting TLS Cert or no TLS Cert provided")
		}
		client.TlsCertificate = tlsCertificate

		tlsKey, keyOk := settings.DecryptedSecureJSONData["tlsKey"]
		if !keyOk {
			log.DefaultLogger.Error("Error decrypting TLS Key or no TLS Key provided")
		}
		client.TlsKey = tlsKey

		if client.SkipVertifyTLS {
			log.DefaultLogger.Info("New kdb+ datasource config setup to skip TLS verification")
		}

		if client.WithCACert {
			caCert, keyOk := settings.DecryptedSecureJSONData["caCert"]
			if !keyOk {
				log.DefaultLogger.Error("Error decrypting CA Cert or no CA Cert provided")
			}
			client.CaCert = caCert
			log.DefaultLogger.Info("Setting custom CA certificate...")
			tlsCaCert := x509.NewCertPool()
			r := tlsCaCert.AppendCertsFromPEM([]byte(client.CaCert))
			if !r {
				log.DefaultLogger.Info("Error parsing custom CA certificate")
			}
			tlsServerConfig.RootCAs = tlsCaCert
		}

		cert, err := tls.X509KeyPair([]byte(client.TlsCertificate), []byte(client.TlsKey))
		if err != nil {
			log.DefaultLogger.Error(fmt.Sprintf("Cert convert error %v", err))
		}

		tlsServerConfig.Certificates = []tls.Certificate{cert}
		tlsServerConfig.InsecureSkipVerify = client.SkipVertifyTLS
		client.TlsServerConfig = tlsServerConfig
	}
	timeOutDuration, err := time.ParseDuration(client.Timeout + "ms")
	if nil != err {
		log.DefaultLogger.Info("Using default timeout")
		timeOutDuration = time.Second
	}
	client.DialTimeout = timeOutDuration
	// Set IPC handler functions
	client.setupKdbConnectionHandlers()
	client.IsOpen = false

	// make channel for synchronous queries
	log.DefaultLogger.Info("Making synchronous query channel")
	client.syncQueue = make(chan *kdbSyncQuery)

	// make channel for synchronous responses
	log.DefaultLogger.Info("Making synchronous response channel")
	client.syncResChan = make(chan *kdbSyncRes)

	// making signals channel
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
	}
	d.IsOpen = false
	close(d.rawReadChan)
	return err
}

func (d *KdbDatasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	response := backend.NewQueryDataResponse()

	for _, q := range req.Queries {
		res := d.query(ctx, req.PluginContext, q)
		response.Responses[q.RefID] = res
	}

	return response, nil
}

func (d *KdbDatasource) query(_ context.Context, pCtx backend.PluginContext, query backend.DataQuery) backend.DataResponse {
	var MyQuery QueryModel
	response := backend.DataResponse{}
	err := json.Unmarshal(query.JSON, &MyQuery)
	if err != nil {
		log.DefaultLogger.Error("Error decoding query and field -%s", err.Error())
		response.Error = err
		return response
	}
	if MyQuery.Timeout < 1 {
		MyQuery.Timeout = 10000
	}
	userDict := buildUserKdbDict(pCtx.User)
	datasourceDict := buildDatasourceKdbDict(pCtx.DataSourceInstanceSettings)
	queryDict := buildQueryKdbDict(query, MyQuery.QueryText)
	masterKeys := kdb.SymbolV([]string{"AQUAQ_KDB_BACKEND_GRAF_DATASOURCE", "Time", "OrgID", "Datasource", "User", "Query", "Timeout"})
	masterValues := kdb.NewList(
		kdb.Float(ADAPTOR_VERSION),
		kdb.Atom(-kdb.KP, time.Now()),
		kdb.Long(pCtx.OrgID),
		datasourceDict,
		userDict,
		queryDict,
		kdb.Long(int64(MyQuery.Timeout)))

	kdbResponse, err := d.RunKdbQuerySync(kdb.NewList(kdb.Atom(kdb.KC, "{[x] value x[`Query;`Query]}"), kdb.NewDict(masterKeys, masterValues)), time.Duration(MyQuery.Timeout)*time.Millisecond)
	if err != nil {
		response.Error = err
		return response
	}

	// Parse response data
	switch {
	case kdbResponse.Type == kdb.XT:
		frame, err := ParseSimpleKdbTable(kdbResponse)
		if err != nil {
			response.Error = err
		} else {
			frame.Name = query.RefID
			response.Frames = append(response.Frames, frame)
		}
	case kdbResponse.Type == kdb.XD:
		frames, err := ParseGroupedKdbTable(kdbResponse, MyQuery.IncludeKeyColumns)
		if err != nil {
			response.Error = err
		} else {
			response.Frames = append(response.Frames, frames...)
		}
	default:
		response.Error = fmt.Errorf("Returned object of unsupported type, only tables supported")
	}

	// Handle temporal column override
	if MyQuery.UseTimeColumn {
		for _, frame := range response.Frames {
			timeOverrideIndex := -1
			for v, field := range frame.Fields {
				if field.Name == MyQuery.TimeColumn {
					timeOverrideIndex = v
					break
				}
			}
			if timeOverrideIndex == -1 {
				response.Error = fmt.Errorf("Temporal column override '%v' is not present in all returned tables", MyQuery.TimeColumn)
				return response
			}
			timeCol := frame.Fields[timeOverrideIndex]
			nonTimeCols := append(frame.Fields[:timeOverrideIndex], frame.Fields[timeOverrideIndex+1:]...)
			frame.Fields = append([]*data.Field{timeCol}, nonTimeCols...)
		}
	}
	return response
}

func (d *KdbDatasource) CheckHealth(_ context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	log.DefaultLogger.Info("CheckHealth called", "request", req)
	userDict := buildUserKdbDict(req.PluginContext.User)
	datasourceDict := buildDatasourceKdbDict(req.PluginContext.DataSourceInstanceSettings)
	k := kdb.SymbolV([]string{"AQUAQ_KDB_BACKEND_GRAF_DATASOURCE", "Time", "OrgID", "Datasource", "User", "Query", "Timeout"})
	v := kdb.NewList(
		kdb.Float(ADAPTOR_VERSION),
		kdb.Atom(-kdb.KP, time.Now()),
		kdb.Long(req.PluginContext.OrgID),
		datasourceDict,
		userDict,
		kdb.NewDict(kdb.SymbolV([]string{"Query", "QueryType"}), kdb.NewList(kdb.Atom(kdb.KC, "1+1"), kdb.Symbol("HEALTHCHECK"))),
		kdb.Long(int64(d.DialTimeout)))

	test, err := d.RunKdbQuerySync(kdb.NewList(kdb.Atom(kdb.KC, "{[x] value x[`Query;`Query]}"), kdb.NewDict(k, v)), d.DialTimeout)
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
