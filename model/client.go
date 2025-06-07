package model

import (
	"time"
)

// Client model
type Client struct {

}

// ClientData includes the Client and extra data
type ClientData struct {
	Client *Client
	QRCode string
}

type QRCodeSettings struct {
	Enabled    bool
	IncludeDNS bool
	IncludeMTU bool
}
