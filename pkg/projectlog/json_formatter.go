package projectlog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	defaultTimestampFormat = time.RFC3339
	FieldKeyMsg            = "msg"
	FieldKeyLevel          = "level"
	FieldKeyTime           = "time"
	FieldKeyLogrusError    = "logrus_error"
	FieldKeyFunc           = "func"
	FieldKeyFile           = "file"
	FieldModule            = "module"
)

type LogFormat struct {
	Level       interface{} `json:"level,omitempty"`
	Module      interface{} `json:"module,omitempty"`
	Time        interface{} `json:"time,omitempty"`
	File        interface{} `json:"file,omitempty"`
	Function    interface{} `json:"function,omitempty"`
	Msg         interface{} `json:"msg,omitempty"`
	LogrusError interface{} `json:"logrus_error,omitempty"`
}

const LogPrefixName = "falcon"

type JSONFormatter struct {
	// TimestampFormat sets the format used for marshaling timestamps.
	TimestampFormat string

	// DisableTimestamp allows disabling automatic timestamps in output
	DisableTimestamp bool

	// DataKey allows users to put all the log entry parameters into a nested dictionary at a given key.
	DataKey string

	// FieldMap allows users to customize the names of keys for default fields.
	// As an example:
	// formatter := &JSONFormatter{
	//   	FieldMap: FieldMap{
	// 		 FieldKeyTime:  "@timestamp",
	// 		 FieldKeyLevel: "@level",
	// 		 FieldKeyMsg:   "@message",
	// 		 FieldKeyFunc:  "@caller",
	//    },
	// }
	FieldMap FieldMap

	// PrettyPrint will indent all json logs
	PrettyPrint bool
}

type fieldKey string

type FieldMap map[fieldKey]string

func (f FieldMap) resolve(key fieldKey) string {
	if k, ok := f[key]; ok {
		return k
	}

	return string(key)
}

type Fields map[string]interface{}

func (f *JSONFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	data := make(Fields, len(entry.Data)+4)
	for k, v := range entry.Data {
		switch v := v.(type) {
		case error:
			// Otherwise errors are ignored by `encoding/json`
			// https://github.com/sirupsen/logrus/issues/137
			data[k] = v.Error()
		default:
			data[k] = v
		}
	}

	if f.DataKey != "" {
		newData := make(Fields, 4)
		newData[f.DataKey] = data
		data = newData
	}

	prefixFieldClashes(data, f.FieldMap, entry.HasCaller())

	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = defaultTimestampFormat
	}

	if !f.DisableTimestamp {
		data[f.FieldMap.resolve(FieldKeyTime)] = entry.Time.Format(timestampFormat)
	}
	data[f.FieldMap.resolve(FieldKeyMsg)] = entry.Message
	data[f.FieldMap.resolve(FieldKeyLevel)] = entry.Level.String()
	data[f.FieldMap.resolve(FieldModule)] = LogPrefixName
	if entry.HasCaller() {
		data[f.FieldMap.resolve(FieldKeyFunc)] = entry.Caller.Function
		data[f.FieldMap.resolve(FieldKeyFile)] = fmt.Sprintf("%s:%d", entry.Caller.File, entry.Caller.Line)
	}

	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	encoder := json.NewEncoder(b)
	if f.PrettyPrint {
		encoder.SetIndent("", "  ")
	}
	if err := encoder.Encode(convertToLogStruct(data)); err != nil {
		return nil, fmt.Errorf("failed to marshal fields to JSON, %v", err)
	}

	return b.Bytes(), nil
}

func prefixFieldClashes(data Fields, fieldMap FieldMap, reportCaller bool) {
	timeKey := fieldMap.resolve(FieldKeyTime)
	if t, ok := data[timeKey]; ok {
		data["fields."+timeKey] = t
		delete(data, timeKey)
	}

	msgKey := fieldMap.resolve(FieldKeyMsg)
	if m, ok := data[msgKey]; ok {
		data["fields."+msgKey] = m
		delete(data, msgKey)
	}

	levelKey := fieldMap.resolve(FieldKeyLevel)
	if l, ok := data[levelKey]; ok {
		data["fields."+levelKey] = l
		delete(data, levelKey)
	}

	moduleKey := fieldMap.resolve(FieldModule)
	if m, ok := data[moduleKey]; ok {
		data["fields."+moduleKey] = m
		delete(data, moduleKey)
	}
	logrusErrKey := fieldMap.resolve(FieldKeyLogrusError)
	if l, ok := data[logrusErrKey]; ok {
		data["fields."+logrusErrKey] = l
		delete(data, logrusErrKey)
	}

	// If reportCaller is not set, 'func' will not conflict.
	if reportCaller {
		funcKey := fieldMap.resolve(FieldKeyFunc)
		if l, ok := data[funcKey]; ok {
			data["fields."+funcKey] = l
		}
		fileKey := fieldMap.resolve(FieldKeyFile)
		if l, ok := data[fileKey]; ok {
			data["fields."+fileKey] = l
		}
	}
}

func convertToLogStruct(data map[string]interface{}) *LogFormat {
	logFormat := &LogFormat{}
	if v, ok := data[FieldKeyMsg]; ok {
		logFormat.Msg = v
	}

	if v, ok := data[FieldKeyLevel]; ok {
		logFormat.Level = v
	}

	if v, ok := data[FieldKeyTime]; ok {
		logFormat.Time = v
	}

	if v, ok := data[FieldKeyLogrusError]; ok {
		logFormat.LogrusError = v
	}

	if v, ok := data[FieldModule]; ok {
		logFormat.Module = v
	}

	if v, ok := data[FieldKeyFunc]; ok {
		logFormat.Function = v
	}

	if v, ok := data[FieldKeyFile]; ok {
		logFormat.File = v
	}

	return logFormat
}
