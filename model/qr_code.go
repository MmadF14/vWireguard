package model

// QRCodeSettings represents settings for QR code generation
type QRCodeSettings struct {
	ShowPrivateKey bool
	ShowPublicKey  bool
	ShowAddress    bool
	ShowDNS        bool
	ShowAllowedIPs bool
} 