package model

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jiajia556/godo/internal/service"
	"github.com/jiajia556/godo/internal/template"
	"github.com/jiajia556/godo/internal/utils"
	"github.com/jiajia556/godo/templates"
)

func genModel(from string) {
	defer runPostGenerationTasks()

	var createTables []string
	var err error
	if strings.HasSuffix(from, ".sql") {
		createTables, err = extractCreateTablesFromSqlFile(from)
	} else {
		createTables, err = extractCreateTablesFromConfigFile(from)
	}
	if err != nil {
		log.Fatalf("Failed to extract create tables: %v", err)
	}

	recordContent, err := templates.TemplateFS.ReadFile("default/internal/common/models/record.go.templ")
	if err != nil {
		utils.OutputFatal(fmt.Sprintf("Error reading record template: %v", err))
	}
	listContent, err := templates.TemplateFS.ReadFile("default/internal/common/models/list.go.templ")
	if err != nil {
		utils.OutputFatal(fmt.Sprintf("Error reading list template: %v", err))
	}

	for _, createTable := range createTables {
		generateModelFromSQL(createTable, string(recordContent), string(listContent))
	}
}

func extractCreateTablesFromSqlFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var createTables []string
	scanner := bufio.NewScanner(file)
	var currentStmt strings.Builder
	capturing := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "--") || strings.HasPrefix(line, "/*") {
			continue
		}
		if strings.HasPrefix(line, "CREATE TABLE") {
			capturing = true
			currentStmt.WriteString(line + "\n")
			continue
		}

		if strings.HasPrefix(line, ")") && strings.HasSuffix(line, ";") {
			capturing = false
			currentStmt.WriteString(line + "\n")
			createTables = append(createTables, currentStmt.String())
			currentStmt = strings.Builder{}
		}

		if capturing {
			currentStmt.WriteString(line + "\n")
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Flush the last statement if we reached EOF while capturing.
	if capturing && currentStmt.Len() > 0 {
		createTables = append(createTables, currentStmt.String())
	}

	return createTables, nil
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
		return nil, errors.New("Failed to connect to database:" + err.Error())
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		return nil, errors.New("Database ping failed:" + err.Error())
	}

	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		return nil, errors.New("Failed to execute SHOW TABLES:" + err.Error())
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err = rows.Scan(&tableName); err != nil {
			log.Printf("Failed to scan table name: %v", err)
			continue
		}
		tables = append(tables, tableName)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.New("Failed to iterate over rows:" + err.Error())
	}

	var createTables []string
	for _, table := range tables {
		query := fmt.Sprintf("SHOW CREATE TABLE `%s`", table)
		var tableName, createStmt string
		err = db.QueryRow(query).Scan(&tableName, &createStmt)
		if err != nil {
			log.Printf("Failed to get create statement for table %s: %v", table, err)
			continue
		}
		createStmt += ";"
		createTables = append(createTables, createStmt)
	}
	return createTables, nil
}

func generateModelFromSQL(sql, recordTmpl, listTmpl string) {
	// Generate model structure from SQL
	structText, structName, err := GenerateModelStruct(sql)
	if err != nil {
		utils.OutputFatal(fmt.Sprintf("Error generating model struct: %v", err))
		return
	}

	// Prepare model package name
	modelPkg := strings.ToLower(structName)

	// Generate record file
	generateModelFile(modelPkg, structName, structText, recordTmpl, "record.go")

	// Generate list file
	generateModelFile(modelPkg, structName, structText, listTmpl, "list.go")
}

func generateModelFile(modelPkg, structName, structText, templateContent, fileName string) {
	// Set up file paths
	var err error
	path := filepath.Join("internal/common/models", modelPkg, fileName)
	path, err = service.GetAbsPath(path)
	if err != nil {
		utils.OutputFatal(fmt.Sprintf("Error getting absolute path: %v", err))
	}

	// Skip if file already exists
	if utils.IsFileExists(path) {
		return
	}

	// Prepare template data
	projectName, err := service.GetProjectName()
	if err != nil {
		utils.OutputFatal(fmt.Sprintf("Error getting project name: %v", err))
	}

	data := template.ModelData{
		ModelPkg:        modelPkg,
		ProjectName:     projectName,
		ModelStruct:     structText,
		ModelStructName: structName,
	}

	// Create directory structure
	dir := filepath.Dir(path)
	if err = os.MkdirAll(dir, os.ModePerm); err != nil {
		utils.OutputFatal(fmt.Sprintf("Error creating directory %s: %v", dir, err))
		return
	}

	// Generate file from template
	if err = template.CreateFile(templateContent, data, path); err != nil {
		utils.OutputFatal(fmt.Sprintf("Error creating %s: %v", fileName, err))
	}
}

func runPostGenerationTasks() {
	modelsPath, err := service.GetAbsPath("internal/common/models")
	if err != nil {
		utils.OutputFatal(fmt.Sprintf("Error getting models path: %v", err))
	}
	utils.RunCommand("goimports", "-w", modelsPath)
	utils.RunCommand("go", "mod", "tidy")
}
