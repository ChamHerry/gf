package hello

import (
	"github.com/test/invalid-project/internal/dao"
	"github.com/test/invalid-project/internal/model"
)

type MyController struct{}

func NewController() *MyController {
	return &MyController{}
}

func (c *MyController) Hello() {
	_ = dao.User
	_ = model.User{}
}
