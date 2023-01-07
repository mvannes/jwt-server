package jwt

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"path"

	"golang.org/x/crypto/bcrypt"
)

var UserExistsError = errors.New("User already exists")
var UserNotFoundError = errors.New("User not found")

type TwoFactorType string

const (
	TwoFactorDisabled      TwoFactorType = "disabled"
	TwoFactorAuthenticator               = "authenticator"
)

type TwoFactor struct {
	Type                  TwoFactorType `json:"type"`
	OneTimePasswordSecret string        `json:"oneTimePasswordSecret"`
}

type User struct {
	Username      string    `json:"username"`
	Name          string    `json:"name"`
	PasswordHash  string    `json:"passwordHash"`
	TwoFactorInfo TwoFactor `json:"twoFactorInfo"`
}

type UserRepository interface {
	GetUser(username string) (*User, error)
	StoreUser(username string, name string, password string) error
	GetUserList() ([]User, error)
}

type JSONUserRepository struct {
	storageDir string
	fileName   string
}

var jsonUserStorageInstance *JSONUserRepository

func NewJSONUserRepository(storageDir string, fileName string) *JSONUserRepository {
	if jsonUserStorageInstance == nil {
		jsonUserStorageInstance = &JSONUserRepository{storageDir: storageDir, fileName: fileName}
	}
	return jsonUserStorageInstance
}

func (us *JSONUserRepository) GetUser(username string) (*User, error) {
	userList, err := us.GetUserList()
	if nil != err {
		return nil, err
	}

	for _, user := range userList {
		if user.Username == username {
			return &user, nil
		}
	}

	return nil, UserNotFoundError
}

func (us *JSONUserRepository) StoreUser(username string, name string, password string) error {
	if _, err := os.Stat(us.storageDir); os.IsNotExist(err) {
		err = os.MkdirAll(us.storageDir, os.ModePerm)
		if nil != err {
			return err
		}
	}

	userList, err := us.GetUserList()
	if nil != err {
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 6)
	if nil != err {
		return err
	}

	u := User{
		Username:     username,
		Name:         name,
		PasswordHash: string(hashedPassword),
		TwoFactorInfo: TwoFactor{
			Type:                  TwoFactorDisabled,
			OneTimePasswordSecret: "",
		},
	}

	userList = append(userList, u)

	jsonList, err := json.Marshal(userList)
	if nil != err {
		return err
	}

	err = os.WriteFile(path.Join(us.storageDir, us.fileName), jsonList, 0644)
	return err
}

func (us *JSONUserRepository) GetUserList() ([]User, error) {
	fileContent, err := ioutil.ReadFile(path.Join(us.storageDir, us.fileName))
	if nil != err {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	var userList []User // Make sure that this does not return users with their password.
	if len(fileContent) > 0 {
		if err = json.Unmarshal(fileContent, &userList); nil != err {
			return userList, err
		}
	}

	return userList, nil
}
