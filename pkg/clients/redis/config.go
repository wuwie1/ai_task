package redis

type RedisConfig struct {
	// host:port address.
	Host     string `json:"host" yaml:"host" toml:"host"`
	Password string `json:"password" yaml:"password" toml:"password"`
	// Database to be selected after connecting to the server.
	Db int `json:"db" yaml:"db" toml:"db"`
	// Maximum number of socket connections.
	// Default is 100 connections per every available CPU as reported by runtime.GOMAXPROCS.
	PoolSize int `json:"pool_size" yaml:"poolSize" toml:"pool_size"`
	// Maximum number of retries before giving up.
	// Default is 3 retries; -1 (not 0) disables retries.
	MaxRetries int `json:"max_retries" yaml:"maxRetries" toml:"max_retries"`
	// Connection age at which client retires (closes) the connection.
	// Default is to not close aged connections.
	MaxConnAge int64 `json:"max_conn_age" yaml:"maxConnAge" toml:"max_conn_age"`
	// Dial timeout for establishing new connections.
	// Default is 5 seconds.
	DialTimeout int64 `json:"dial_timeout" yaml:"dialTimeout" toml:"dial_timeout"`
	// Timeout for socket reads. If reached, commands will fail
	// with a timeout instead of blocking. Use value -1 for no timeout and 0 for default.
	// Default is 3 seconds.
	ReadTimeout int64 `json:"read_timeout" yaml:"readTimeout" toml:"read_timeout"`
	// Timeout for socket writes. If reached, commands will fail
	// with a timeout instead of blocking.
	// Default is ReadTimeout.
	WriteTimeout int64 `json:"write_timeout" yaml:"writeTimeout" toml:"write_timeout"`
	// Minimum number of idle connections which is useful when establishing
	// new connection is slow.
	MinIdleConns int `json:"min_idle_conns" yaml:"minIdleConns" toml:"min_idle_conns"`
	// Amount of time client waits for connection if all connections
	// are busy before returning an error.
	// Default is ReadTimeout + 1 second.
	PoolTimeout int64 `json:"pool_timeout" yaml:"poolTimeout" toml:"pool_timeout"`
	// Amount of time after which client closes idle connections.
	// Should be less than server's timeout.
	// Default is 5 minutes. -1 disables idle timeout check.
	IdleTimeout int64 `json:"idle_timeout" yaml:"idleTimeout" toml:"idle_timeout"`
}

type RedisFailoverConfig struct {
	Hosts      []string `json:"hosts" yaml:"hosts" toml:"hosts"`
	Password   string   `json:"password" yaml:"password" toml:"password"`
	Db         int      `json:"db" yaml:"db" toml:"db"`
	PoolSize   int      `json:"pool_size" yaml:"poolSize" toml:"pool_size"`
	MasterName string   `json:"master_name" yaml:"masterName"`
}

type RedisClusterConfig struct {
	Hosts    []string `json:"hosts" yaml:"hosts" toml:"hosts"`
	Password string   `json:"password" yaml:"password" toml:"password"`
	// Maximum number of socket connections.
	// Default is 100 connections per every available CPU as reported by runtime.GOMAXPROCS.
	PoolSize int `json:"pool_size" yaml:"poolSize" toml:"pool_size"`
	// Maximum number of retries before giving up.
	// Default is 3 retries; -1 (not 0) disables retries.
	MaxRetries int `json:"max_retries" yaml:"maxRetries" toml:"max_retries"`
	// Connection age at which client retires (closes) the connection.
	// Default is to not close aged connections.
	MaxConnAge int64 `json:"max_conn_age" yaml:"maxConnAge" toml:"max_conn_age"`
	// Dial timeout for establishing new connections.
	// Default is 5 seconds.
	DialTimeout int64 `json:"dial_timeout" yaml:"dialTimeout" toml:"dial_timeout"`
	// Timeout for socket reads. If reached, commands will fail
	// with a timeout instead of blocking. Use value -1 for no timeout and 0 for default.
	// Default is 3 seconds.
	ReadTimeout int64 `json:"read_timeout" yaml:"readTimeout" toml:"read_timeout"`
	// Timeout for socket writes. If reached, commands will fail
	// with a timeout instead of blocking.
	// Default is ReadTimeout.
	WriteTimeout int64 `json:"write_timeout" yaml:"writeTimeout" toml:"write_timeout"`
	// Minimum number of idle connections which is useful when establishing
	// new connection is slow.
	MinIdleConns int `json:"min_idle_conns" yaml:"minIdleConns" toml:"min_idle_conns"`
	// Amount of time client waits for connection if all connections
	// are busy before returning an error.
	// Default is ReadTimeout + 1 second.
	PoolTimeout int64 `json:"pool_timeout" yaml:"poolTimeout" toml:"pool_timeout"`
	// Amount of time after which client closes idle connections.
	// Should be less than server's timeout.
	// Default is 5 minutes. -1 disables idle timeout check.
	IdleTimeout int64 `json:"idle_timeout" yaml:"idleTimeout" toml:"idle_timeout"`
}

func (rc *RedisConfig) DefaultConfig() {
	if rc.PoolSize == 0 {
		rc.PoolSize = 100
	}
	if rc.MaxRetries == 0 {
		rc.MaxRetries = 3
	}
	if rc.DialTimeout == 0 {
		rc.DialTimeout = 30
	}
	if rc.ReadTimeout == 0 {
		rc.ReadTimeout = 5
	}
	if rc.WriteTimeout == 0 {
		rc.WriteTimeout = 5
	}
	if rc.MinIdleConns == 0 {
		rc.MinIdleConns = 10
	}
	if rc.PoolTimeout == 0 {
		rc.PoolTimeout = 30
	}
	if rc.IdleTimeout == 0 {
		rc.IdleTimeout = 30
	}
}

func (rc *RedisClusterConfig) DefaultConfig() {
	if rc.PoolSize == 0 {
		rc.PoolSize = 100
	}
	if rc.MaxRetries == 0 {
		rc.MaxRetries = 3
	}
	if rc.DialTimeout == 0 {
		rc.DialTimeout = 30
	}
	if rc.ReadTimeout == 0 {
		rc.ReadTimeout = 5
	}
	if rc.WriteTimeout == 0 {
		rc.WriteTimeout = 5
	}
	if rc.MinIdleConns == 0 {
		rc.MinIdleConns = 10
	}
	if rc.PoolTimeout == 0 {
		rc.PoolTimeout = 30
	}
	if rc.IdleTimeout == 0 {
		rc.IdleTimeout = 30
	}
}
