package plugin

import (
	"fmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	uuid "github.com/nu7hatch/gouuid"
	kdb "github.com/sv/kdbgo"
)

func guidParser(data *kdb.K) []string {
	uuidArr := data.Data.([]uuid.UUID)
	guidArr := make([]string, len(uuidArr))
	for i, entry := range uuidArr {
		guidArr[i] = entry.String()
	}
	return guidArr
}

func charParser(data *kdb.K) []string {
	byteArray := make([]string, data.Len())
	for i := 0; i < data.Len(); i++ {
		byteArray[i] = string(data.Index(i).(byte))
	}
	return byteArray
}

func stringParser(data *kdb.K) ([]string, error) {
	stringCol := data.Data.([]*kdb.K)
	stringArray := make([]string, data.Len())
	for i, word := range stringCol {
		if word.Type != kdb.KC {
			return nil, fmt.Errorf("A column is present which is neither a vector nor a string column. kdb+ type at index %v: %v", i, word.Type)
		}
		stringArray[i] = word.Data.(string)
	}
	return stringArray, nil
}

func ParseSimpleKdbTable(res *kdb.K) (*data.Frame, error) {
	frame := data.NewFrame("response")
	kdbTable := res.Data.(kdb.Table)
	tabData := kdbTable.Data

	for colIndex, columnName := range kdbTable.Columns {
		log.DefaultLogger.Info(strconv.Itoa(int(tabData[colIndex].Type)))
		switch {
		case tabData[colIndex].Type == kdb.K0:
			stringColumn, err := stringParser(tabData[colIndex])
			if err != nil {
				return nil, fmt.Errorf("The following column: %v return this error: %v", columnName, err)
			}
			frame.Fields = append(frame.Fields, data.NewField(columnName, nil, stringColumn))
		case tabData[colIndex].Type == kdb.KC:
			frame.Fields = append(frame.Fields, data.NewField(columnName, nil, charParser(tabData[colIndex])))
		case tabData[colIndex].Type == kdb.KN:
			durArr := tabData[colIndex].Data.([]time.Duration)
			durIntArr := make([]int64, len(durArr))
			for i, dur := range durArr {
				durIntArr[i] = int64(dur)
			}
			frame.Fields = append(frame.Fields, data.NewField(columnName, nil, durIntArr))

		case tabData[colIndex].Type == kdb.KT:
			//Time
			kdbTimeArr := tabData[colIndex].Data.([]kdb.Time)
			timeArr := make([]time.Time, len(kdbTimeArr))
			for index, t := range kdbTimeArr {
				timeArr[index] = time.Time(t)
			}
			frame.Fields = append(frame.Fields, data.NewField(columnName, nil, timeArr))

		case tabData[colIndex].Type == kdb.UU:
			//GUID
			frame.Fields = append(frame.Fields, data.NewField(columnName, nil, guidParser(tabData[colIndex])))
		case tabData[colIndex].Type == kdb.KU:
			//Minute
			minArr := tabData[colIndex].Data.([]kdb.Minute)
			minTimeArr := make([]time.Time, len(minArr))
			for index, min := range minArr {
				minTimeArr[index] = time.Time(min)
			}
			frame.Fields = append(frame.Fields, data.NewField(columnName, nil, minTimeArr))
		case tabData[colIndex].Type == kdb.KV:
			//Second
			secArr := tabData[colIndex].Data.([]kdb.Second)
			secTimeArr := make([]time.Time, len(secArr))
			for index, sec := range secArr {
				secTimeArr[index] = time.Time(sec)
			}
			frame.Fields = append(frame.Fields, data.NewField(columnName, nil, secTimeArr))
		case tabData[colIndex].Type == kdb.KM:
			// Month
			monthArr := tabData[colIndex].Data.([]kdb.Month)
			monthIntArr := make([]int32, len(monthArr))
			for index, val := range monthArr {
				monthIntArr[index] = int32(val)
			}
			frame.Fields = append(frame.Fields, data.NewField(columnName, nil, monthIntArr))

		default:
			frame.Fields = append(frame.Fields, data.NewField(columnName, nil, tabData[colIndex].Data))
		}

	}
	return frame, nil
}

func ParseGroupedKdbTable(res *kdb.K, includeKeys bool) ([]*data.Frame, error) {
	kdbDict := res.Data.(kdb.Dict)
	if kdbDict.Key.Type != kdb.XT || kdbDict.Value.Type != kdb.XT {
		return nil, fmt.Errorf("Either the key or the value of the returned dictionary object is not a table of type 98.")
	}
	rc := kdbDict.Key.Len()
	valData := kdbDict.Value.Data.(kdb.Table)
	frameArray := make([]*data.Frame, rc)
	k := kdbDict.Key.Data.(kdb.Table)
	keyColCount := len(k.Columns)

	for row := 0; row < rc; row++ {
		keyData := correctedTableIndex(k, row)
		frameName := parseFrameName(keyData.Value)
		frame := data.NewFrame(frameName)
		rowData := correctedTableIndex(valData, row)
		depth, err := getDepth(rowData.Value.Data.([]*kdb.K))
		if err != nil {
			return nil, err
		}
		var masterCols []string
		var masterData []*kdb.K
		if includeKeys {
			masterCols = append(keyData.Key.Data.([]string), rowData.Key.Data.([]string)...)
			masterData = append(keyData.Value.Data.([]*kdb.K), rowData.Value.Data.([]*kdb.K)...)
		} else {
			masterCols = rowData.Key.Data.([]string)
			masterData = rowData.Value.Data.([]*kdb.K)
		}
		for i, colName := range masterCols {
			KObj := masterData[i]
			var dat interface{}
			if KObj.Type < 0 {
				if KObj.Type == -kdb.KC {
					KObj.Data = string(KObj.Data.(byte))
				}
				dat = projectAtom(KObj.Data, depth)
			} else {
				switch {
				case KObj.Type == kdb.KC:
					// if the column is a key column, this is a string. Otherwise it is a char list
					if i < keyColCount {
						dat = projectAtom(KObj.Data, depth)
					} else {
						dat = charParser(KObj)
					}
				case KObj.Type > kdb.K0:
					dat = KObj.Data
				case KObj.Type == kdb.K0:
					stringColumn, err := stringParser(KObj)
					if err != nil {
						return nil, fmt.Errorf("Error parsing data of type K0")
					}
					dat = stringColumn
				}
			}
			frame.Fields = append(frame.Fields, data.NewField(colName, nil, dat))
		}
		frameArray[row] = frame
	}
	return frameArray, nil
}

func parseFrameName(key *kdb.K) string {
	// handling for homogenous dictionaries
	var frameNameArray []string
	if key.Type != kdb.K0 {
		if key.Type == kdb.KC {
			for _, l := range key.Data.([]interface{}) {
				frameNameArray = append(frameNameArray, string(l.(byte)))
			}
		} else {
			for _, val := range key.Data.([]interface{}) {
				frameNameArray = append(frameNameArray, fmt.Sprint(val))
			}
		}
		// handling for heterogenous dictionaries
	} else {
		for _, obj := range key.Data.([]*kdb.K) {
			if obj.Type == -kdb.KC {
				frameNameArray = append(frameNameArray, string(obj.Data.(byte)))
			} else {
				frameNameArray = append(frameNameArray, fmt.Sprint(obj.Data))
			}
		}
	}
	// concat all key strings together
	return strings.Join(frameNameArray, " - ")
}

func getDepth(colArray []*kdb.K) (int, error) {
	d := -1
	aggPresent := false
	for _, K := range colArray {
		if K.Type < 0 {
			aggPresent = true
			continue
		}
		if d == -1 {
			d = K.Len()
			continue
		}
		if d != K.Len() {
			return 0, fmt.Errorf("Columns are present of non-equal length")
		}
	}
	if d == -1 {
		if aggPresent {
			return 1, nil
		}
		return 0, fmt.Errorf("At least one key's value is an empty list '()'")
	}
	return d, nil
}

func correctedIndex(k *kdb.K, i int) interface{} {
	if k.Type < kdb.K0 || k.Type > kdb.XT {
		return nil
	}
	if k.Len() == 0 {
		// need to return null of that type
		if k.Type == kdb.K0 {
			return &kdb.K{kdb.K0, kdb.NONE, make([]*kdb.K, 0)}
		}
		return nil

	}
	if k.Type == kdb.K0 {
		return k.Data.([]*kdb.K)[i]
	}
	if k.Type > kdb.K0 && k.Type <= kdb.KT {
		return indexKdbArray(k, i)
	}
	// case for table
	// need to return dict with header
	if k.Type != kdb.XT {
		return nil
	}
	var t = k.Data.(kdb.Table)
	return &kdb.K{kdb.XD, kdb.NONE, correctedTableIndex(t, i)}
}

func indexKdbArray(k *kdb.K, i int) interface{} {
	switch {
	case k.Type == kdb.KB:
		return &kdb.K{-k.Type, kdb.NONE, k.Data.([]bool)[i]}
	case k.Type == kdb.UU:
		return &kdb.K{-k.Type, kdb.NONE, k.Data.([]uuid.UUID)[i]}
	case k.Type == kdb.KG:
		return &kdb.K{-k.Type, kdb.NONE, k.Data.([]byte)[i]}
	case k.Type == kdb.KH:
		return &kdb.K{-k.Type, kdb.NONE, k.Data.([]int16)[i]}
	case k.Type == kdb.KI:
		return &kdb.K{-k.Type, kdb.NONE, k.Data.([]int32)[i]}
	case k.Type == kdb.KJ:
		return &kdb.K{-k.Type, kdb.NONE, k.Data.([]int64)[i]}
	case k.Type == kdb.KE:
		return &kdb.K{-k.Type, kdb.NONE, k.Data.([]float32)[i]}
	case k.Type == kdb.KF:
		return &kdb.K{-k.Type, kdb.NONE, k.Data.([]float64)[i]}
	case k.Type == kdb.KC:
		return &kdb.K{-k.Type, kdb.NONE, k.Data.(string)[i]}
	case k.Type == kdb.KS:
		return &kdb.K{-k.Type, kdb.NONE, k.Data.([]string)[i]}
	case k.Type == kdb.KP:
		return &kdb.K{-k.Type, kdb.NONE, k.Data.([]time.Time)[i]}
	case k.Type == kdb.KM:
		return &kdb.K{-k.Type, kdb.NONE, k.Data.([]int32)[i]}
	case k.Type == kdb.KD:
		return &kdb.K{-k.Type, kdb.NONE, k.Data.([]time.Time)[i]}
	case k.Type == kdb.KZ:
		return &kdb.K{-k.Type, kdb.NONE, k.Data.([]time.Time)[i]}
	case k.Type == kdb.KN:
		return &kdb.K{-k.Type, kdb.NONE, k.Data.([]time.Duration)[i]}
	case k.Type == kdb.KU:
		return &kdb.K{-k.Type, kdb.NONE, k.Data.([]time.Time)[i]}
	case k.Type == kdb.KV:
		return &kdb.K{-k.Type, kdb.NONE, k.Data.([]time.Time)[i]}
	case k.Type == kdb.KT:
		return &kdb.K{-k.Type, kdb.NONE, k.Data.([]time.Time)[i]}
	}
	return nil
}

func correctedTableIndex(tbl kdb.Table, i int) kdb.Dict {
	var d = kdb.Dict{}
	d.Key = &kdb.K{kdb.KS, kdb.NONE, tbl.Columns}
	vslice := make([]*kdb.K, len(tbl.Columns))
	d.Value = &kdb.K{kdb.K0, kdb.NONE, vslice}
	for ci := range tbl.Columns {
		kd := correctedIndex(tbl.Data[ci], i)
		vslice[ci] = kd.(*kdb.K)
	}
	return d
}

func projectAtom(a interface{}, d int) interface{} {
	var o interface{}
	switch v := a.(type) {
	case int8:
		arr := make([]int8, d)
		for i := 0; i < d; i++ {
			arr[i] = v
		}
		o = arr
	case *int8:
		arr := make([]*int8, d)
		for i := 0; i < d; i++ {
			arr[i] = v
		}
		o = arr
	case int16:
		arr := make([]int16, d)
		for i := 0; i < d; i++ {
			arr[i] = v
		}
		o = arr
	case *int16:
		arr := make([]*int16, d)
		for i := 0; i < d; i++ {
			arr[i] = v
		}
		o = arr
	case int32:
		arr := make([]int32, d)
		for i := 0; i < d; i++ {
			arr[i] = v
		}
		o = arr
	case *int32:
		arr := make([]*int32, d)
		for i := 0; i < d; i++ {
			arr[i] = v
		}
		o = arr
	case int64:
		arr := make([]int64, d)
		for i := 0; i < d; i++ {
			arr[i] = v
		}
		o = arr
	case *int64:
		arr := make([]*int64, d)
		for i := 0; i < d; i++ {
			arr[i] = v
		}
		o = arr
	case uint8:
		arr := make([]uint8, d)
		for i := 0; i < d; i++ {
			arr[i] = v
		}
		o = arr
	case *uint8:
		arr := make([]*uint8, d)
		for i := 0; i < d; i++ {
			arr[i] = v
		}
		o = arr
	case uint16:
		arr := make([]uint16, d)
		for i := 0; i < d; i++ {
			arr[i] = v
		}
		o = arr
	case *uint16:
		arr := make([]*uint16, d)
		for i := 0; i < d; i++ {
			arr[i] = v
		}
		o = arr
	case uint32:
		arr := make([]uint32, d)
		for i := 0; i < d; i++ {
			arr[i] = v
		}
		o = arr
	case *uint32:
		arr := make([]*uint32, d)
		for i := 0; i < d; i++ {
			arr[i] = v
		}
		o = arr
	case uint64:
		arr := make([]uint64, d)
		for i := 0; i < d; i++ {
			arr[i] = v
		}
		o = arr
	case *uint64:
		arr := make([]*uint64, d)
		for i := 0; i < d; i++ {
			arr[i] = v
		}
		o = arr
	case float32:
		arr := make([]float32, d)
		for i := 0; i < d; i++ {
			arr[i] = v
		}
		o = arr
	case *float32:
		arr := make([]*float32, d)
		for i := 0; i < d; i++ {
			arr[i] = v
		}
		o = arr
	case float64:
		arr := make([]float64, d)
		for i := 0; i < d; i++ {
			arr[i] = v
		}
		o = arr
	case *float64:
		arr := make([]*float64, d)
		for i := 0; i < d; i++ {
			arr[i] = v
		}
		o = arr
	case string:
		arr := make([]string, d)
		for i := 0; i < d; i++ {
			arr[i] = v
		}
		o = arr
	case *string:
		arr := make([]*string, d)
		for i := 0; i < d; i++ {
			arr[i] = v
		}
		o = arr
	case bool:
		arr := make([]bool, d)
		for i := 0; i < d; i++ {
			arr[i] = v
		}
		o = arr
	case *bool:
		arr := make([]*bool, d)
		for i := 0; i < d; i++ {
			arr[i] = v
		}
		o = arr
	case time.Time:
		arr := make([]time.Time, d)
		for i := 0; i < d; i++ {
			arr[i] = v
		}
		o = arr
	case *time.Time:
		arr := make([]*time.Time, d)
		for i := 0; i < d; i++ {
			arr[i] = v
		}
		o = arr
	default:
		panic(fmt.Errorf("field '%s' specified with unsupported type %T", a, v))
	}
	return o
}
