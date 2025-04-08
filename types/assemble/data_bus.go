package assemble

import "sync"

type DataBus struct {
	m sync.Map
}

func (d *DataBus) GetVal(key string) interface{} {
	val, ok := d.m.Load(key)
	if ok {
		return val
	}
	return nil
}

func (d *DataBus) SetVal(key string, val interface{}) {
	d.m.Store(key, val)
}
