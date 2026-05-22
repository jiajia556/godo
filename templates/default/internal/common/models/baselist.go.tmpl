package models

import "github.com/jiajia556/tool-box/mysqlx"

type BaseList[T mysqlx.Model, R DBRecord] struct {
	mysqlx.Session
	Records       *[]T
	total         int64
	RecordFactory func() R
}

func (l *BaseList[T, R]) FindAll() error {
	return l.DB().Find(l.Records).Error
}

func (l *BaseList[T, R]) IsEmpty() bool {
	return len(*l.Records) == 0
}

func (l *BaseList[T, R]) GetTotal() int64 {
	return l.total
}

func (l *BaseList[T, R]) SetTotal(total int64) {
	l.total = total
}

func (l *BaseList[T, R]) Foreach(fn func(key int, value R) bool) {
	if l.RecordFactory == nil {
		panic("RecordFactory is nil")
	}
	for k, data := range *l.Records {
		r := l.RecordFactory()
		r.SetSession(l.Session)
		r.SetModel(data)
		if !fn(k, r) {
			break
		}
	}
}
