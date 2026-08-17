package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractCreateTablesFromSQLFileHandlesCommonMySQLDDL(t *testing.T) {
	sqlPath := filepath.Join(t.TempDir(), "schema.SQL")
	content := `
-- schema setup should be ignored
SET NAMES utf8mb4;

create table if not exists ` + "`app`.`users`" + ` (
  ` + "`ID`" + ` int NOT NULL,
  ` + "`note`" + ` varchar(255) DEFAULT 'contains;semicolon',
  PRIMARY KEY (` + "`ID`" + `)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

/* a block comment; with punctuation */
CREATE TABLE logs (
  message varchar(255) COMMENT 'a closing parenthesis ) is text'
);
`
	if err := os.WriteFile(sqlPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	statements, err := extractCreateTablesFromSqlFile(sqlPath)
	if err != nil {
		t.Fatalf("extractCreateTablesFromSqlFile() error = %v", err)
	}
	if len(statements) != 2 {
		t.Fatalf("got %d CREATE TABLE statements, want 2: %v", len(statements), statements)
	}
	if !strings.Contains(statements[0], "ENGINE=InnoDB") {
		t.Fatalf("first statement lost table options: %s", statements[0])
	}
	if !strings.Contains(statements[0], "contains;semicolon") {
		t.Fatalf("first statement split at a quoted semicolon: %s", statements[0])
	}
}

func TestGenerateModelStructHandlesQualifiedTableAndConstraints(t *testing.T) {
	ddl := "CREATE TABLE IF NOT EXISTS `app`.`users` (\n" +
		"`ID` int NOT NULL,\n" +
		"`display_name` varchar(100) DEFAULT 'KeepCase',\n" +
		"PRIMARY KEY (`ID`),\n" +
		"UNIQUE KEY `uk_name` (`display_name`)\n" +
		") ENGINE=InnoDB;"

	generated, structName, tableName, err := GenerateModelStruct(ddl)
	if err != nil {
		t.Fatalf("GenerateModelStruct() error = %v", err)
	}
	if structName != "Users" || tableName != "users" {
		t.Fatalf("names = %q, %q; want Users, users", structName, tableName)
	}
	if !strings.Contains(generated, "ID       int32") {
		t.Fatalf("ID type did not preserve the SQL int type:\n%s", generated)
	}
	if !strings.Contains(generated, "column:ID;notNull;primaryKey") {
		t.Fatalf("primary key tags are missing:\n%s", generated)
	}
	if strings.Contains(generated, "UkName") {
		t.Fatalf("table constraint was emitted as a field:\n%s", generated)
	}
	if !strings.Contains(generated, "default:KeepCase") {
		t.Fatalf("string default value changed case:\n%s", generated)
	}
}

func TestGenerateModelStructIgnoresHeaderCommentParentheses(t *testing.T) {
	ddl := "CREATE TABLE users /* ignored ( comment ) */ (`id` bigint);"
	generated, _, _, err := GenerateModelStruct(ddl)
	if err != nil {
		t.Fatalf("GenerateModelStruct() error = %v", err)
	}
	if !strings.Contains(generated, "Id       int64") {
		t.Fatalf("field block was parsed incorrectly:\n%s", generated)
	}
}

func TestSplitSQLStatementsRejectsUnterminatedInput(t *testing.T) {
	if _, err := splitSQLStatements("CREATE TABLE users (name varchar(20) DEFAULT 'oops);"); err == nil {
		t.Fatal("splitSQLStatements() accepted an unterminated quote")
	}
	if _, err := splitSQLStatements("/* unterminated"); err == nil {
		t.Fatal("splitSQLStatements() accepted an unterminated block comment")
	}
}
