package database

import (
	"log"
	"sync"

	"github.com/go-redis/redis/v8"

	_ "github.com/go-sql-driver/mysql"
	"github.com/yeom-c/data-api/app"
	"xorm.io/xorm"
)

var once sync.Once
var instance *database

type database struct {
	DataConn          *xorm.Engine
	StaticDataGenConn *xorm.Engine
	StaticDataConn    map[string]*xorm.Engine
	SessionConn       map[string]*redis.Client
}

func Database() *database {
	once.Do(func() {
		if instance == nil {
			instance = &database{}

			dataConn, err := xorm.NewEngine(app.Config().DbDriver, app.Config().DbConn)
			if err != nil {
				log.Fatal("failed to connect quasar_data database: ", err)
			}

			staticDataGenConn, err := xorm.NewEngine(app.Config().DbStaticDataGenDriver, app.Config().DbStaticDataGenConn)
			if err != nil {
				log.Fatal("failed to connect static_data_gen database: ", err)
			}

			staticDataLocalConn, err := xorm.NewEngine(app.Config().DbStaticDataLocalDriver, app.Config().DbStaticDataLocalConn)
			if err != nil {
				log.Fatal("failed to connect local static_data database: ", err)
			}

			staticDataTestConn, err := xorm.NewEngine(app.Config().DbStaticDataTestDriver, app.Config().DbStaticDataTestConn)
			if err != nil {
				log.Fatal("failed to connect test static_data database: ", err)
			}

			staticDataDevConn, err := xorm.NewEngine(app.Config().DbStaticDataDevDriver, app.Config().DbStaticDataDevConn)
			if err != nil {
				log.Fatal("failed to connect dev static_data database: ", err)
			}

			staticDataStagingConn, err := xorm.NewEngine(app.Config().DbStaticDataStagingDriver, app.Config().DbStaticDataStagingConn)
			if err != nil {
				log.Fatal("failed to connect staging static_data database: ", err)
			}

			staticDataProductionConn, err := xorm.NewEngine(app.Config().DbStaticDataProductionDriver, app.Config().DbStaticDataProductionConn)
			if err != nil {
				log.Fatal("failed to connect production static_data database: ", err)
			}

			if app.Config().SqlShow {
				dataConn.ShowSQL(true)
				staticDataGenConn.ShowSQL(true)
				staticDataLocalConn.ShowSQL(true)
				staticDataTestConn.ShowSQL(true)
				staticDataDevConn.ShowSQL(true)
				staticDataStagingConn.ShowSQL(true)
				staticDataProductionConn.ShowSQL(true)
			}

			instance.DataConn = dataConn
			instance.StaticDataGenConn = staticDataGenConn
			instance.StaticDataConn = map[string]*xorm.Engine{
				"local":      staticDataLocalConn,
				"test":       staticDataTestConn,
				"develop":    staticDataDevConn,
				"staging":    staticDataStagingConn,
				"production": staticDataProductionConn,
			}

			instance.SessionConn = map[string]*redis.Client{}
			if app.Config().RedisGameServerSessionLocalConn != "" {
				sessionLocal, err := redis.ParseURL(app.Config().RedisGameServerSessionLocalConn)
				if err != nil {
					log.Fatal("failed to connect local session redis: ", err)
				}
				instance.SessionConn["local"] = redis.NewClient(sessionLocal)
			}

			if app.Config().RedisGameServerSessionTestConn != "" {
				sessionTest, err := redis.ParseURL(app.Config().RedisGameServerSessionTestConn)
				if err != nil {
					log.Fatal("failed to connect test session redis: ", err)
				}
				instance.SessionConn["test"] = redis.NewClient(sessionTest)
			}

			if app.Config().RedisGameServerSessionDevConn != "" {
				sessionDev, err := redis.ParseURL(app.Config().RedisGameServerSessionDevConn)
				if err != nil {
					log.Fatal("failed to connect dev session redis: ", err)
				}
				instance.SessionConn["develop"] = redis.NewClient(sessionDev)
			}

			if app.Config().RedisGameServerSessionStagingConn != "" {
				sessionStaging, err := redis.ParseURL(app.Config().RedisGameServerSessionStagingConn)
				if err != nil {
					log.Fatal("failed to connect staging session redis: ", err)
				}
				instance.SessionConn["staging"] = redis.NewClient(sessionStaging)
			}

			if app.Config().RedisGameServerSessionProductionConn != "" {
				sessionProduction, err := redis.ParseURL(app.Config().RedisGameServerSessionProductionConn)
				if err != nil {
					log.Fatal("failed to connect production session redis: ", err)
				}
				instance.SessionConn["production"] = redis.NewClient(sessionProduction)
			}
		}
	})

	return instance
}
