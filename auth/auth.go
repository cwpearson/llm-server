package auth

import (
	"net/http"
	"strings"
	"os"
	"errors"
	"fmt"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"main/userdb"
)

const Userkey = "user"
const registration_secret_env = "LLM_SERVER_REGISTRATION_SECRET"

func Cookie_store_secret() []byte {
	return []byte("secret")
}

func registration_secret() (string, error) {
	value, ok := os.LookupEnv(registration_secret_env)
	if (!ok) {
		return "", errors.New(fmt.Sprintf("could not read registration secret: %s was not in the environment", registration_secret_env))
	} else {
		return value, nil
	}
}

func IsAuthorized(c *gin.Context) bool {
	session := sessions.Default(c)
	user := session.Get(Userkey)
	return user != nil 
}

// AuthRequired is a simple middleware to check the session.
func AuthRequired(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(Userkey)
	if user == nil {
		// Abort the request with the appropriate error code
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	// Continue down the chain to handler etc
	c.Next()
}

// login is a handler that parses a form and checks for specific data.
func Login(db userdb.DB) func(c *gin.Context) {
	return func(c *gin.Context) {
	session := sessions.Default(c)
	username := c.PostForm("username")
	password := c.PostForm("password")

	// Validate form input
	if strings.Trim(username, " ") == "" || strings.Trim(password, " ") == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parameters can't be empty"})
		return
	}

	// check password
	ok, err := db.IsPassword(username, password)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication failed"})
		return
	}

	// Save the username in the session
	session.Set(Userkey, username) // In real world usage you'd set this to the users ID
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Successfully authenticated user"})
}
}

// logout is the handler called for the user to log out.
func Logout(c *gin.Context) {
	session := sessions.Default(c)
	user := session.Get(Userkey)
	if user == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session token"})
		return
	}
	session.Delete(Userkey)
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}

// produce a handler for user registration
func Register(db userdb.DB) func(c *gin.Context) {

	return func(c *gin.Context) {

	// session := sessions.Default(c)
	username := c.PostForm("username")
	password := c.PostForm("password")
	secret := c.PostForm("secret")

	expected, err := registration_secret()
	if (err != nil) {
		c.Status(http.StatusInternalServerError)
		return
	} else if secret != expected {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "incorrect registration secret"})
		return
	}

	// Validate form input
	if strings.Trim(username, " ") == "" || strings.Trim(password, " ") == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parameters can't be empty"})
		return
	}

	_, err = db.Add(username, password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error while adding user"})
		return
	}

	// log the user in

	
	c.JSON(http.StatusOK, gin.H{"message": "Successfully added user"})
}
}