package model

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jiajia556/godo/internal/utils"
)

type fieldInfo struct {
	name     string
	typeName string
	gormTags string
	jsonTag  string
}

var createTableHeaderRE = regexp.MustCompile("(?i)^\\s*CREATE\\s+(?:TEMPORARY\\s+)?TABLE\\s+(?:IF\\s+NOT\\s+EXISTS\\s+)?(?:(?:`[^`]+`|[A-Za-z0-9_$]+)\\s*\\.\\s*)?(?:`([^`]+)`|([A-Za-z0-9_$]+))")

// GenerateStruct generates Go struct definition from SQL create table statement
func GenerateModelStruct(sql string) (string, string, string, error) {
	tableName, fields, err := parseSQL(sql)
	if err != nil {
		return "", "", "", err
	}

	return buildStruct(tableName, fields), toCamelCase(tableName), utils.CamelToSnake(tableName), nil
}

func parseSQL(sql string) (string, []fieldInfo, error) {
	tableName, err := extractTableName(sql)
	if err != nil {
		return "", nil, err
	}

	fieldDefinitions, err := extractFieldDefinitions(sql)
	if err != nil {
		return "", nil, err
	}

	// Extract table-level primary key constraints like: PRIMARY KEY (`id`) or PRIMARY KEY (id, other_id)
	// so we can mark the corresponding fields even if the column definition itself doesn't contain "primary key".
	pkCols := extractPrimaryKeyColumns(fieldDefinitions)
	pkSet := make(map[string]struct{}, len(pkCols))
	for _, c := range pkCols {
		pkSet[strings.ToLower(c)] = struct{}{}
	}

	var fields []fieldInfo
	for _, def := range fieldDefinitions {
		fi, err := parseField(def, pkSet)
		if err != nil {
			return "", nil, err
		}
		if fi.name == "" {
			continue
		}
		fields = append(fields, fi)
	}

	return tableName, fields, nil
}

// extractPrimaryKeyColumns returns column names declared in table-level PRIMARY KEY constraints.
// Example defs:
//
//	PRIMARY KEY (`id`)
//	PRIMARY KEY (id, other_id)
//
// It is intentionally conservative: it only parses column identifiers inside the parentheses.
func extractPrimaryKeyColumns(defs []string) []string {
	var out []string
	// Capture content inside PRIMARY KEY ( ... ).
	re := regexp.MustCompile(`(?i)^\s*PRIMARY\s+KEY\s*\(([^\)]*)\)`)
	// Match identifiers optionally wrapped in backticks.
	identRe := regexp.MustCompile("[`]?([A-Za-z0-9_]+)[`]?")

	for _, d := range defs {
		m := re.FindStringSubmatch(strings.TrimSpace(d))
		if len(m) < 2 {
			continue
		}
		inside := m[1]
		idents := identRe.FindAllStringSubmatch(inside, -1)
		for _, im := range idents {
			if len(im) >= 2 {
				out = append(out, im[1])
			}
		}
	}
	return out
}

func extractTableName(sql string) (string, error) {
	matches := createTableHeaderRE.FindStringSubmatch(sql)
	if len(matches) < 3 {
		return "", fmt.Errorf("table name not found")
	}
	tableName := matches[1]
	if tableName == "" {
		tableName = matches[2]
	}
	return tableName, nil
}

// extractFieldDefinitions extracts column definitions from a CREATE TABLE statement.
// It uses a simple state machine (paren nesting + quote tracking) so commas in
// types, comments, or indexes won't break the split.
func extractFieldDefinitions(sql string) ([]string, error) {
	header := createTableHeaderRE.FindStringIndex(sql)
	if header == nil {
		return nil, fmt.Errorf("CREATE TABLE header not found")
	}
	start := -1
	level := 0
	end := -1
	var quote byte
	inLineComment := false
	inBlockComment := false
	escaped := false
	for i := header[1]; i < len(sql); i++ {
		ch := sql[i]
		next := byte(0)
		if i+1 < len(sql) {
			next = sql[i+1]
		}
		if inLineComment {
			if ch == '\n' {
				inLineComment = false
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
					i++
					continue
				}
				quote = 0
			}
			continue
		}

		switch {
		case ch == '-' && next == '-':
			inLineComment = true
			i++
		case ch == '#':
			inLineComment = true
		case ch == '/' && next == '*':
			inBlockComment = true
			i++
		case ch == '\'' || ch == '"' || ch == '`':
			quote = ch
		case ch == '(':
			if start == -1 {
				start = i
			}
			level++
		case ch == ')':
			if start == -1 {
				continue
			}
			level--
			if level == 0 {
				end = i
				i = len(sql)
			}
		}
	}
	if start == -1 {
		return nil, fmt.Errorf("field definitions not found")
	}
	if end == -1 || end <= start {
		return nil, fmt.Errorf("field definitions not found (unmatched parentheses)")
	}

	inner := sql[start+1 : end]

	// Split by top-level commas (ignore commas inside parentheses/quotes).
	defs := splitFieldDefinitions(inner)

	out := make([]string, 0, len(defs))
	for _, d := range defs {
		s := strings.TrimSpace(d)
		if s != "" {
			out = append(out, s)
		}
	}
	return out, nil
}

// splitFieldDefinitions splits a column-definition block by top-level commas.
// It tracks quote state and backslash escapes to avoid splitting inside strings.
func splitFieldDefinitions(body string) []string {
	var defs []string
	var cur strings.Builder
	level := 0
	inSingle := false
	inDouble := false
	inBacktick := false
	escaped := false

	for i := 0; i < len(body); i++ {
		ch := body[i]

		// Handle backslash escaping.
		if ch == '\\' && !escaped {
			escaped = true
			cur.WriteByte(ch)
			continue
		}

		if !escaped {
			// Toggle quote state (only when not already inside the other quote kind).
			if ch == '\'' && !inDouble && !inBacktick {
				inSingle = !inSingle
				cur.WriteByte(ch)
				continue
			}
			if ch == '"' && !inSingle && !inBacktick {
				inDouble = !inDouble
				cur.WriteByte(ch)
				continue
			}
			if ch == '`' && !inSingle && !inDouble {
				inBacktick = !inBacktick
				cur.WriteByte(ch)
				continue
			}
		}

		// When not inside quotes, update parentheses nesting.
		if !inSingle && !inDouble && !inBacktick {
			if ch == '(' {
				level++
			} else if ch == ')' {
				if level > 0 {
					level--
				}
			}
		}

		// Split on top-level comma.
		if ch == ',' && level == 0 && !inSingle && !inDouble && !inBacktick {
			part := strings.TrimSpace(cur.String())
			if part != "" {
				defs = append(defs, part)
			}
			cur.Reset()
			escaped = false
			continue
		}

		cur.WriteByte(ch)
		escaped = false
	}

	// Append the remaining tail.
	if s := strings.TrimSpace(cur.String()); s != "" {
		defs = append(defs, s)
	}
	return defs
}

func parseField(def string, primaryKeyCols map[string]struct{}) (fieldInfo, error) {
	if isTableConstraint(def) {
		return fieldInfo{}, nil
	}
	// Improved regex to fully capture type description
	re := regexp.MustCompile("^\\s*(?:`([^`]+)`|([A-Za-z0-9_$]+))\\s+(.+)$")
	matches := re.FindStringSubmatch(def)
	if len(matches) < 4 {
		return fieldInfo{}, fmt.Errorf("invalid field definition: %s", def)
	}

	fieldName := matches[1]
	if fieldName == "" {
		fieldName = matches[2]
	}

	// Apply table-level PRIMARY KEY constraints.
	if _, ok := primaryKeyCols[strings.ToLower(fieldName)]; ok {
		// We store this as a gorm tag flag; buildGormTags will render it.
		// Note: for composite primary keys gorm typically needs additional options;
		// this at least marks the columns as primaryKey.
		// (If the column already has inline primary key, this is idempotent.)
		// Defer actual insertion after mapTypeAndTags so we don't lose other tags.
	}
	typeInfo := strings.TrimSpace(matches[3])
	lowerTypeInfo := strings.ToLower(typeInfo)

	// Preserve the original type mapping.
	goType, tags := mapTypeAndTags(typeInfo)
	if _, ok := primaryKeyCols[strings.ToLower(fieldName)]; ok {
		tags["primaryKey"] = "true"
	}

	// Conservative enhancement: detect common constraints and add gorm tags.
	// (Do not change the Go type mapping.)
	if strings.Contains(lowerTypeInfo, "unsigned") {
		tags["unsigned"] = "true"
	}
	if strings.Contains(lowerTypeInfo, "auto_increment") || strings.Contains(lowerTypeInfo, "autoincrement") {
		tags["autoIncrement"] = "true"
	}
	if strings.Contains(lowerTypeInfo, "primary key") || strings.Contains(lowerTypeInfo, "primary_key") {
		tags["primaryKey"] = "true"
	}
	if strings.Contains(lowerTypeInfo, "not null") {
		tags["notNull"] = "true"
	}
	// MySQL generated columns should be query-only in GORM.
	if isGeneratedColumn(lowerTypeInfo) {
		tags["->"] = "true"
	}
	// Best-effort DEFAULT value extraction.
	if idx := strings.Index(strings.ToUpper(typeInfo), "DEFAULT "); idx >= 0 {
		after := strings.TrimSpace(typeInfo[idx+8:])
		if after != "" {
			if strings.HasPrefix(after, "'") || strings.HasPrefix(after, "\"") {
				q := after[0]
				if j := strings.IndexByte(after[1:], q); j >= 0 {
					tags["default"] = after[1 : 1+j]
				} else {
					tags["default"] = strings.Trim(after, "'\"")
				}
			} else {
				parts := strings.Fields(after)
				if len(parts) > 0 {
					tags["default"] = strings.TrimRight(parts[0], ",")
				}
			}
		}
	}

	return fieldInfo{
		name:     toCamelCase(fieldName),
		typeName: goType,
		gormTags: buildGormTags(fieldName, tags),
		jsonTag:  toSnakeCase(fieldName),
	}, nil
}

func isTableConstraint(def string) bool {
	upper := strings.ToUpper(strings.TrimSpace(def))
	for _, prefix := range []string{
		"PRIMARY KEY", "UNIQUE ", "KEY ", "INDEX ", "CONSTRAINT ",
		"FOREIGN KEY", "CHECK ", "FULLTEXT ", "SPATIAL ",
	} {
		if strings.HasPrefix(upper, prefix) {
			return true
		}
	}
	return false
}

func isGeneratedColumn(typeInfo string) bool {
	return strings.Contains(typeInfo, "generated always as") || strings.Contains(typeInfo, " as (")
}

func mapTypeAndTags(sqlType string) (string, map[string]string) {
	tags := make(map[string]string)

	s := strings.ToLower(strings.TrimSpace(sqlType))
	m := regexp.MustCompile(`^(\w+)`).FindStringSubmatch(s)
	baseType := ""
	if len(m) > 1 {
		baseType = m[1]
	}
	unsigned := strings.Contains(s, "unsigned")

	var goType string
	switch baseType {
	case "tinyint":
		// Commonly used as bool(1) in MySQL.
		//if strings.HasPrefix(s, "tinyint(1)") {
		//	goType = "bool"
		//	break
		//}
		if unsigned {
			goType = "uint8"
		} else {
			goType = "int8"
		}
	case "smallint":
		if unsigned {
			goType = "uint16"
		} else {
			goType = "int16"
		}
	case "mediumint":
		// No native 24-bit int in Go; use int32/uint32.
		if unsigned {
			goType = "uint32"
		} else {
			goType = "int32"
		}
	case "int", "integer":
		if unsigned {
			goType = "uint32"
		} else {
			goType = "int32"
		}
	case "bigint":
		if unsigned {
			goType = "uint64"
		} else {
			goType = "int64"
		}
	case "bit":
		// BIT(1) is often used as a boolean.
		if strings.HasPrefix(s, "bit(1)") {
			goType = "bool"
			break
		}
		goType = "[]byte"
	case "float":
		goType = "float32"
	case "double", "real":
		goType = "float64"
	case "decimal", "numeric":
		goType = "decimal.Decimal"

	case "date", "datetime", "timestamp", "time", "year":
		goType = "time.Time"

	case "char", "varchar", "tinytext", "text", "mediumtext", "longtext":
		goType = "string"
	case "enum", "set":
		// Keep as string; application can define strong types if needed.
		goType = "string"
	case "json":
		// Most projects store json as string/[]byte; choose []byte to avoid encoding assumptions.
		goType = "[]byte"

	case "binary", "varbinary", "tinyblob", "blob", "mediumblob", "longblob":
		goType = "[]byte"

	case "bool", "boolean":
		goType = "bool"

	default:
		goType = "string"
	}

	return goType, tags
}

func buildGormTags(fieldName string, tags map[string]string) string {
	parts := []string{"column:" + fieldName}
	keys := make([]string, 0, len(tags))
	for k := range tags {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := tags[k]
		if v == "true" {
			parts = append(parts, k)
		} else {
			parts = append(parts, fmt.Sprintf("%s:%s", k, v))
		}
	}
	return strings.Join(parts, ";")
}

func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i := range parts {
		parts[i] = capitalize(parts[i])
	}
	return strings.Join(parts, "")
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && size == 1 {
		return s
	}
	return string(unicode.ToUpper(r)) + s[size:]
}

func toSnakeCase(s string) string {
	return strings.ToLower(s)
}

func buildStruct(tableName string, fields []fieldInfo) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("type %s struct {\n", toCamelCase(tableName)))

	for _, f := range fields {
		sb.WriteString(fmt.Sprintf("    %-8s %-16s `gorm:\"%s\" json:\"%s\"`\n",
			f.name, f.typeName, f.gormTags, f.jsonTag))
	}

	sb.WriteString("}")
	return sb.String()
}
