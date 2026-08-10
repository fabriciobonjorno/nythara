package security

import "testing"

func TestPasswordHashAndVerification(t *testing.T) {
	hash, err := HashPassword("corvo-no-eclipse-2026")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "corvo-no-eclipse-2026" {
		t.Fatal("senha foi armazenada em texto puro")
	}
	if !VerifyPassword(hash, "corvo-no-eclipse-2026") {
		t.Fatal("senha correta foi rejeitada")
	}
	if VerifyPassword(hash, "senha-incorreta") {
		t.Fatal("senha incorreta foi aceita")
	}
}

func TestTokenHashDoesNotExposeToken(t *testing.T) {
	token, hash, err := NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if token == "" || len(hash) != 32 {
		t.Fatalf("token inválido: token=%q hash=%d bytes", token, len(hash))
	}
	if string(hash) == token {
		t.Fatal("hash do token expõe o token")
	}
}
