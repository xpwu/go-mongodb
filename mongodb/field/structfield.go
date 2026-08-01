package field

import (
	"github.com/xpwu/go-db-mongo/mongodb"
	"github.com/xpwu/go-db-mongo/mongodb/filter"
	"github.com/xpwu/go-db-mongo/mongodb/updater"
)

type UserInfo struct {
	Name string
	Addr AddrInfo
	Age  int
}

type AddrInfo struct {
	Province string
	City     string
	Zone     string
}

type UserInfoField interface {
	mongodb.Field
	filter.ComparableFilter[UserInfo]
	updater.BaseUpdater[UserInfo]

	Name() StringField
	Age() IntField
}

type userInfoField struct {
	BaseField[UserInfo]
}

var UserInfoColl = NewUserInfoField("")

func NewUserInfoField(name string) UserInfoField {
	return &userInfoField{BaseField: BaseField[UserInfo]{name}}
}

func (u *userInfoField) Name() StringField {
	return NewStringField(SubField(u.FullName(), "Name"))
}

func (u *userInfoField) Age() IntField {
	return NewIntField(SubField(u.FullName(), "Age"))
}
