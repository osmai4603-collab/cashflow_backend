package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

func parseJSONDuration(raw json.RawMessage, current time.Duration) (time.Duration, error) {
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return current, nil
	}

	var value string
	if raw[0] == '"' {
		if err := json.Unmarshal(raw, &value); err != nil {
			return 0, err
		}
		if duration, err := time.ParseDuration(value); err == nil {
			return duration, nil
		}
		seconds, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid duration %q", value)
		}
		return time.Duration(seconds) * time.Second, nil
	}

	var numeric json.Number
	if err := json.Unmarshal(raw, &numeric); err != nil {
		return 0, fmt.Errorf("invalid duration: %w", err)
	}
	nanoseconds, err := numeric.Int64()
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q", numeric)
	}
	return time.Duration(nanoseconds), nil
}

func (s *ServerSettings) UnmarshalJSON(data []byte) error {
	type alias ServerSettings
	aux := struct {
		*alias
		ReadTimeout       json.RawMessage `json:"read_timeout"`
		ReadHeaderTimeout json.RawMessage `json:"read_header_timeout"`
		WriteTimeout      json.RawMessage `json:"write_timeout"`
		IdleTimeout       json.RawMessage `json:"idle_timeout"`
		DrainDuration     json.RawMessage `json:"drain_duration"`
		ShutdownTimeout   json.RawMessage `json:"shutdown_timeout"`
	}{alias: (*alias)(s)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	var err error
	if s.ReadTimeout, err = parseJSONDuration(aux.ReadTimeout, s.ReadTimeout); err != nil {
		return fmt.Errorf("read_timeout: %w", err)
	}
	if s.ReadHeaderTimeout, err = parseJSONDuration(aux.ReadHeaderTimeout, s.ReadHeaderTimeout); err != nil {
		return fmt.Errorf("read_header_timeout: %w", err)
	}
	if s.WriteTimeout, err = parseJSONDuration(aux.WriteTimeout, s.WriteTimeout); err != nil {
		return fmt.Errorf("write_timeout: %w", err)
	}
	if s.IdleTimeout, err = parseJSONDuration(aux.IdleTimeout, s.IdleTimeout); err != nil {
		return fmt.Errorf("idle_timeout: %w", err)
	}
	if s.DrainDuration, err = parseJSONDuration(aux.DrainDuration, s.DrainDuration); err != nil {
		return fmt.Errorf("drain_duration: %w", err)
	}
	if s.ShutdownTimeout, err = parseJSONDuration(aux.ShutdownTimeout, s.ShutdownTimeout); err != nil {
		return fmt.Errorf("shutdown_timeout: %w", err)
	}
	return nil
}

func (d *DatabaseSettings) UnmarshalJSON(data []byte) error {
	type alias DatabaseSettings
	aux := struct {
		*alias
		MaxConnLifetime json.RawMessage `json:"max_conn_lifetime"`
		MaxConnIdleTime json.RawMessage `json:"max_conn_idle_time"`
	}{alias: (*alias)(d)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	var err error
	if d.MaxConnLifetime, err = parseJSONDuration(aux.MaxConnLifetime, d.MaxConnLifetime); err != nil {
		return fmt.Errorf("max_conn_lifetime: %w", err)
	}
	if d.MaxConnIdleTime, err = parseJSONDuration(aux.MaxConnIdleTime, d.MaxConnIdleTime); err != nil {
		return fmt.Errorf("max_conn_idle_time: %w", err)
	}
	return nil
}

func (w *WorkerSettings) UnmarshalJSON(data []byte) error {
	type alias WorkerSettings
	aux := struct {
		*alias
		TimeWorkerCron json.RawMessage `json:"time_worker_cron"`
	}{alias: (*alias)(w)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	var err error
	if w.TimeWorkerCron, err = parseJSONDuration(aux.TimeWorkerCron, w.TimeWorkerCron); err != nil {
		return fmt.Errorf("time_worker_cron: %w", err)
	}
	return nil
}

func (s *StockSettings) UnmarshalJSON(data []byte) error {
	type alias StockSettings
	aux := struct {
		*alias
		ReorderInterval json.RawMessage `json:"reorder_interval"`
	}{alias: (*alias)(s)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	var err error
	if s.ReorderInterval, err = parseJSONDuration(aux.ReorderInterval, s.ReorderInterval); err != nil {
		return fmt.Errorf("reorder_interval: %w", err)
	}
	return nil
}

func (l *LimitSettings) UnmarshalJSON(data []byte) error {
	type alias LimitSettings
	aux := struct {
		*alias
		TimeCPU      json.RawMessage `json:"time_cpu"`
		TimeReal     json.RawMessage `json:"time_real"`
		TimeRealCron json.RawMessage `json:"time_real_cron"`
	}{alias: (*alias)(l)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	var err error
	if l.TimeCPU, err = parseJSONDuration(aux.TimeCPU, l.TimeCPU); err != nil {
		return fmt.Errorf("time_cpu: %w", err)
	}
	if l.TimeReal, err = parseJSONDuration(aux.TimeReal, l.TimeReal); err != nil {
		return fmt.Errorf("time_real: %w", err)
	}
	if l.TimeRealCron, err = parseJSONDuration(aux.TimeRealCron, l.TimeRealCron); err != nil {
		return fmt.Errorf("time_real_cron: %w", err)
	}
	return nil
}
