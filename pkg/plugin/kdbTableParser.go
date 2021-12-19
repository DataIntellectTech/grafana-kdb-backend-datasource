package plugin

import (
	"fmt"
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

func ParseSimpleKdbTable(res *kdb.K) (*data.Frame, error) {
	frame := data.NewFrame("response")
	kdbTable := res.Data.(kdb.Table)
	tabData := kdbTable.Data

	for colIndex, columnName := range kdbTable.Columns {
		//Manual handling of string cols
		switch {
		case tabData[colIndex].Type == kdb.K0:
			stringColumn, err := stringParser(tabData[colIndex])
			if err != nil {
				return nil, fmt.Errorf("The following column: %v return this error: %v", columnName, err)
			}
			frame.Fields = append(frame.Fields, data.NewField(columnName, nil, stringColumn))
		case tabData[colIndex].Type == kdb.KC:
			frame.Fields = append(frame.Fields, data.NewField(columnName, nil, charParser(tabData[colIndex])))
		default:
			frame.Fields = append(frame.Fields, data.NewField(columnName, nil, tabData[colIndex].Data))
		}

	}
	return frame, nil
}

func ParseGroupedKdbTable(res *kdb.K) ([]*data.Frame, error) {
	kdbDict := res.Data.(kdb.Dict)
	if kdbDict.Key.Type != kdb.XT || kdbDict.Value.Type != kdb.XT {
		return nil, fmt.Errorf("Either the key or the value of the returned dictionary obejct is not a table of type 98.")
	}
	rc := kdbDict.Key.Len()
	valData := kdbDict.Value.Data.(kdb.Table)
	frameArray := make([]*data.Frame, rc)

	for i := 0; i < rc; i++ {
		k := kdbDict.Key.Data.(kdb.Table)
		frameName := correctedTableIndex(k, i).Value.String()
		frame := data.NewFrame(frameName)
		rowData := correctedTableIndex(valData, i)
		depth, err := getDepth(rowData.Value.Data.([]*kdb.K))
		if err != nil {
			return nil, err
		}
		for i, colName := range rowData.Key.Data.([]string) {
			KObj := rowData.Value.Data.([]*kdb.K)[i]
			var dat interface{}
			if KObj.Type < 0 {
				dat = projectAtom(KObj.Data, depth)
			} else {
				switch {
				case KObj.Type == kdb.KC:
					dat = charParser(KObj)
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
		frameArray[i] = frame
	}
	return frameArray, nil
}

func getDepth(colArray []*kdb.K) (int, error) {
	d := -1
	aggPresent := false
	for _, K := range colArray {
		//log.DefaultLogger.Info(fmt.Sprintf("%v", K))
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
		return &kdb.K{-k.Type, kdb.NONE, k.Data.([]byte)[i]}
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
