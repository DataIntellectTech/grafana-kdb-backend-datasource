package plugin

import (
	"fmt"
	"reflect"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	kdb "github.com/sv/kdbgo"
)

func ParseSimpleKdbTable(res *kdb.K) (*data.Frame, error) {
	frame := data.NewFrame("response")
	kdbTable := res.Data.(kdb.Table)
	tabData := kdbTable.Data

	for colIndex, columnName := range kdbTable.Columns {
		//Manual handling of string cols
		if tabData[colIndex].Type == kdb.K0 {
			stringCol := tabData[colIndex].Data.([]*kdb.K)
			stringArray := make([]string, len(stringCol))
			for i, word := range stringCol {
				if word.Type != kdb.KC {
					return nil, fmt.Errorf("A column is present which is neither a vector nor a string column: %v. kdb+ type at index %v: %v", columnName, i, word.Type)
				}
				stringArray[i] = word.Data.(string)
			}
			frame.Fields = append(frame.Fields, data.NewField(columnName, nil, stringArray))
		} else {
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
		k := (kdbDict.Key.Data.(kdb.Table))
		frameName := k.Index(i).Value.String()
		frame := data.NewFrame(frameName)
		rowData := valData.Index(i) // I think this index call is what's causing variable nesting levels later on
		log.DefaultLogger.Info("A")
		for i, colName := range rowData.Key.Data.([]string) {
			log.DefaultLogger.Info(fmt.Sprintf("VALUES OUTER: %v", rowData.Value.Data.([]*kdb.K)[i].Data))
			log.DefaultLogger.Info(fmt.Sprintf("TYPES OUTER: %v", rowData.Value.Data.([]*kdb.K)[i].Type))
			log.DefaultLogger.Info(fmt.Sprintf("REFLECTED TYPE OF OUTER DATA: %v", reflect.TypeOf(rowData.Value.Data.([]*kdb.K)[i].Data)))
			// Looks like there is a bug with one of the transformations:
			// If the column is a flat type then the data is nested correctly, but atomic (so needs to be enlisted) - this is fine
			// If the column is nested however, then although the returned type of "rowData.Value.Data.([]*kdb.K)[i].Type" is non-zero, the
			// object its "Data" field contains is type *kdb.K (extra level of nesting for some reason)
			// below section is to account for this. This is probably due to a bug in the Indexing function, so this should be changed
			var dat interface{}
			if rowData.Value.Data.([]*kdb.K)[i].Type < 0 {
				log.DefaultLogger.Info("ENTERED ENLISTER")
				dat = enlistAtom(rowData.Value.Data.([]*kdb.K)[i].Data)
				log.DefaultLogger.Info(fmt.Sprintf("ADDING FOLLOWING AS FIELD: %v", dat))
				log.DefaultLogger.Info(fmt.Sprintf("FIELD OF FOLLOWING TYPE: %v", reflect.TypeOf(dat)))
			} else {
				dat = rowData.Value.Data.([]*kdb.K)[i].Data.(*kdb.K).Data
			}
			frame.Fields = append(frame.Fields, data.NewField(colName, nil, dat))
		}
		frameArray[i] = frame
	}
	return frameArray, nil
}

func enlistAtom(a interface{}) interface{} {
	var o interface{}
	switch v := a.(type) {
	case int8:
		o = []int8{v}
	case *int8:
		o = []*int8{v}
	case int16:
		o = []int16{v}
	case *int16:
		o = []*int16{v}
	case int32:
		o = []int32{v}
	case *int32:
		o = []*int32{v}
	case int64:
		o = []int64{v}
	case *int64:
		o = []*int64{v}
	case uint8:
		o = []uint8{v}
	case *uint8:
		o = []*uint8{v}
	case uint16:
		o = []uint16{v}
	case *uint16:
		o = []*uint16{v}
	case uint32:
		o = []uint32{v}
	case *uint32:
		o = []*uint32{v}
	case uint64:
		o = []uint64{v}
	case *uint64:
		o = []*uint64{v}
	case float32:
		o = []float32{v}
	case *float32:
		o = []*float32{v}
	case float64:
		o = []float64{v}
	case *float64:
		o = []*float64{v}
	case string:
		o = []string{v}
	case *string:
		o = []*string{v}
	case bool:
		o = []bool{v}
	case *bool:
		o = []*bool{v}
	case time.Time:
		o = []time.Time{v}
	case *time.Time:
		o = []*time.Time{v}
	default:
		panic(fmt.Errorf("field '%s' specified with unsupported type %T", a, v))
	}
	return o
}
