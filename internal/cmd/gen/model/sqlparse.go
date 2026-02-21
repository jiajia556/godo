package model

import (
	"fmt"
	"regexp"
	"strings"
)

type fieldInfo struct {
	name     string
	typeName string
	gormTags string
	jsonTag  string
}

// GenerateStruct generates Go struct definition from SQL create table statement
func GenerateModelStruct(sql string) (string, string, error) {
	tableName, fields, err := parseSQL(sql)
	if err != nil {
		return "", "", err
	}

	return buildStruct(tableName, fields), toCamelCase(tableName), nil
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

	var fields []fieldInfo
	for _, def := range fieldDefinitions {
		fi, err := parseField(def)
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

func extractTableName(sql string) (string, error) {
	re := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+[\x60]?(\w+)[\x60]?`)
	matches := re.FindStringSubmatch(sql)
	if len(matches) < 2 {
		return "", fmt.Errorf("table name not found")
	}
	return toCamelCase(matches[1]), nil
}

// extractFieldDefinitions extracts column definitions from a CREATE TABLE statement.
// It uses a simple state machine (paren nesting + quote tracking) so commas in
// types, comments, or indexes won't break the split.
func extractFieldDefinitions(sql string) ([]string, error) {
	// Find the first "(" that begins the column definition block.
	start := strings.Index(sql, "(")
	if start < 0 {
		return nil, fmt.Errorf("field definitions not found")
	}

	// Find the matching closing ")" (taking nested parentheses into account).
	level := 0
	end := -1
	for i := start; i < len(sql); i++ {
		ch := sql[i]
		if ch == '(' {
			level++
		} else if ch == ')' {
			level--
			if level == 0 {
				end = i
				break
			}
		}
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
			if ch == '\'' && !inDouble {
				inSingle = !inSingle
				cur.WriteByte(ch)
				continue
			}
			if ch == '"' && !inSingle {
				inDouble = !inDouble
				cur.WriteByte(ch)
				continue
			}
		}

		// When not inside quotes, update parentheses nesting.
		if !inSingle && !inDouble {
			if ch == '(' {
				level++
			} else if ch == ')' {
				if level > 0 {
					level--
				}
			}
		}

		// Split on top-level comma.
		if ch == ',' && level == 0 && !inSingle && !inDouble {
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

func parseField(def string) (fieldInfo, error) {
	// Improved regex to fully capture type description
	re := regexp.MustCompile("[\x60]?(\\w+)[\x60]?\\s+(.+)")
	matches := re.FindStringSubmatch(def)
	if len(matches) < 3 {
		return fieldInfo{}, fmt.Errorf("invalid field definition: %s", def)
	}

	fieldName := matches[1]
	// Keep legacy behavior: only generate fields for columns starting with a-z.
	if len(fieldName) == 0 || []byte(fieldName)[0] < 'a' || []byte(fieldName)[0] > 'z' {
		return fieldInfo{}, nil
	}
	typeInfo := strings.ToLower(strings.TrimSpace(matches[2]))

	// Preserve the original type mapping.
	goType, tags := mapTypeAndTags(typeInfo)

	// Conservative enhancement: detect common constraints and add gorm tags.
	// (Do not change the Go type mapping.)
	if strings.Contains(typeInfo, "unsigned") {
		tags["unsigned"] = "true"
	}
	if strings.Contains(typeInfo, "auto_increment") || strings.Contains(typeInfo, "autoincrement") {
		tags["autoIncrement"] = "true"
	}
	if strings.Contains(typeInfo, "primary key") || strings.Contains(typeInfo, "primary_key") {
		tags["primaryKey"] = "true"
	}
	if strings.Contains(typeInfo, "not null") {
		tags["notNull"] = "true"
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

func mapTypeAndTags(sqlType string) (string, map[string]string) {
	tags := make(map[string]string)
	baseType := regexp.MustCompile(`^(\w+)(?:\(.*?\))?`).FindString(sqlType)
	baseType = strings.ToLower(baseType)
	unsigned := strings.Contains(sqlType, "unsigned")

	var goType string
	switch {
	case strings.HasPrefix(baseType, "tinyint"):
		if unsigned {
			goType = "uint8"
		} else {
			goType = "int8"
		}
	case strings.HasPrefix(baseType, "int") || strings.HasPrefix(baseType, "bigint"):
		if unsigned {
			goType = "uint64"
		} else {
			goType = "int64"
		}
	case strings.HasPrefix(baseType, "decimal"):
		goType = "decimal.Decimal"
	case strings.Contains(baseType, "datetime") || strings.Contains(baseType, "timestamp"):
		goType = "mytime.DateTime"
	case strings.HasPrefix(baseType, "varchar"), strings.HasPrefix(baseType, "text"):
		goType = "string"
	case strings.HasPrefix(baseType, "boolean"):
		goType = "bool"
	default:
		goType = "string"
	}

	return goType, tags
}

func buildGormTags(fieldName string, tags map[string]string) string {
	parts := []string{"column:" + fieldName}
	for k, v := range tags {
		if v == "true" {
			parts = append(parts, k)
		} else {
			parts = append(parts, fmt.Sprintf("%s:%s", k, v))
		}
	}
	return strings.Join(parts, ";")
}

func isSpecialType(t string) bool {
	return strings.Contains(t, ".") || t == "string"
}

func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i := range parts {
		parts[i] = strings.Title(parts[i])
	}
	return strings.Join(parts, "")
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
