package model

import (
	"encoding/json"
	"strconv"
	"time"
)

// Server model
type Server struct {
	KeyPair   *ServerKeypair
	Interface *ServerInterface
}

// ServerKeypair model
type ServerKeypair struct {
	PrivateKey string    `json:"private_key"`
	PublicKey  string    `json:"public_key"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ServerInterface model
type ServerInterface struct {
	Addresses  []string  `json:"addresses"`
	ListenPort int       `json:"listen_port"`
	UpdatedAt  time.Time `json:"updated_at"`
	PostUp     string    `json:"post_up"`
	PreDown    string    `json:"pre_down"`
	PostDown   string    `json:"post_down"`
	// فیلد جدید برای فاصله بررسی (بر حسب دقیقه)
	CheckInterval int `json:"check_interval"`
}

// UnmarshalJSON implements custom decoding to allow listen_port and check_interval
// to be provided as either strings or numbers.
func (s *ServerInterface) UnmarshalJSON(data []byte) error {
	type Alias ServerInterface
	aux := &struct {
		ListenPort    any `json:"listen_port"`
		CheckInterval any `json:"check_interval"`
		*Alias
	}{Alias: (*Alias)(s)}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if aux.ListenPort != nil {
		switch v := aux.ListenPort.(type) {
		case string:
			p, err := strconv.Atoi(v)
			if err != nil {
				return err
			}
			s.ListenPort = p
		case float64:
			s.ListenPort = int(v)
		}
	}

	if aux.CheckInterval != nil {
		switch v := aux.CheckInterval.(type) {
		case string:
			p, err := strconv.Atoi(v)
			if err != nil {
				return err
			}
			s.CheckInterval = p
		case float64:
			s.CheckInterval = int(v)
		}
	}

	return nil
}
