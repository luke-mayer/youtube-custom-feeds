package logic

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestLogin(t *testing.T) {
	type Expected struct {
		StatusCode int
		Message    string
	}
	tests := []struct {
		TestUID  string
		Expected Expected
	}{
		{"new-UID", Expected{
			http.StatusCreated,
			"{\"message\":\"Successfully registered new user\"}\n",
		}},
		{"new-UID", Expected{
			http.StatusOK,
			"{\"message\":\"Successfully Logged In\"}\n",
		}},
	}

	dockerized, err := strconv.ParseBool(os.Getenv("DOCKERIZED"))
	if !dockerized || err != nil {
		err := godotenv.Load("../../.env")
		if err != nil {
			t.Fatalf("Error injecting environment variables: %s\n", err)
		}
	}

	s, err := GetState()
	if err != nil {
		t.Fatalf("Error initalizing state: %s\n", err)
	}

	var testUID string
	for _, test := range tests {

		testUID := test.TestUID

		e := echo.New()
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(""))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set("isTest", "true")
		c.Set("testUID", testUID)

		expectedStatusCode := test.Expected.StatusCode
		expectedMessage := test.Expected.Message

		if assert.NoError(t, s.Login(c)) {
			assert.Equal(t, expectedStatusCode, rec.Code)
			assert.Equal(t, expectedMessage, rec.Body.String())
		}
	}

	err = deleteUser(s, testUID)
	if err != nil {
		log.Print(err)
	}
}
