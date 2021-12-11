package plugin

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type Cfg struct {
	Host      string
	Port      int
	User      string
	Pass      string
	autoStart bool
}

func getConfig() (Cfg, error) {
	f, err := os.Open("../../test/testConfig.csv")
	defer f.Close()
	cfg := Cfg{}
	if err != nil {
		return cfg, err
	}
	data, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return cfg, err
	}
	cfgMap := make(map[string]string)
	requiredCols := []string{"host", "port", "user", "pass", "autoStart"}
	for i, v := range data[0] {
		cfgMap[v] = data[1][i]
	}
	for _, v := range requiredCols {
		pass := false
		for k := range cfgMap {
			if k == v {
				pass = true
				continue
			}
		}
		if !pass {
			return cfg, fmt.Errorf("Missing required column from config CSV: " + v)
		}
	}
	cfg.Host = cfgMap["host"]
	cfg.Port, err = strconv.Atoi(cfgMap["port"])
	if err != nil {
		return cfg, fmt.Errorf("Could not convert test port to int: " + cfgMap["port"])
	}
	cfg.User = cfgMap["user"]
	cfg.Pass = cfgMap["pass"]
	i, err := strconv.Atoi(cfgMap["autoStart"])
	if err != nil {
		return cfg, fmt.Errorf("Could not convert autoStart to int: " + cfgMap["autoStart"])
	}
	cfg.autoStart = i > 0
	return cfg, nil
}

type testServer struct {
	cmd  *exec.Cmd
	auto bool
}

func getConfigAndInit() (*KdbDatasource, *testServer, error) {
	cfg, err := getConfig()
	if err != nil {
		return nil, nil, err
	}
	ds := &KdbDatasource{}
	testSrv := &testServer{}
	if cfg.autoStart {
		testSrv.auto = true
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.Command("..\\..\\test\\testServer.bat", strconv.Itoa(cfg.Port))
		} else {
			cmd = exec.Command("q", "-p "+strconv.Itoa(cfg.Port))
		}
		err = cmd.Start()
		if err != nil {
			return nil, nil, err
		}
		testSrv.cmd = cmd
	}
	ds.Host = cfg.Host
	ds.Port = cfg.Port
	ds.User = cfg.User
	ds.Pass = cfg.Pass
	ds.DialTimeout = time.Duration(time.Second * 5)
	ds.WithTls = false
	return ds, testSrv, err
}

func cleanup(ds *KdbDatasource, testSrv *testServer) {
	ds.KdbHandle.Close()
	if ds.rawReadChan != nil {
		close(ds.rawReadChan)
	}
	if ds.signals != nil {
		close(ds.signals)
	}
	if ds.syncQueue != nil {
		close(ds.syncQueue)
	}
	if ds.syncResChan != nil {
		close(ds.syncResChan)
	}
	if testSrv.auto {
		testSrv.cmd.Process.Kill()
	}
	ds = nil
}

func TestOpenConnectionStd(t *testing.T) {
	// Init
	ds, testSrv, err := getConfigAndInit()
	if err != nil {
		t.Errorf("Error loading config: %v", err)
		return
	}
	t.Logf("kdb+ test server: %s:%v:%s:%s", ds.Host, ds.Port, ds.User, ds.Pass)
	err = ds.openConnection()
	if err != nil {
		t.Errorf("Error opening connection: %v", err)
		return
	}

	// Cleanup
	t.Logf("Finished test, cleaning up...")
	cleanup(ds, testSrv)
	t.Logf("Cleaned up kdb+ test server")
}

func TestCloseConnection(t *testing.T) {
	// Init
	ds, testSrv, err := getConfigAndInit()
	if err != nil {
		t.Errorf("Error loading config: %v", err)
		return
	}
	t.Logf("kdb+ test server: %s:%v:%s:%s", ds.Host, ds.Port, ds.User, ds.Pass)
	err = ds.openConnection()
	if err != nil {
		t.Errorf("Error opening connection: %v", err)
		return
	}
	if !ds.IsOpen {
		t.Errorf("Connection is not assigned as open in KdbDatasource object after calling openConnection")
		cleanup(ds, testSrv)
		return
	}

	err = ds.closeConnection()
	if err != nil {
		t.Errorf("Error calling closeConnection: %v", err)
		cleanup(ds, testSrv)
		return
	}
	if ds.IsOpen {
		t.Errorf("Connection is assigned as open in KdbDatasource object after calling closeConnection")
	}
	cleanup(ds, testSrv)
	return
}

func TestRunKdbQuerySync(t *testing.T) {
	ds := KdbDatasource{}
	log.Print(ds)

}

func TestQueryData(t *testing.T) {
	ds := KdbDatasource{}

	resp, err := ds.QueryData(
		context.Background(),
		&backend.QueryDataRequest{
			Queries: []backend.DataQuery{
				{RefID: "A"},
			},
		},
	)
	if err != nil {
		t.Error(err)
	}

	if len(resp.Responses) != 1 {
		t.Fatal("QueryData must return a response")
	}
}
