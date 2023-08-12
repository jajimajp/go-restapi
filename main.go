package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"runtime"

	"github.com/gin-gonic/gin"
)

func main() {
	ConfigRuntime()
	StartWorkers()
	StartGin()
}

// ConfigRuntime sets the number of operating system threads.
func ConfigRuntime() {
	nuCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(nuCPU)
	fmt.Printf("Running with %d CPUs\n", nuCPU)
}

// StartWorkers start starsWorker by goroutine.
func StartWorkers() {
	go statsWorker()
}

// StartGin starts gin web server with setting router.
func StartGin() {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(rateLimit, gin.Recovery())

	router.POST("/signup", postSignup)
	router.GET("/users/:user_id", getUsersById)
	router.PATCH("/users/:user_id", patchUsersById)
	router.POST("/close", postClose)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := router.Run(":" + port); err != nil {
		log.Panicf("error: %s", err)
	}
}

type user struct {
	ID       string `json:"user_id"`
	Password string `json:"password"`
	Nickname string `json:"nickname"`
	Comment  string `json:"comment"`
}

var users = []user{
	{
		ID:       "TaroYamada",
		Password: "PaSSwd4TY",
		Nickname: "たろー",
		Comment:  "僕は元気です",
	},
}

func main() {
	router := gin.Default()

	router.POST("/signup", postSignup)
	router.GET("/users/:user_id", getUsersById)
	router.PATCH("/users/:user_id", patchUsersById)
	router.POST("/close", postClose)

	router.Run("localhost:8080")
}

// アカウントの新規作成
func postSignup(c *gin.Context) {
	var newUser user
	c.Bind(&newUser)
	// TODO: id, pass 以外の入力を受けないようにする

	// バリデーション: user_id と password の入力が必須
	if newUser.ID == "" || newUser.Password == "" {
		var errMsg string
		if newUser.ID == "" && newUser.Password == "" {
			errMsg = "required user_id and password"
		} else if newUser.ID == "" {
			errMsg = "required user_id"
		} else /* newUser.Password == "" */ {
			errMsg = "required password"
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Account creation failed",
			"cause":   errMsg,
		})
		return
	}

	// user_id のバリデーション: 6~20文字、半角英数字
	if len(newUser.ID) < 6 || 20 < len(newUser.ID) {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Account creation failed",
			"cause":   "user_id should have length 6~20",
		})
		return
	}
	match, _ := regexp.MatchString("^[a-zA-Z0-9]+$", newUser.ID)
	if !match {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Account creation failed",
			"cause":   "invalid user_id pattern",
		})
		return
	}

	// password のバリデーション: 8~20文字、半角英数字記号(空白と制御コードを除くASCII文字)
	if len(newUser.ID) < 6 || 20 < len(newUser.ID) {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Account creation failed",
			"cause":   "password should have length 6~20",
		})
		return
	}
	match, _ = regexp.MatchString("^[!-~]+$", newUser.ID)
	if !match {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Account creation failed",
			"cause":   "invalid password pattern",
		})
		return
	}

	fmt.Printf("id: %s; pass: %s", newUser.ID, newUser.Password)
	// 既に同じuser_idを持つアカウントが存在している場合
	for _, u := range users {
		if u.ID == newUser.ID {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "Account creation failed",
				"cause":   "already same user_id is used",
			})
			return
		}
	}

	users = append(users, newUser)
	c.IndentedJSON(http.StatusCreated, gin.H{
		"message": "Account successfully created",
		"user":    newUser,
	})
}

// 指定user_idのユーザ情報を返す
func getUsersById(c *gin.Context) {
	user_id := c.Param("user_id")

	// Basic Authentication credentials の取得
	basic_id, password, hasAuth := c.Request.BasicAuth()

	authenticated := false
	if hasAuth {
		for _, u := range users {
			if u.ID == basic_id {
				if u.Password == password {
					authenticated = true
					break
				}
			}
		}
	}
	if !authenticated {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Authentication Failed"})
		return
	}

	for _, u := range users {
		if u.ID == user_id {
			// TODO: nickname とcommentの空欄処理
			c.JSON(http.StatusOK, gin.H{
				"message": "User details by user_id",
				"user":    u,
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{
		"message": "No User found",
	})

}

// 指定user_idのユーザ情報を更新し、更新したユーザ情報を返す
func patchUsersById(c *gin.Context) {
	user_id := c.Param("user_id")

	var jsonMap map[string]string

	c.BindJSON(&jsonMap)

	nickname, existsNickname := jsonMap["nickname"]
	comment, existsComment := jsonMap["comment"]
	_, existsID := jsonMap["user_id"]
	_, existsPassword := jsonMap["password"]

	// 指定user_idのユーザ情報が存在しない場合は失敗
	userFound := false
	for _, u := range users {
		if u.ID == user_id {
			userFound = true
			break
		}
	}
	if !userFound {
		c.JSON(http.StatusNotFound, gin.H{"message": "No User found"})
		return
	}

	// nicknameとcommentが両方とも指定されていない場合は失敗
	if !existsNickname && !existsComment {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "User updation failed",
			"cause":   "required nickname or comment",
		})
		return
	}

	// user_idやpasswordを変更しようとしている場合は失敗
	if existsID || existsPassword {
		c.JSON(http.StatusNotFound, gin.H{
			"message": "User updation failed",
			"cause":   "not updatable user_id and password",
		})
		return
	}

	// Basic Authentication credentials の取得
	basic_id, password, hasAuth := c.Request.BasicAuth()
	authenticated := false
	if hasAuth {
		for _, u := range users {
			if u.ID == basic_id {
				if u.Password == password {
					authenticated = true
					break
				}
			}
		}
	}
	// Authorizationヘッダでの認証が失敗した場合
	if !authenticated {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Authentication Failed"})
		return
	}

	// 認証と異なるIDのユーザを指定した場合は失敗
	if basic_id != user_id {
		c.JSON(http.StatusForbidden, gin.H{"message": "No Permission for Update"})
		return
	}

	for i, u := range users {
		if u.ID == user_id {
			users[i].Nickname = nickname
			users[i].Comment = comment
			// TODO: recipeの形式
			c.JSON(http.StatusOK, gin.H{
				"message": "User successfully updated",
				"recipe":  users[i],
			})
			return
		}
	}
}

// アカウントの削除
func postClose(c *gin.Context) {
	// Basic Authentication credentials の取得
	basic_id, password, hasAuth := c.Request.BasicAuth()
	authenticated := false
	var user_idx int
	if hasAuth {
		for i, u := range users {
			if u.ID == basic_id {
				if u.Password == password {
					authenticated = true
					user_idx = i
					break
				}
			}
		}
	}
	// Authorizationヘッダでの認証が失敗した場合
	if !authenticated {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Authentication Failed"})
		return
	}

	// 該当ユーザの削除
	users[user_idx] = users[len(users)-1]
	users = users[:len(users)-1]
	c.JSON(http.StatusOK, gin.H{"message": "Account and user successfully removed"})
}
