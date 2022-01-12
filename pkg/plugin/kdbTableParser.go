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

func parser(inputData *kdb.K, columnName string) *data.Field {

	switch {
	case inputData.Type == kdb.K0:
		stringColumn, err := stringParser(inputData)
		if err != nil {
			//return nil, fmt.Errorf("The following column: %v return this error: %v", columnName, err)
		}
		return data.NewField(columnName, nil, stringColumn)
	case inputData.Type == kdb.KC:
		return data.NewField(columnName, nil, charParser(inputData))

	case inputData.Type == kdb.KN:
		//timespan
		durArr := inputData.Data.([]time.Duration)
		durIntArr := make([]int64, len(durArr))
		for i, dur := range durArr {
			durIntArr[i] = int64(dur)
		}
		return data.NewField(columnName, nil, durIntArr)

	case inputData.Type == kdb.KT:
		//Time
		kdbTimeArr := inputData.Data.([]kdb.Time)
		timeArr := make([]int32, len(kdbTimeArr))
		for index, entry := range kdbTimeArr {
			timeArr[index] = int32(time.Time(entry).Hour()*3600000 + time.Time(entry).Minute()*60000 + time.Time(entry).Second()*1000 + time.Time(entry).Nanosecond()/1000000)

		}
		return data.NewField(columnName, nil, timeArr)

	case inputData.Type == kdb.UU:
		//GUID

		uuidArr := inputData.Data.([]uuid.UUID)
		guidArr := make([]string, len(uuidArr))
		for i, entry := range uuidArr {
			guidArr[i] = entry.String()
		}

		return data.NewField(columnName, nil, guidArr)

	case inputData.Type == kdb.KU:
		//Minute
		minArr := inputData.Data.([]kdb.Minute)
		minTimeArr := make([]int32, len(minArr))
		for index, entry := range minArr {
			minTimeArr[index] = int32(time.Time(entry).Minute() + time.Time(entry).Hour()*60)
		}
		return data.NewField(columnName, nil, minTimeArr)

	case inputData.Type == kdb.KV:
		//Second
		secArr := inputData.Data.([]kdb.Second)
		secTimeArr := make([]int32, len(secArr))
		for index, entry := range secArr {
			secTimeArr[index] = int32(time.Time(entry).Second() + time.Time(entry).Minute()*60 + time.Time(entry).Hour()*3600)
		}
		return data.NewField(columnName, nil, secTimeArr)

	case inputData.Type == kdb.KM:
		// Month
		monthArr := inputData.Data.([]kdb.Month)
		monthIntArr := make([]int32, len(monthArr))
		for index, val := range monthArr {
			monthIntArr[index] = int32(val)
		}
		return data.NewField(columnName, nil, monthIntArr)

	default:
		return data.NewField(columnName, nil, inputData.Data)
	}
}

func ParseSimpleKdbTable(res *kdb.K) (*data.Frame, error) {
	log.DefaultLogger.Info("Simple Table")
	frame := data.NewFrame("response")
	kdbTable := res.Data.(kdb.Table)
	tabData := kdbTable.Data

	for colIndex, columnName := range kdbTable.Columns {
		log.DefaultLogger.Info(strconv.Itoa(int(tabData[colIndex].Type)))
		frame.Fields = append(frame.Fields, parser(tabData[colIndex], columnName))
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
				log.DefaultLogger.Info(strconv.Itoa(int(KObj.Type)))
				switch {
				case KObj.Type == kdb.KC:
					// if the column is a key column, this is a string. Otherwise it is a char list
					if i < keyColCount {
						dat = projectAtom(KObj.Data, depth)
					} else {

						dat = charParser(KObj)
					}
				case KObj.Type > kdb.K0:
					frame.Fields = append(frame.Fields, parser(KObj, colName))
					continue

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
		return &kdb.K{-k.Type, kdb.NONE, k.Data.([]kdb.Month)[i]}
	case k.Type == kdb.KD:
		return &kdb.K{-k.Type, kdb.NONE, k.Data.([]time.Time)[i]}
	case k.Type == kdb.KZ:
		return &kdb.K{-k.Type, kdb.NONE, k.Data.([]time.Time)[i]}
	case k.Type == kdb.KN:
		return &kdb.K{-k.Type, kdb.NONE, k.Data.([]time.Duration)[i]}
	case k.Type == kdb.KU:
		return &kdb.K{-k.Type, kdb.NONE, k.Data.([]kdb.Minute)[i]}
	case k.Type == kdb.KV:
		return &kdb.K{-k.Type, kdb.NONE, k.Data.([]kdb.Second)[i]}
	case k.Type == kdb.KT:
		return &kdb.K{-k.Type, kdb.NONE, k.Data.([]kdb.Time)[i]}
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
	case time.Duration:
		arr := make([]int64, d)
		for i := 0; i < d; i++ {
			arr[i] = int64(v)
		}
		o = arr
	case kdb.Minute:
		arr := make([]int32, d)
		for i := 0; i < d; i++ {
			arr[i] = int32(time.Time(v).Sub(time.Time{}) / time.Minute)
		}
		o = arr
	case kdb.Month:
		arr := make([]int32, d)
		for i := 0; i < d; i++ {
			arr[i] = int32(v)
		}
		o = arr
	case kdb.Second:
		arr := make([]int32, d)
		for i := 0; i < d; i++ {
			arr[i] = int32(time.Time(v).Second() + time.Time(v).Minute()*60 + time.Time(v).Hour()*3600)
		}
		o = arr
	case uuid.UUID:
		arr := make([]string, d)
		for i := 0; i < d; i++ {
			arr[i] = v.String()
		}
		o = arr
	case kdb.Time:
		arr := make([]int32, d)
		for i := 0; i < d; i++ {
			arr[i] = int32(time.Time(v).Hour()*3600000 + time.Time(v).Minute()*60000 + time.Time(v).Second()*1000 + time.Time(v).Nanosecond()/1000000)
		}
		o = arr
	default:
		panic(fmt.Errorf("field '%s' specified with unsupported type %T", a, v))
	}
	return o
}
