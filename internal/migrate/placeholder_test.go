package migrate

import (
	"testing"
)

func TestExpandPlaceholdersRequired(t *testing.T) {
	in := "CREATE TABLE {{schema}}.users(id INT);"
	out, err := expandPlaceholders(in, "tenant1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "CREATE TABLE tenant1.users(id INT);" {
		t.Fatalf("mismatch: %s", out)
	}
}

func TestExpandPlaceholdersOptionalWithSchema(t *testing.T) {
	in := "CREATE TABLE {{schema?}}users(id INT);"
	out, err := expandPlaceholders(in, "tenant1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "CREATE TABLE tenant1.users(id INT);" {
		t.Fatalf("mismatch: %s", out)
	}
}

func TestExpandPlaceholdersOptionalNoSchema(t *testing.T) {
	in := "CREATE TABLE {{schema?}}users(id INT);"
	out, err := expandPlaceholders(in, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "CREATE TABLE users(id INT);" {
		t.Fatalf("mismatch: %s", out)
	}
}

func TestExpandPlaceholdersRequiredMissingSchema(t *testing.T) {
	in := "CREATE TABLE {{schema}}.users(id INT);"
	if _, err := expandPlaceholders(in, ""); err == nil {
		t.Fatalf("expected error when schema empty with required placeholder")
	}
}

func TestExpandPlaceholdersMultiple(t *testing.T) {
	in := "INSERT INTO {{schema}}.a SELECT * FROM {{schema}}.b;"
	out, err := expandPlaceholders(in, "tenant1")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if out != "INSERT INTO tenant1.a SELECT * FROM tenant1.b;" {
		t.Fatalf("mismatch: %s", out)
	}
}

func TestExpandPlaceholdersEscapedRequired(t *testing.T) {
	in := "CREATE TABLE \\{{schema}}.users(id INT);" // escaped token should remain literal
	out, err := expandPlaceholders(in, "tenant1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "CREATE TABLE {{schema}}.users(id INT);" {
		t.Fatalf("mismatch escaped required: %s", out)
	}
}

func TestExpandPlaceholdersEscapedOptional(t *testing.T) {
	in := "CREATE TABLE \\{{schema?}}users(id INT);"
	out, err := expandPlaceholders(in, "tenant1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "CREATE TABLE {{schema?}}users(id INT);" {
		t.Fatalf("mismatch escaped optional with schema: %s", out)
	}
	out2, err := expandPlaceholders(in, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out2 != "CREATE TABLE {{schema?}}users(id INT);" {
		t.Fatalf("mismatch escaped optional without schema: %s", out2)
	}
}
