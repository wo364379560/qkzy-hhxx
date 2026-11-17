package datasource

type datasource struct {
	Mysql *mysqlClient
	Redis *redisClient
}

func (ds *datasource) Init() {
	ds.Mysql.openMysql()
	ds.Redis.openRedis()
}

func getDatasource() *datasource {
	return &datasource{
		Mysql: &mysqlClient{},
		Redis: &redisClient{},
	}
}

var Datasource *datasource = getDatasource()
