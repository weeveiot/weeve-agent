package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/dao"
	"gitlab.com/weeve/edge-server/edge-pipeline-service/internal/model"
	"golang.org/x/crypto/bcrypt"
)

// Login using user name and password
// @Summary Login
// @Description Login using user name and password
// @Tags login
// @Accept  json
// @Produce  json
// @Success 200
// @Failure 400
// @Router /login [post]
// @Param data body model.User true "Email & Password required"
func Login(w http.ResponseWriter, r *http.Request) {
	user := &model.User{}
	err := json.NewDecoder(r.Body).Decode(user)
	if err != nil {
		var resp = map[string]interface{}{"status": false, "message": "Invalid request"}
		json.NewEncoder(w).Encode(resp)
		return
	}
	resp := FindOne(user.Email, user.Password)
	json.NewEncoder(w).Encode(resp)
}

func FindOne(email, password string) map[string]interface{} {
	user := &model.User{}

	// if err := db.Where("Email = ?", email).First(user).Error; err != nil {
	// 	var resp = map[string]interface{}{"status": false, "message": "Email address not found"}
	// 	return resp
	// }

	// errf := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	// if errf != nil && errf == bcrypt.ErrMismatchedHashAndPassword { //Password does not match!
	// 	var resp = map[string]interface{}{"status": false, "message": "Invalid login credentials. Please try again"}
	// 	return resp
	// }
	fmt.Printf("%s %s", email, password)
	fmt.Printf("%+v\n", user)

	if email != "managerapi@weeve.network" {
		var resp = map[string]interface{}{"status": false, "message": "Invalid login credentials. Please try again"}
		return resp
	}

	if password != "Weeve@01" {
		var resp = map[string]interface{}{"status": false, "message": "Invalid login credentials. Please try again"}
		return resp
	}

	expiresAt := time.Now().Add(time.Minute * 100000).Unix()
	tk := &model.Token{
		UserID: 1234,
		Name:   "Test",
		Email:  email,
		StandardClaims: &jwt.StandardClaims{
			ExpiresAt: expiresAt,
		},
	}

	// tk := &model.Token{
	// 	UserID: user.ID,
	// 	Name:   user.Name,
	// 	Email:  user.Email,
	// 	StandardClaims: &jwt.StandardClaims{
	// 		ExpiresAt: expiresAt,
	// 	},
	// }

	token := jwt.NewWithClaims(jwt.GetSigningMethod("HS256"), tk)

	tokenString, error := token.SignedString([]byte("secret"))
	if error != nil {
		fmt.Println(error)
	}

	var resp = map[string]interface{}{"status": false, "message": "logged in"}
	resp["token"] = tokenString //Store the token in the response
	resp["user"] = user
	return resp
}

func CreateUser(w http.ResponseWriter, r *http.Request) {

	user := &model.User{}
	json.NewDecoder(r.Body).Decode(user)

	pass, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println(err)
		err := model.ErrorResponse{
			Err: "Password Encryption  failed",
		}
		json.NewEncoder(w).Encode(err)
	}

	user.Password = string(pass)

	var inInterface map[string]interface{}
	inrec, _ := json.Marshal(user)
	json.Unmarshal(inrec, &inInterface)

	createdUser := dao.SaveData("Users", inInterface)
	// var errMessage = createdUser.Error

	// if createdUser.Error != nil {
	// 	fmt.Println(errMessage)
	// }
	json.NewEncoder(w).Encode(createdUser)
}
