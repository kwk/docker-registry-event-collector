package settings

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

const (
	// DefaultDbHost is the default hostname or IP on which the MongoDB server
	// runs.
	DefaultDbHost = "127.0.0.1"
	// DefaultDbPort is the default port on which the MongoDB server runs.
	DefaultDbPort = 27017
	// DefaultDbUser is the default port on which the MongoDB server runs.
	DefaultDbUser = ""
	// DefaultDbPassword is the default username with which to connect to
	// the MongoDB server.
	DefaultDbPassword = ""
	// DefaultDbName is the default database name that is used on the MongoDB
	// server.
	DefaultDbName = "docker-registry-db"

	// DefaultEndpointListenOnIP is the default IP on which to listen for docker
	// registry events.
	DefaultEndpointListenOnIP = "0.0.0.0"
	// DefaultEndpointListenOnPort is the default port on which to listen for
	// docker registry events.
	DefaultEndpointListenOnPort = 10443
	// DefaultEndpointCertPath is the default path to SSL certificate with which
	// to secure the endpoint that receices docker events.
	DefaultEndpointCertPath = "certs/domain.crt"
	// DefaultEndpointCertKeyPath is the default path the SSL key that is used in
	// conjunction with the SSL certificate.
	DefaultEndpointCertKeyPath = "certs/domain.key"
	// DefaultEndpointRoute is the default HTTP route at which the HTTP endpoint
	// accepts post requests from the docker registry.
	DefaultEndpointRoute = "/events"
)

// Settings for the mongo db backend and the HTTP endpoint frontend.
type Settings struct {
	// DbHost is the MongoDB hostname or IP used to connect to.
	DbHost string
	// DbPort is the MongoDB port to connect to.
	DbPort uint
	// DbUser is the username with which to connect to the MongoDB server.
	DbUser string
	// DbPassword is the password with which to connect to the MongoDB server.
	DbPassword string
	// DbName is the database name that will be used on the MongoDB server.
	DbName string
	// EndpointListenOnIP is the IP on which to listen for docker registry events.
	EndpointListenOnIP string
	// EndpointListenOnPort is the port on which to listen for docker registry
	// events.
	EndpointListenOnPort uint
	// EndpointCertPath is the filepath to the SSL certificate with which the HTTP
	// server will be secured to accept docker registry events.
	EndpointCertPath string
	// EndpointCertKeyPath is used in conjunction with EndpointCertPath.
	EndpointCertKeyPath string
	// EndpointRoute is the HTTP route at which the HTTP endpoint accepts post
	// requests from the docker registry.
	EndpointRoute string
}

// GetMongoDBConnectionString builds a connection string for the mongo backend
// and returns it to you for your convenience. Depending on whether a username
// or passoword is given, this string will be included in the connection string.
func (s Settings) GetMongoDBConnectionString() string {
	var mongoConnStr string
	if s.DbUser != "" && s.DbPassword != "" {
		mongoConnStr = fmt.Sprintf("mongodb://%s:%s@%s:%d/%s",
			s.DbUser, s.DbPassword, s.DbHost, s.DbPort, s.DbName)
	} else {
		mongoConnStr = fmt.Sprintf("mongodb://%s:%d/%s", s.DbHost, s.DbPort, s.DbName)
	}
	return mongoConnStr
}

// GetEndpointConnectionString builds and returns a string with the IP and port
// separated by a colon. Nothing special but anyway.
func (s Settings) GetEndpointConnectionString() string {
	return fmt.Sprintf("%s:%d", s.EndpointListenOnIP, s.EndpointListenOnPort)
}

// CreateFromCommandLineFlags parses all flags from the command line and returns
// an initialized Settings object and an error object if any. For instance if it
// cannot find the SSL certificate file or the SSL key file it will set the
// returned error appropriately.
func (Settings) CreateFromCommandLineFlags() (Settings, error) {
	var s Settings

	// Parse command line arguments
	flag.StringVar(&s.DbHost, "dbHost", DefaultDbHost, "mongo db host")
	flag.UintVar(&s.DbPort, "dbPort", DefaultDbPort, "mongo db host")
	flag.StringVar(&s.DbUser, "dbUser", DefaultDbUser, "mongo db username")
	flag.StringVar(&s.DbPassword, "dbPassword", DefaultDbPassword, "mongo db password")
	flag.StringVar(&s.DbName, "dbName", DefaultDbName, "mongo database name")
	flag.StringVar(&s.EndpointListenOnIP, "listenOnIp", DefaultEndpointListenOnIP, "On which IP to listen for notifications from a docker registry")
	flag.UintVar(&s.EndpointListenOnPort, "listenOnPort", DefaultEndpointListenOnPort, "On which port to listen for notifications from a docker registry")
	flag.StringVar(&s.EndpointCertPath, "certPath", DefaultEndpointCertPath, "Path to SSL certfificate file")
	flag.StringVar(&s.EndpointCertKeyPath, "certKeyPath", DefaultEndpointCertKeyPath, "Path to SSL certificate key")
	flag.StringVar(&s.EndpointRoute, "route", DefaultEndpointRoute, "HTTP route at which docker-registry events are accepted (must start with \"/\")")
	flag.Parse()

	// Check if certificate and key file exist
	if _, err := os.Stat(s.EndpointCertPath); os.IsNotExist(err) {
		return s, fmt.Errorf("Failed to find certificate file \"%s\": %s", s.EndpointCertPath, err)
	}
	if _, err := os.Stat(s.EndpointCertKeyPath); os.IsNotExist(err) {
		return s, fmt.Errorf("Failed to find certificate key file \"%s\": %s", s.EndpointCertKeyPath, err)
	}

	// Check if HTTP route begins with /
	if s.EndpointRoute == "" || !strings.HasPrefix(s.EndpointRoute, "/") {
		return s, fmt.Errorf("HTTP route must start with /: \"%s\"", s.EndpointRoute)
	}

	return s, nil
}

// Print all settings values in a nicely formatted way.
func (s Settings) Print() {
	fmt.Printf("Settings:\n")
	fmt.Printf("=========\n\n")
	fmt.Printf("  MongoDB:\n")
	fmt.Printf("  ---------\n")
	fmt.Printf("    DB Host     = %s\n", s.DbHost)
	fmt.Printf("    DB Port     = %d\n", s.DbPort)
	fmt.Printf("    DB User     = %s\n", s.DbUser)
	fmt.Printf("    DB Password = <not shown for security reasons>\n")
	fmt.Printf("    DB Name     = %s\n\n", s.DbName)
	fmt.Printf("  Docker endpoint:\n")
	fmt.Printf("  ----------------\n")
	fmt.Printf("    IP                   = %s\n", s.EndpointListenOnIP)
	fmt.Printf("    Port                 = %d\n", s.EndpointListenOnPort)
	fmt.Printf("    Certificate path     = %s\n", s.EndpointCertPath)
	fmt.Printf("    Certificate key path = %s\n", s.EndpointCertKeyPath)
	fmt.Printf("    HTTP route           = %s\n\n", s.EndpointRoute)
}
