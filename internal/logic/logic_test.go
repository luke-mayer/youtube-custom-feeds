package logic

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestLogin(t *testing.T) {
	s, err := GetState()
	if err != nil {
		t.Fatalf("Error initalizing state: %s\n", err)
	}

	testUID := "test-uid"

	err = registerUser(s, testUID)
	if err != nil {
		t.Fatalf("Error adding test user to database: %s\n", err)
	}

	defer deleteUser(s, testUID)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(""))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("isTest", "true")
	c.Set("testUID", testUID)

	expectedMessage := "{\"message\":\"Successfully Logged In\"}\n"

	if assert.NoError(t, s.Login(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, expectedMessage, rec.Body.String())
	}
}
