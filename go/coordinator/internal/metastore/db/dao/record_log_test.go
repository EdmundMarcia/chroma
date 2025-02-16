package dao

import (
	"github.com/chroma/chroma-coordinator/internal/metastore/db/dbcore"
	"github.com/chroma/chroma-coordinator/internal/metastore/db/dbmodel"
	"github.com/chroma/chroma-coordinator/internal/types"
	"github.com/pingcap/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
	"testing"
)

type RecordLogDbTestSuite struct {
	suite.Suite
	db           *gorm.DB
	Db           *recordLogDb
	t            *testing.T
	collectionId types.UniqueID
	records      [][]byte
}

func (suite *RecordLogDbTestSuite) SetupSuite() {
	log.Info("setup suite")
	suite.db = dbcore.ConfigDatabaseForTesting()
	suite.Db = &recordLogDb{
		db: suite.db,
	}
	suite.collectionId = types.NewUniqueID()
	suite.records = make([][]byte, 0, 5)
	suite.records = append(suite.records, []byte("test1"), []byte("test2"),
		[]byte("test3"), []byte("test4"), []byte("test5"))
}

func (suite *RecordLogDbTestSuite) SetupTest() {
	log.Info("setup test")
	suite.db.Migrator().DropTable(&dbmodel.RecordLog{})
	suite.db.Migrator().CreateTable(&dbmodel.RecordLog{})
}

func (suite *RecordLogDbTestSuite) TearDownTest() {
	log.Info("teardown test")
	suite.db.Migrator().DropTable(&dbmodel.RecordLog{})
	suite.db.Migrator().CreateTable(&dbmodel.RecordLog{})
}

func (suite *RecordLogDbTestSuite) TestRecordLogDb_PushLogs() {
	// run push logs in transaction
	// id: 0,
	// offset: 0, 1, 2
	// records: test1, test2, test3
	count, err := suite.Db.PushLogs(suite.collectionId, suite.records[:3])
	assert.NoError(suite.t, err)
	assert.Equal(suite.t, 3, count)

	// verify logs are pushed
	var recordLogs []*dbmodel.RecordLog
	suite.db.Where("collection_id = ?", types.FromUniqueID(suite.collectionId)).Find(&recordLogs)
	assert.Len(suite.t, recordLogs, 3)
	for index := range recordLogs {
		assert.Equal(suite.t, int64(index+1), recordLogs[index].ID)
		assert.Equal(suite.t, suite.records[index], *recordLogs[index].Record)
	}

	// run push logs in transaction
	// id: 1,
	// offset: 0, 1
	// records: test4, test5
	count, err = suite.Db.PushLogs(suite.collectionId, suite.records[3:])
	assert.NoError(suite.t, err)
	assert.Equal(suite.t, 2, count)

	// verify logs are pushed
	suite.db.Where("collection_id = ?", types.FromUniqueID(suite.collectionId)).Find(&recordLogs)
	assert.Len(suite.t, recordLogs, 5)
	for index := range recordLogs {
		assert.Equal(suite.t, int64(index+1), recordLogs[index].ID, "id mismatch for index %d", index)
		assert.Equal(suite.t, suite.records[index], *recordLogs[index].Record, "record mismatch for index %d", index)
	}
}

func (suite *RecordLogDbTestSuite) TestRecordLogDb_PullLogsFromID() {
	// push some logs
	count, err := suite.Db.PushLogs(suite.collectionId, suite.records[:3])
	assert.NoError(suite.t, err)
	assert.Equal(suite.t, 3, count)
	count, err = suite.Db.PushLogs(suite.collectionId, suite.records[3:])
	assert.NoError(suite.t, err)
	assert.Equal(suite.t, 2, count)

	// pull logs from id 0 batch_size 3
	var recordLogs []*dbmodel.RecordLog
	recordLogs, err = suite.Db.PullLogs(suite.collectionId, 0, 3)
	assert.NoError(suite.t, err)
	assert.Len(suite.t, recordLogs, 3)
	for index := range recordLogs {
		assert.Equal(suite.t, int64(index+1), recordLogs[index].ID, "id mismatch for index %d", index)
		assert.Equal(suite.t, suite.records[index], *recordLogs[index].Record, "record mismatch for index %d", index)
	}

	// pull logs from id 0 batch_size 6
	recordLogs, err = suite.Db.PullLogs(suite.collectionId, 0, 6)
	assert.NoError(suite.t, err)
	assert.Len(suite.t, recordLogs, 5)

	for index := range recordLogs {
		assert.Equal(suite.t, int64(index+1), recordLogs[index].ID, "id mismatch for index %d", index)
		assert.Equal(suite.t, suite.records[index], *recordLogs[index].Record, "record mismatch for index %d", index)
	}

	// pull logs from id 3 batch_size 4
	recordLogs, err = suite.Db.PullLogs(suite.collectionId, 3, 4)
	assert.NoError(suite.t, err)
	assert.Len(suite.t, recordLogs, 3)
	for index := range recordLogs {
		assert.Equal(suite.t, int64(index+3), recordLogs[index].ID, "id mismatch for index %d", index)
		assert.Equal(suite.t, suite.records[index+2], *recordLogs[index].Record, "record mismatch for index %d", index)
	}
}

func TestRecordLogDbTestSuite(t *testing.T) {
	testSuite := new(RecordLogDbTestSuite)
	testSuite.t = t
	suite.Run(t, testSuite)
}
