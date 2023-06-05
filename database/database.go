package database

import (
	"log"
	"sync"

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
		}
	})

	return instance
}
