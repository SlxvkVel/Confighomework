package main

import (
	"os"
	"testing"
)

func TestParseConstants(t *testing.T) {
	parser := NewParser()

	inputFile, err := os.CreateTemp("", "test_constants.conf")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(inputFile.Name())

	outputFile, err := os.CreateTemp("", "test_constants.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(outputFile.Name())

	testData := `
// Тест констант
PORT := 8080;
HOST := @"localhost";
OCTAL := 0o755;
EXPRESSION := ?(+ 10 20);
`

	if _, err := inputFile.WriteString(testData); err != nil {
		t.Fatal(err)
	}
	inputFile.Close()

	if err := parser.Parse(inputFile.Name(), outputFile.Name()); err != nil {
		t.Fatalf("Ошибка парсинга: %v", err)
	}
}

func TestParseArrays(t *testing.T) {
	parser := NewParser()

	inputFile, err := os.CreateTemp("", "test_arrays.conf")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(inputFile.Name())

	outputFile, err := os.CreateTemp("", "test_arrays.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(outputFile.Name())

	testData := `
HOSTS := [ @"host1", @"host2", @"host3" ];
PORTS := [ 80, 443, 8080 ];

[network]
hosts = HOSTS
ports = PORTS
`

	if _, err := inputFile.WriteString(testData); err != nil {
		t.Fatal(err)
	}
	inputFile.Close()

	if err := parser.Parse(inputFile.Name(), outputFile.Name()); err != nil {
		t.Fatalf("Ошибка парсинга: %v", err)
	}
}

func TestParseExpressions(t *testing.T) {
	parser := NewParser()

	inputFile, err := os.CreateTemp("", "test_expressions.conf")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(inputFile.Name())

	outputFile, err := os.CreateTemp("", "test_expressions.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(outputFile.Name())

	testData := `
A := 10;
B := 5;
SUM := ?(+ A B);
DIFF := ?(- A B);
CHAR_CODE := ?(ord "Z");
ABS_VAL := ?(abs -15);

[calculations]
sum = SUM
diff = DIFF
char_code = CHAR_CODE
abs_val = ABS_VAL
`

	if _, err := inputFile.WriteString(testData); err != nil {
		t.Fatal(err)
	}
	inputFile.Close()

	if err := parser.Parse(inputFile.Name(), outputFile.Name()); err != nil {
		t.Fatalf("Ошибка парсинга: %v", err)
	}
}

func TestParseMultilineComments(t *testing.T) {
	parser := NewParser()

	inputFile, err := os.CreateTemp("", "test_comments.conf")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(inputFile.Name())

	outputFile, err := os.CreateTemp("", "test_comments.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(outputFile.Name())

	testData := `
{-
 Этот комментарий
 должен быть полностью
 проигнорирован
-}
VALUE := 42;

[section]
key = VALUE
`

	if _, err := inputFile.WriteString(testData); err != nil {
		t.Fatal(err)
	}
	inputFile.Close()

	if err := parser.Parse(inputFile.Name(), outputFile.Name()); err != nil {
		t.Fatalf("Ошибка парсинга: %v", err)
	}

	outputData, err := os.ReadFile(outputFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	expected := "[section]\nkey = 42\n"
	if string(outputData) != expected {
		t.Errorf("Ожидалось:\n%s\nНо получено:\n%s", expected, string(outputData))
	}
}

func TestParseInvalidName(t *testing.T) {
	parser := NewParser()

	inputFile, err := os.CreateTemp("", "test_invalid.conf")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(inputFile.Name())

	outputFile, err := os.CreateTemp("", "test_invalid.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(outputFile.Name())

	testData := `invalid_name := 42;`  // строчные буквы - должно вызвать ошибку

	if _, err := inputFile.WriteString(testData); err != nil {
		t.Fatal(err)
	}
	inputFile.Close()

	err = parser.Parse(inputFile.Name(), outputFile.Name())
	if err == nil {
		t.Error("Ожидалась ошибка для невалидного имени переменной")
	}
}

func TestParseValidNamesWithUnderscore(t *testing.T) {
	parser := NewParser()

	inputFile, err := os.CreateTemp("", "test_valid_names.conf")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(inputFile.Name())

	outputFile, err := os.CreateTemp("", "test_valid_names.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(outputFile.Name())

	testData := `
// Имена с подчеркиванием должны быть валидны
MY_VAR := 10;
ANOTHER_VAR := 20;
_RESULT := ?(+ MY_VAR ANOTHER_VAR);

[test]
value = _RESULT
`

	if _, err := inputFile.WriteString(testData); err != nil {
		t.Fatal(err)
	}
	inputFile.Close()

	if err := parser.Parse(inputFile.Name(), outputFile.Name()); err != nil {
		t.Fatalf("Ошибка парсинга: %v", err)
	}
}