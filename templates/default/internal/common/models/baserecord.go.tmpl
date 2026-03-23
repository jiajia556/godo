package models

import (
	"github.com/jiajia556/tool-box/mysqlx"
)

type DBRecord interface {
	SetSession(session mysqlx.Session)
	SetModel(data mysqlx.Model)
}
type BaseRecord[T mysqlx.Model] struct {
	mysqlx.Session
	Model T
}

func (m *BaseRecord[T]) Exists() bool {
	return m.Model.ID() > 0
}

func (m *BaseRecord[T]) Create() error {
	return m.DB().Create(m.Model).Error
}

func (m *BaseRecord[T]) Update() error {
	return m.DB().Save(m.Model).Error
}

func (m *BaseRecord[T]) Read(id uint64) error {
	return m.DB().Take(m.Model, id).Error
}

func (m *BaseRecord[T]) Delete() error {
	return m.DB().Delete(m.Model).Error
}

func (m *BaseRecord[T]) SetSession(session mysqlx.Session) {
	m.Session = session
}

func (m *BaseRecord[T]) SetModel(data mysqlx.Model) {
	m.Model = data.(T)
}
