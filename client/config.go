package client

type Config struct {
	URI      string `conf:"uri"`
	User     string `conf:"user"`
	Password string `conf:"passwd"`
	MaxConn  uint64 `conf:"maxconn"`
}

//func (c *Config) Id() string {
//	sha := sha1.New()
//	sha.Write([]byte(c.User))
//	sha.Write([]byte(c.Password))
//	sha.Write([]byte(c.URI))
//	buf := make([]byte, 8)
//	binary.BigEndian.PutUint64(buf, c.MaxConn)
//	sha.Write(buf)
//	return hex.EncodeToString(sha.Sum([]byte{}))
//}
