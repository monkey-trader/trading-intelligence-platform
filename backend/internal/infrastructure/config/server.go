// Package config liest die Server-Konfiguration aus der Umgebung.
package config

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

const (
	defaultHost = "127.0.0.1"
	defaultPort = 3000
	minimumPort = 1
	maximumPort = 65535
)

// hostWithPortPattern: host:port, IPv6 in eckigen Klammern, Port 1–5 Ziffern.
var hostWithPortPattern = regexp.MustCompile(`^(\[[0-9a-fA-F:.]+\]|[A-Za-z0-9.-]+):\d{1,5}$`)

// loopbackIPv4Pattern erkennt 127.x.y.z.
var loopbackIPv4Pattern = regexp.MustCompile(`^127(\.\d{1,3}){3}$`)

// ServerConfig enthält Host, Port und die erlaubten Host-Header (host:port).
type ServerConfig struct {
	Host         string
	Port         int
	AllowedHosts []string
}

// RemoteHostNotAllowedError meldet: HOST ist keine Loopback-Adresse und ALLOW_REMOTE ist nicht true.
type RemoteHostNotAllowedError struct{ Host string }

// Error beschreibt den abgelehnten Host und nennt ALLOW_REMOTE.
func (err *RemoteHostNotAllowedError) Error() string {
	return fmt.Sprintf(
		"HOST %q ist keine Loopback-Adresse. Die API hat noch keine Authentifizierung; nur mit ALLOW_REMOTE=true erlaubt.",
		err.Host,
	)
}

// InvalidPortError meldet: PORT ist keine ganze Zahl von 1 bis 65535.
type InvalidPortError struct{ Value string }

// Error nennt den ungültigen PORT-Wert.
func (err *InvalidPortError) Error() string {
	return `Ungültiger PORT: "` + err.Value + `"`
}

// MissingAllowedHostsError meldet: ALLOW_REMOTE=true, aber ALLOWED_HOSTS fehlt oder ist leer.
type MissingAllowedHostsError struct{}

// Error erklärt, dass ALLOWED_HOSTS bei ALLOW_REMOTE=true Pflicht ist.
func (err *MissingAllowedHostsError) Error() string {
	return "Mit ALLOW_REMOTE=true muss ALLOWED_HOSTS gesetzt sein " +
		"(kommagetrennte Host-Header mit Port, z.B. meinrechner.local:3000)."
}

// InvalidAllowedHostError meldet: ein Eintrag in ALLOWED_HOSTS hat nicht das Format host:port.
type InvalidAllowedHostError struct{ Entry string }

// Error nennt den ungültigen Eintrag aus ALLOWED_HOSTS.
func (err *InvalidAllowedHostError) Error() string {
	return fmt.Sprintf("ALLOWED_HOSTS enthält %q, erwartet wird host:port (z.B. meinrechner.local:3000).", err.Entry)
}

// ReadServerConfig liest HOST, PORT, ALLOW_REMOTE und ALLOWED_HOSTS.
// Leere Werte und reine Leerzeichen gelten als nicht gesetzt. Ein Host außerhalb von Loopback
// ist nur mit ALLOW_REMOTE=true erlaubt; dann ist ALLOWED_HOSTS Pflicht.
func ReadServerConfig(lookupEnv func(string) string) (ServerConfig, error) {
	allowRemote := lookupEnv("ALLOW_REMOTE") == "true"

	host := strings.TrimSpace(lookupEnv("HOST"))
	if host == "" {
		host = defaultHost
	}
	if !isLoopbackHost(host) && !allowRemote {
		return ServerConfig{}, &RemoteHostNotAllowedError{Host: host}
	}

	port, err := parsePort(strings.TrimSpace(lookupEnv("PORT")))
	if err != nil {
		return ServerConfig{}, err
	}

	allowedHosts := withBoundHost(
		[]string{fmt.Sprintf("127.0.0.1:%d", port), fmt.Sprintf("localhost:%d", port), fmt.Sprintf("[::1]:%d", port)},
		host,
		port,
	)
	if allowRemote {
		remoteHosts, err := parseRemoteAllowedHosts(lookupEnv("ALLOWED_HOSTS"))
		if err != nil {
			return ServerConfig{}, err
		}
		allowedHosts = append(allowedHosts, remoteHosts...)
	}

	return ServerConfig{Host: host, Port: port, AllowedHosts: allowedHosts}, nil
}

func parsePort(rawPort string) (int, error) {
	if rawPort == "" {
		return defaultPort, nil
	}
	port, err := strconv.Atoi(rawPort)
	if err != nil || port < minimumPort || port > maximumPort {
		return 0, &InvalidPortError{Value: rawPort}
	}
	return port, nil
}

func parseRemoteAllowedHosts(rawAllowedHosts string) ([]string, error) {
	var remoteHosts []string
	for _, entry := range strings.Split(rawAllowedHosts, ",") {
		trimmedEntry := strings.TrimSpace(entry)
		if trimmedEntry == "" {
			continue
		}
		if !hostWithPortPattern.MatchString(trimmedEntry) {
			return nil, &InvalidAllowedHostError{Entry: trimmedEntry}
		}
		remoteHosts = append(remoteHosts, trimmedEntry)
	}
	if len(remoteHosts) == 0 {
		return nil, &MissingAllowedHostsError{}
	}
	return remoteHosts, nil
}

// withBoundHost ergänzt den gebundenen Loopback-Host (z.B. 127.0.0.2), damit Aufrufe über genau
// diese Adresse nicht vom Host-Check abgelehnt werden. IPv6 wird in eckigen Klammern geschrieben.
func withBoundHost(loopbackHosts []string, host string, port int) []string {
	if !isLoopbackHost(host) {
		return loopbackHosts
	}
	boundHost := fmt.Sprintf("%s:%d", host, port)
	if strings.Contains(host, ":") {
		boundHost = fmt.Sprintf("[%s]:%d", host, port)
	}
	if slices.Contains(loopbackHosts, boundHost) {
		return loopbackHosts
	}
	return append(loopbackHosts, boundHost)
}

func isLoopbackHost(host string) bool {
	return host == "localhost" || host == "::1" || loopbackIPv4Pattern.MatchString(host)
}
