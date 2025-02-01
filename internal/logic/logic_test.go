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
		t.Fatalf("Issue initalizing state: %s\n", err)
	}

	testId := "testId"
	err = registerUser(s, testId)
	if err != nil {
		t.Fatalf("Issue adding test user to database: %s\n", err)
	}

	defer deleteUser(s, testId)

	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(""))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Firebase-ID", testId)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	expectedMessage := "{\"message\":\"Successfully Logged In\"}\n"

	if assert.NoError(t, s.Login(c)) {
		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Equal(t, expectedMessage, rec.Body.String())
	}
}
