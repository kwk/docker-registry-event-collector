package main

import (
	"errors"
	"fmt"
	"github.com/golang/glog"
	"gopkg.in/mgo.v2"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

// Config stores all the configuration represented in a config.yml
type Config struct {
	DialInfo   MongoDialConfig `yaml:"dial_info,omitempty"`
	Collection string          `yaml:"collection,omitempty"`
	Server     ServerConfig    `yaml:"server,omitempty"`
}

// MongoDialConfig stores how we connect to the MongoDB server. In addition to
// the regular dial info we also provide another field to point to a password
// file.
type MongoDialConfig struct {
	mgo.DialInfo `yaml:",inline"`
	PasswordFile string `yaml:"password_file,omitempty"`
}

// ServerConfig stores information on how to configure the server accepting
// events from the registry.
type ServerConfig struct {
	Address string    `yaml:"address,omitempty"`
	Port    uint      `yaml:"port,omitempty"`
	Route   string    `yaml:"route,omitempty"`
	Ssl     SslConfig `yaml:"ssl,omitempty"`
}

// SslConfig stores some information about certificates and may be extended in
// the future
type SslConfig struct {
	Cert    string `yaml:"cert,omitempty"`     // path to certificate file in PEM format
	CertKey string `yaml:"cert_key,omitempty"` // path to certificate key file in PEM format
}

// GetEndpointConnectionString builds and returns a string with the IP and port
// separated by a colon. Nothing special but anyway.
func (s Config) GetEndpointConnectionString() string {
	return fmt.Sprintf("%s:%d", s.Server.Address, s.Server.Port)
}

// LoadConfig parses all flags from the command line and returns
// an initialized Settings object and an error object if any. For instance if it
// cannot find the SSL certificate file or the SSL key file it will set the
// returned error appropriately.
func LoadConfig(path string) (*Config, error) {
	c := &Config{}
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to read config %s: %s", path, err)
	}
	if err = yaml.Unmarshal(contents, c); err != nil {
		return nil, fmt.Errorf("Failed to parse config: %s", err)
	}
	if err = validate(c); err != nil {
		return nil, fmt.Errorf("Invalid config: %s", err)
	}
	printConfig(c, "Loaded this configuration:\n")
	return c, nil
}

func validate(c *Config) error {
	if len(c.DialInfo.Addrs) == 0 {
		return fmt.Errorf("dial_info.addrs must not be empty")
	}

	if c.DialInfo.DialInfo.Timeout == 0 {
		c.DialInfo.DialInfo.Timeout = 10 * time.Second
	}

	if c.DialInfo.DialInfo.Database == "" {
		return errors.New("dial_info.database is required")
	}
	if c.Collection == "" {
		return errors.New("collection is required")
	}

	// Check if certificate and key file exist
	if _, err := os.Stat(c.Server.Ssl.Cert); os.IsNotExist(err) {
		return fmt.Errorf("Failed to find certificate file (server.ssl.cert) \"%s\": %s", c.Server.Ssl.Cert, err)
	}
	if _, err := os.Stat(c.Server.Ssl.CertKey); os.IsNotExist(err) {
		return fmt.Errorf("Failed to find certificate key file (server.ssl.cert_key) \"%s\": %s", c.Server.Ssl.CertKey, err)
	}

	// Check if HTTP route begins with /
	if c.Server.Route == "" || !strings.HasPrefix(c.Server.Route, "/") {
		return fmt.Errorf("HTTP route (server.route) must start with /: \"%s\"", c.Server.Route)
	}

	return nil
}

func printConfig(c *Config, msg string) {
	/*d, err := yaml.Marshal(c)
	if err != nil {
		glog.Fatalf("error: %v", err)
	}
	glog.Info(msg + "\n------\n" + string(d) + "------\n")*/
	glog.Info(msg)
}
