package plugin

import (
	"fmt"

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
	if kdbDict.Key.Type != kdb.KT || kdbDict.Value.Type != kdb.KT {
		return nil, fmt.Errorf("Either the key or the value of the returned dictionary obejct is not a table of type 98.")
	}
	rc := kdbDict.Key.Len()
	valData := kdbDict.Value.Data.(kdb.Table)
	frameArray := make([]*data.Frame, rc)

	for i := 0; i < rc; i++ {
		k := (kdbDict.Key.Data.(kdb.Table))
		frameName := k.Index(i).Value.String()
		rowData := valData.Index(i)
		frame, err := ParseSimpleKdbTable(&kdb.K{Type: kdb.XT, Attr: kdb.NONE, Data: kdb.Table{Columns: rowData.Key.Data.([]string), Data: rowData.Value.Data.([]*kdb.K)}})
		if err != nil {
			return nil, fmt.Errorf("Error parsing at least one series from the grouped response object")
		}
		frame.Name = frameName
		frameArray[i] = frame
	}
	return frameArray, nil
}
