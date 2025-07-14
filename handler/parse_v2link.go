package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/MmadF14/vwireguard/model"
)

// ParseV2LinkString parses a vmess/vless/trojan link into V2rayTunnelConfig
func ParseV2LinkString(link string) (*model.V2rayTunnelConfig, error) {
	if strings.HasPrefix(link, "vmess://") {
		data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(link, "vmess://"))
		if err != nil {
			return nil, err
		}
		var v struct {
			Add  string `json:"add"`
			Port string `json:"port"`
			ID   string `json:"id"`
			Host string `json:"host"`
			Path string `json:"path"`
			Net  string `json:"net"`
			TLS  string `json:"tls"`
			SNI  string `json:"sni"`
			FP   string `json:"fp"`
			Alpn string `json:"alpn"`
			Flow string `json:"flow"`
		}
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, err
		}
		port, _ := strconv.Atoi(v.Port)
		cfg := &model.V2rayTunnelConfig{
			Protocol:      "vmess",
			RemoteAddress: v.Add,
			RemotePort:    port,
			UUID:          v.ID,
			Flow:          v.Flow,
			Security:      "none",
			ServerName:    v.Host,
			Fingerprint:   v.FP,
			Network:       v.Net,
			Path:          v.Path,
			SNI:           v.SNI,
		}
		if v.TLS != "" {
			cfg.Security = v.TLS
		}
		if v.Alpn != "" {
			cfg.Alpn = strings.Split(v.Alpn, ",")
		}
		return cfg, nil
	}

	if strings.HasPrefix(link, "vless://") || strings.HasPrefix(link, "trojan://") {
		u, err := url.Parse(link)
		if err != nil {
			return nil, err
		}
		protocol := strings.TrimSuffix(u.Scheme, "")
		port, _ := strconv.Atoi(u.Port())
		cfg := &model.V2rayTunnelConfig{
			Protocol:      protocol,
			RemoteAddress: u.Hostname(),
			RemotePort:    port,
			Security:      "none",
		}
		if protocol == "vless" {
			cfg.UUID = u.User.Username()
		} else if protocol == "trojan" {
			cfg.Password = u.User.Username()
		}
		q := u.Query()
		if s := q.Get("security"); s != "" {
			cfg.Security = s
		}
		if s := q.Get("flow"); s != "" {
			cfg.Flow = s
		}
		if s := q.Get("host"); s != "" {
			cfg.ServerName = s
		}
		if s := q.Get("sni"); s != "" {
			cfg.SNI = s
		}
		if s := q.Get("fp"); s != "" {
			cfg.Fingerprint = s
		}
		if s := q.Get("alpn"); s != "" {
			cfg.Alpn = strings.Split(s, ",")
		}
		if s := q.Get("type"); s != "" {
			cfg.Network = s
		}
		if s := q.Get("path"); s != "" {
			cfg.Path = s
		}
		if s := q.Get("serviceName"); s != "" && cfg.Path == "" {
			cfg.Path = s
		}
		return cfg, nil
	}
	return nil, fmt.Errorf("invalid link")
}

// ParseV2Link handles POST /api/utils/parse_v2link
func ParseV2Link() echo.HandlerFunc {
	return func(c echo.Context) error {
		var req struct {
			Link string `json:"link"`
		}
		if err := c.Bind(&req); err != nil || req.Link == "" {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "invalid link"})
		}
		cfg, err := ParseV2LinkString(req.Link)
		if err != nil {
			return c.JSON(http.StatusBadRequest, jsonHTTPResponse{false, "invalid link"})
		}
		return c.JSON(http.StatusOK, cfg)
	}
}
