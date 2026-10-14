package oracle

import (
	"errors"
	"strconv"
	"strings"

	go_ora "github.com/sijms/go-ora/v2"
)

type Config struct {
	Host, User, Password, Name, Mode string
	Port                             int
}

func (c Config) DSN() (string, error) {
	if strings.TrimSpace(c.Host) == "" || strings.TrimSpace(c.User) == "" || strings.TrimSpace(c.Name) == "" {
		return "", errors.New("host, username and service name/SID are required")
	}
	if c.Port < 1 || c.Port > 65535 {
		return "", errors.New("port must be between 1 and 65535")
	}
	options := map[string]string{"CONNECTION TIMEOUT": "10", "TIMEOUT": "60"}
	service := c.Name
	switch c.Mode {
	case "service":
	case "sid":
		service = ""
		options["SID"] = c.Name
	default:
		return "", errors.New("connection mode must be service or sid")
	}
	return go_ora.BuildUrl(c.Host, c.Port, service, c.User, c.Password, options), nil
}

func ParsePort(text string) (int, error) {
	port, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil || port < 1 || port > 65535 {
		return 0, errors.New("port must be between 1 and 65535")
	}
	return port, nil
}
