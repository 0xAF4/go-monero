package rpc

import (
	"encoding/json"
	"fmt"

	"github.com/0xAF4/go-monero/levin"
)

type UniversalRequest map[string]interface{}

func (u UniversalRequest) MarshalToBlob() []byte {
	pStorage := levin.PortableStorage{
		Entries: []levin.Entry{},
	}
	for key, val := range u {
		var sVal levin.Serializable
		switch v := val.(type) {
		case string:
			sVal = levin.BoostString(v)
		case uint64:
			sVal = levin.BoostUint64(v)
		case []uint64:
			sVal = levin.BoostUint64Array(v)
		default:
			panic(fmt.Errorf("unsupported type for key %s: %T", key, val))
		}
		entry := levin.Entry{
			Name:         key,
			Serializable: sVal,
		}
		pStorage.Entries = append(pStorage.Entries, entry)
	}
	return pStorage.Bytes()
}

func (u UniversalRequest) MarshalToJson() []byte {
	js, err := json.Marshal(u)
	if err != nil {
		panic(fmt.Errorf("failed to marshal json: %w", err))
	}
	return js
}

func (u *UniversalRequest) FromPortableStorate(response []byte) error {
	// Ответ приходит в бинарном формате (portable storage)
	rStorage, err := levin.NewPortableStorageFromBytes(response)
	if err != nil {
		return err
	}

	for _, val := range rStorage.Entries {
		(*u)[val.Name] = val.Value
	}
	return nil
}

func (u *UniversalRequest) FromJson(js []byte) {
	
}
