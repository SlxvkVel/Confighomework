package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	ErrInvalidSyntax     = fmt.Errorf("неверный синтаксис")
	ErrUndefinedVariable = fmt.Errorf("неопределенная переменная")
	ErrInvalidExpression = fmt.Errorf("неверное выражение")
)

type Parser struct {
	variables map[string]interface{}
}

// NewParser создает новый экземпляр Parser.
func NewParser() *Parser {
	return &Parser{
		variables: make(map[string]interface{}),
	}
}

// Parse читает входной файл и генерирует выходной TOML.
func (p *Parser) Parse(inputPath string, outputPath string) error {
	content, err := os.ReadFile(inputPath)
	if err != nil {
		return err
	}

	// Удаляем многострочные комментарии
	content = p.removeMultilineComments(content)

	// Обрабатываем построчно
	lines := strings.Split(string(content), "\n")
	var output strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Игнорируем однострочные комментарии
		if strings.HasPrefix(line, "//") {
			continue
		}

		// Обработка объявления константы
		if strings.Contains(line, ":=") && strings.HasSuffix(line, ";") {
			if err := p.handleConstant(line); err != nil {
				return err
			}
			continue
		}

		// Обработка массива
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			// Проверяем, это секция TOML или массив значений
			inner := strings.TrimPrefix(line, "[")
			inner = strings.TrimSuffix(inner, "]")
			inner = strings.TrimSpace(inner)

			// Если это массив значений (содержит запятые)
			if strings.Contains(inner, ",") && !strings.Contains(inner, "=") {
				values := strings.Split(inner, ",")
				array := make([]string, 0)
				for _, v := range values {
					v = strings.TrimSpace(v)
					if v != "" {
						resolved, err := p.resolveValue(v)
						if err != nil {
							return err
						}
						array = append(array, fmt.Sprintf("%v", resolved))
					}
				}
				// Сохраняем массив с уникальным именем
				arrayName := fmt.Sprintf("_array_%d", len(p.variables))
				p.variables[arrayName] = array
			} else {
				output.WriteString(fmt.Sprintf("[%s]\n", inner))
			}
			continue
		}

		// Обработка пар ключ=значение
		if strings.Contains(line, "=") && !strings.Contains(line, ":=") {
			if err := p.handleKeyValue(line, &output); err != nil {
				return err
			}
			continue
		}
	}

	// Запись в выходной файл
	return os.WriteFile(outputPath, []byte(output.String()), 0644)
}

// removeMultilineComments удаляет многострочные комментарии { ... }
func (p *Parser) removeMultilineComments(content []byte) []byte {
	re := regexp.MustCompile(`\{-[\s\S]*?-\}`)
	return re.ReplaceAll(content, []byte{})
}

// handleConstant обрабатывает объявление константы
func (p *Parser) handleConstant(line string) error {
	line = strings.TrimSuffix(line, ";")
	parts := strings.Split(line, ":=")
	if len(parts) != 2 {
		return ErrInvalidSyntax
	}
	name := strings.TrimSpace(parts[0])
	valueStr := strings.TrimSpace(parts[1])

	// Проверяем валидность имени (заглавные буквы и подчеркивания)
	if !p.isValidName(name) {
		return fmt.Errorf("неверное имя переменной '%s': должны быть только заглавные буквы и подчеркивания", name)
	}

	// Обработка константных выражений
	if strings.HasPrefix(valueStr, "?(") && strings.HasSuffix(valueStr, ")") {
		expr := strings.TrimPrefix(valueStr, "?(")
		expr = strings.TrimSuffix(expr, ")")
		value, err := p.evaluateExpression(expr)
		if err != nil {
			return err
		}
		p.variables[name] = value
		return nil
	}

	// Обработка массивов
	if strings.HasPrefix(valueStr, "[") && strings.HasSuffix(valueStr, "]") {
		arrayStr := strings.TrimPrefix(valueStr, "[")
		arrayStr = strings.TrimSuffix(arrayStr, "]")
		values := strings.Split(arrayStr, ",")
		array := make([]string, 0)
		for _, v := range values {
			v = strings.TrimSpace(v)
			if v != "" {
				resolved, err := p.resolveValue(v)
				if err != nil {
					return err
				}
				array = append(array, fmt.Sprintf("%v", resolved))
			}
		}
		p.variables[name] = array
		return nil
	}

	// Обработка строк
	if strings.HasPrefix(valueStr, `@"`) && strings.HasSuffix(valueStr, `"`) {
		value := strings.TrimPrefix(valueStr, `@"`)
		value = strings.TrimSuffix(value, `"`)
		p.variables[name] = value
		return nil
	}

	// Обработка чисел (включая восьмеричные)
	if p.isNumber(valueStr) {
		value, err := p.parseNumber(valueStr)
		if err != nil {
			return err
		}
		p.variables[name] = value
		return nil
	}

	// Обработка ссылок на другие переменные
	if val, exists := p.variables[valueStr]; exists {
		p.variables[name] = val
		return nil
	}

	// Если это просто строка
	p.variables[name] = valueStr
	return nil
}

// handleKeyValue обрабатывает пары ключ=значение
func (p *Parser) handleKeyValue(line string, output *strings.Builder) error {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return ErrInvalidSyntax
	}

	key := strings.TrimSpace(parts[0])
	valueStr := strings.TrimSpace(parts[1])

	value, err := p.resolveValue(valueStr)
	if err != nil {
		return err
	}

	// Форматирование значения для TOML
	var formattedValue string
	switch v := value.(type) {
	case string:
		// Проверяем, является ли строка логическим значением
		if v == "true" || v == "false" {
			formattedValue = v
		} else {
			formattedValue = fmt.Sprintf(`"%s"`, v)
		}
	case int:
		formattedValue = fmt.Sprintf("%d", v)
	case []string:
		formattedValue = "["
		for i, item := range v {
			if i > 0 {
				formattedValue += ", "
			}
			// Проверяем, нужно ли добавлять кавычки
			if _, err := strconv.Atoi(item); err != nil && item != "true" && item != "false" {
				formattedValue += fmt.Sprintf(`"%s"`, item)
			} else {
				formattedValue += item
			}
		}
		formattedValue += "]"
	default:
		formattedValue = fmt.Sprintf("%v", v)
	}

	output.WriteString(fmt.Sprintf("%s = %s\n", key, formattedValue))
	return nil
}

// resolveValue разрешает значение, включая переменные и выражения
func (p *Parser) resolveValue(valueStr string) (interface{}, error) {
	// Если это переменная
	if val, exists := p.variables[valueStr]; exists {
		return val, nil
	}

	// Если это число
	if p.isNumber(valueStr) {
		return p.parseNumber(valueStr)
	}

	// Если это строка в кавычках
	if strings.HasPrefix(valueStr, `@"`) && strings.HasSuffix(valueStr, `"`) {
		value := strings.TrimPrefix(valueStr, `@"`)
		value = strings.TrimSuffix(value, `"`)
		return value, nil
	}

	// Если это просто строка
	return valueStr, nil
}

// evaluateExpression вычисляет константное выражение
func (p *Parser) evaluateExpression(expr string) (int, error) {
	tokens := strings.Fields(expr)
	if len(tokens) < 2 {
		return 0, ErrInvalidExpression
	}

	switch tokens[0] {
	case "+":
		if len(tokens) != 3 {
			return 0, ErrInvalidExpression
		}
		a, err := p.getNumberValue(tokens[1])
		if err != nil {
			return 0, err
		}
		b, err := p.getNumberValue(tokens[2])
		if err != nil {
			return 0, err
		}
		return a + b, nil

	case "-":
		if len(tokens) != 3 {
			return 0, ErrInvalidExpression
		}
		a, err := p.getNumberValue(tokens[1])
		if err != nil {
			return 0, err
		}
		b, err := p.getNumberValue(tokens[2])
		if err != nil {
			return 0, err
		}
		return a - b, nil

	case "ord":
		if len(tokens) != 2 {
			return 0, ErrInvalidExpression
		}
		str, err := p.getStringValue(tokens[1])
		if err != nil {
			return 0, err
		}
		if len(str) == 0 {
			return 0, fmt.Errorf("ord: строка не может быть пустой")
		}
		if strings.HasPrefix(str, `"`) && strings.HasSuffix(str, `"`) {
			str = strings.TrimPrefix(str, `"`)
			str = strings.TrimSuffix(str, `"`)
		}
		if len(str) == 0 {
			return 0, fmt.Errorf("ord: строка не может быть пустой")
		}
		return int(str[0]), nil

	case "abs":
		if len(tokens) != 2 {
			return 0, ErrInvalidExpression
		}
		val, err := p.getNumberValue(tokens[1])
		if err != nil {
			return 0, err
		}
		if val < 0 {
			return -val, nil
		}
		return val, nil

	default:
		return 0, fmt.Errorf("неизвестная операция: %s", tokens[0])
	}
}

// getNumberValue получает числовое значение из токена
func (p *Parser) getNumberValue(token string) (int, error) {
	// Если это переменная
	if val, exists := p.variables[token]; exists {
		switch v := val.(type) {
		case int:
			return v, nil
		case string:
			return 0, fmt.Errorf("переменная %s не является числом", token)
		default:
			return 0, fmt.Errorf("неподдерживаемый тип для переменной %s", token)
		}
	}

	// Если это число
	return p.parseNumber(token)
}

// getStringValue получает строковое значение из токена
func (p *Parser) getStringValue(token string) (string, error) {
	// Если это переменная
	if val, exists := p.variables[token]; exists {
		switch v := val.(type) {
		case string:
			return v, nil
		case int:
			return "", fmt.Errorf("переменная %s не является строкой", token)
		default:
			return "", fmt.Errorf("неподдерживаемый тип для переменной %s", token)
		}
	}

	// Если это строка в кавычках
	if strings.HasPrefix(token, `@"`) && strings.HasSuffix(token, `"`) {
		value := strings.TrimPrefix(token, `@"`)
		value = strings.TrimSuffix(value, `"`)
		return value, nil
	}

	// Если это просто строка
	return token, nil
}

// isNumber проверяет, является ли строка числом
func (p *Parser) isNumber(s string) bool {
	// Восьмеричные числа
	if strings.HasPrefix(s, "0o") || strings.HasPrefix(s, "0O") {
		_, err := strconv.ParseInt(s[2:], 8, 64)
		return err == nil
	}

	// Десятичные числа
	_, err := strconv.Atoi(s)
	return err == nil
}

// parseNumber парсит число из строки
func (p *Parser) parseNumber(s string) (int, error) {
	// Восьмеричные числа
	if strings.HasPrefix(s, "0o") || strings.HasPrefix(s, "0O") {
		val, err := strconv.ParseInt(s[2:], 8, 64)
		return int(val), err
	}

	// Десятичные числа
	val, err := strconv.Atoi(s)
	return val, err
}

// isValidName проверяет валидность имени (заглавные буквы и подчеркивания)
func (p *Parser) isValidName(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !(r >= 'A' && r <= 'Z') && r != '_' {
			return false
		}
	}
	return true
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Использование: go run main.go <input.conf> <output.toml>")
		return
	}

	inputPath := os.Args[1]
	outputPath := os.Args[2]

	parser := NewParser()
	if err := parser.Parse(inputPath, outputPath); err != nil {
		fmt.Printf("Ошибка: %v\n", err)
		return
	}

	fmt.Println("Парсинг завершен успешно.")
}