package mysql

// ConnectionConfig stores connection configs.
type ConnectionConfig struct {
	Host            string
	Port            int
	Username        string
	Password        string
	Database        string
	ConnectionLimit int
}

// Config provides configs for mysql.
type Config struct {
	Master ConnectionConfig
	Slave  *ConnectionConfig
}
