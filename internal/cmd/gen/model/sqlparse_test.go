package model

import "testing"

func Test_mapTypeAndTags_CommonMySQLTypes(t *testing.T) {
	tests := []struct {
		name   string
		sql    string
		goType string
	}{
		{"tinyint bool", "tinyint(1)", "bool"},
		{"tinyint", "tinyint", "int8"},
		{"tinyint unsigned", "tinyint unsigned", "uint8"},
		{"smallint", "smallint", "int16"},
		{"int unsigned", "int unsigned", "uint32"},
		{"bigint unsigned", "bigint unsigned", "uint64"},
		{"decimal", "decimal(10,2)", "decimal.Decimal"},
		{"float", "float", "float32"},
		{"double", "double", "float64"},
		{"datetime", "datetime", "time.Time"},
		{"timestamp", "timestamp", "time.Time"},
		{"varchar", "varchar(255)", "string"},
		{"text", "text", "string"},
		{"json", "json", "[]byte"},
		{"blob", "blob", "[]byte"},
		{"enum", "enum('a','b')", "string"},
		{"bit(1)", "bit(1)", "bool"},
		{"bit(8)", "bit(8)", "[]byte"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := mapTypeAndTags(tt.sql)
			if got != tt.goType {
				t.Fatalf("mapTypeAndTags(%q) = %q, want %q", tt.sql, got, tt.goType)
			}
		})
	}
}
