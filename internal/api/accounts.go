package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func listAccounts(c *gin.Context) {
	conn, _ := GetNatsCredentialFromContext(c)
	list, err := conn.Monitor.Accounts()
	if err != nil {
		monitorError(c, "accounts", err)
		return
	}
	c.JSON(http.StatusOK, list)
}

func getAccount(c *gin.Context) {
	conn, _ := GetNatsCredentialFromContext(c)
	detail, err := conn.Monitor.Account(c.Param("name"))
	if err != nil {
		monitorError(c, "account", err)
		return
	}
	c.JSON(http.StatusOK, detail)
}
