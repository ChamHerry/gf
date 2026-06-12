// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package internal

import (
	"github.com/gogf/gf/v2/database/gdb"
)

// HelloDao is the dao for table hello.
type HelloDao struct {
	DB    gdb.DB
	Table string
}

// NewHelloDao creates and returns a new HelloDao.
func NewHelloDao() *HelloDao {
	return &HelloDao{}
}
