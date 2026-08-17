package model

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jiajia556/godo/internal/service"
	"github.com/jiajia556/godo/internal/template"
	"github.com/jiajia556/godo/internal/utils"
	"github.com/jiajia556/godo/templates"
)

func genModel(from string) error {
	var createTables []string
	var err error
	if strings.EqualFold(filepath.Ext(from), ".sql") {
		createTables, err = extractCreateTablesFromSqlFile(from)
	} else {
		createTables, err = extractCreateTablesFromConfigFile(from)
	}
	if err != nil {
		return fmt.Errorf("extract CREATE TABLE statements from %s: %w", from, err)
	}
	if len(createTables) == 0 {
		return fmt.Errorf("no CREATE TABLE statements found in %s", from)
	}

	recordContent, err := templates.TemplateFS.ReadFile("default/internal/common/models/record.go.templ")
	if err != nil {
		return fmt.Errorf("read record template: %w", err)
	}
	listContent, err := templates.TemplateFS.ReadFile("default/internal/common/models/list.go.templ")
	if err != nil {
		return fmt.Errorf("read list template: %w", err)
	}
	modelContent, err := templates.TemplateFS.ReadFile("default/internal/common/models/model.go.templ")
	if err != nil {
		return fmt.Errorf("read model template: %w", err)
	}

	var generatedFiles []string
	for _, createTable := range createTables {
		files, err := generateModelFromSQL(createTable, string(recordContent), string(listContent), string(modelContent))
		if err != nil {
			return err
		}
		generatedFiles = append(generatedFiles, files...)
	}
	return runPostGenerationTasks(generatedFiles)
}

func extractCreateTablesFromSqlFile(filePath string) ([]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read SQL file: %w", err)
	}

	statements, err := splitSQLStatements(string(content))
	if err != nil {
		return nil, fmt.Errorf("parse SQL file: %w", err)
	}
	createTables := make([]string, 0, len(statements))
	for _, statement := range statements {
		if createTableHeaderRE.MatchString(statement) {
			createTables = append(createTables, strings.TrimSpace(statement)+";")
		}
	}

	return createTables, nil
}

func splitSQLStatements(content string) ([]string, error) {
	var statements []string
	var current strings.Builder
	var quote byte
	inLineComment := false
	inBlockComment := false
	escaped := false

	flush := func() {
		if statement := strings.TrimSpace(current.String()); statement != "" {
			statements = append(statements, statement)
		}
		current.Reset()
	}

	for i := 0; i < len(content); i++ {
		ch := content[i]
		next := byte(0)
		if i+1 < len(content) {
			next = content[i+1]
		}

		if inLineComment {
			if ch == '\n' {
				inLineComment = false
				current.WriteByte(ch)
			}
			continue
		}
		if inBlockComment {
			if ch == '*' && next == '/' {
				inBlockComment = false
				i++
			}
			continue
		}
		if quote != 0 {
			current.WriteByte(ch)
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' && quote != '`' {
				escaped = true
				continue
			}
			if ch == quote {
				if next == quote {
					current.WriteByte(next)
					i++
					continue
				}
				quote = 0
			}
			continue
		}

		switch {
		case ch == '-' && next == '-':
			current.WriteByte(' ')
			inLineComment = true
			i++
		case ch == '#':
			current.WriteByte(' ')
			inLineComment = true
		case ch == '/' && next == '*':
			current.WriteByte(' ')
			inBlockComment = true
			i++
		case ch == '\'' || ch == '"' || ch == '`':
			quote = ch
			current.WriteByte(ch)
		case ch == ';':
			flush()
		default:
			current.WriteByte(ch)
		}
	}

	if quote != 0 {
		return nil, fmt.Errorf("unterminated %q quote", quote)
	}
	if inBlockComment {
		return nil, fmt.Errorf("unterminated block comment")
	}
	flush()
	return statements, nil
}

func extractCreateTablesFromConfigFile(filePath string) ([]string, error) {
	err := service.LoadConfig(filePath)
	if err != nil {
		return nil, err
	}
	conf := service.GetConfig()
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		conf.Mysql.User,
		conf.Mysql.Password,
		conf.Mysql.Host,
		conf.Mysql.Port,
		conf.Mysql.DBName,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		return nil, fmt.Errorf("execute SHOW TABLES: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err = rows.Scan(&tableName); err != nil {
			return nil, fmt.Errorf("scan table name: %w", err)
		}
		tables = append(tables, tableName)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate over tables: %w", err)
	}

	var createTables []string
	for _, table := range tables {
		query := fmt.Sprintf("SHOW CREATE TABLE `%s`", table)
		var tableName, createStmt string
		err = db.QueryRow(query).Scan(&tableName, &createStmt)
		if err != nil {
			return nil, fmt.Errorf("get CREATE TABLE statement for %s: %w", table, err)
		}
		createStmt += ";"
		createTables = append(createTables, createStmt)
	}
	return createTables, nil
}

func generateModelFromSQL(sql, recordTmpl, listTmpl, modelTmpl string) ([]string, error) {
	// Generate model structure from SQL
	structText, structName, tableName, err := GenerateModelStruct(sql)
	if err != nil {
		return nil, fmt.Errorf("generate model struct: %w", err)
	}

	// Prepare model package name
	modelPkg := strings.ToLower(structName)

	// Generate record file
	generatedFiles := make([]string, 0, 3)
	if path, err := generateModelFile(modelPkg, tableName, structName, structText, sql, recordTmpl, "record.go"); err != nil {
		return nil, err
	} else if path != "" {
		generatedFiles = append(generatedFiles, path)
	}

	// Generate list file
	if path, err := generateModelFile(modelPkg, tableName, structName, structText, sql, listTmpl, "list.go"); err != nil {
		return nil, err
	} else if path != "" {
		generatedFiles = append(generatedFiles, path)
	}

	// Generate model file
	if path, err := generateModelFile(modelPkg, tableName, structName, structText, sql, modelTmpl, "model.go"); err != nil {
		return nil, err
	} else if path != "" {
		generatedFiles = append(generatedFiles, path)
	}
	return generatedFiles, nil
}

func generateModelFile(modelPkg, tableName, structName, structText, createDDL, templateContent, fileName string) (string, error) {
	// Set up file paths
	var err error
	path := filepath.Join("internal/common/models", modelPkg, fileName)
	path, err = service.GetAbsPath(path)
	if err != nil {
		return "", fmt.Errorf("resolve model file %s: %w", fileName, err)
	}

	// Skip if file already exists
	if utils.IsFileExists(path) {
		return "", nil
	}

	// Prepare template data
	projectName, err := service.GetProjectName()
	if err != nil {
		return "", fmt.Errorf("get project name: %w", err)
	}

	createDDL = strings.ReplaceAll(createDDL, "\r\n", " ")
	createDDL = strings.ReplaceAll(createDDL, "\n", " ")

	data := template.ModelData{
		ModelPkg:        modelPkg,
		ProjectName:     projectName,
		ModelStruct:     structText,
		ModelStructName: structName,
		TableName:       tableName,
		CreateDDL:       createDDL,
		UseTime:         strings.Contains(structText, "time.Time"),
		UseDecimal:      strings.Contains(structText, "decimal.Decimal"),
	}

	// Create directory structure
	dir := filepath.Dir(path)
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create model directory %s: %w", dir, err)
	}

	// Generate file from template
	if err = template.CreateFile(templateContent, data, path); err != nil {
		return "", fmt.Errorf("write model file %s: %w", fileName, err)
	}
	return path, nil
}

func runPostGenerationTasks(generatedFiles []string) error {
	if err := utils.FormatGoFiles(generatedFiles...); err != nil {
		return fmt.Errorf("format generated models: %w", err)
	}
	projectRoot, err := service.GetProjectRoot()
	if err != nil {
		return fmt.Errorf("get project root: %w", err)
	}
	output, err := utils.NewCommandRunner().WithDir(projectRoot).RunCommandOutput("go", "mod", "tidy")
	if err != nil {
		return fmt.Errorf("run go mod tidy: %w\n%s", err, strings.TrimSpace(output))
	}
	return nil
}
